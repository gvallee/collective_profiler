/*************************************************************************
 * Copyright (c) 2019-2010, Mellanox Technologies, Inc. All rights reserved.
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <mpi.h>

#include "alltoallv_profiler.h"
#include "logger.h"
#include "grouping.h"
#include "pattern.h"

static avSRCountNode_t *head = NULL;
static avTimingsNode_t *op_timing_exec_head = NULL;
static avTimingsNode_t *op_timing_exec_tail = NULL;
static avPattern_t *spatterns = NULL;
static avPattern_t *rpatterns = NULL;
static avCallPattern_t *call_patterns = NULL;

static int world_size = -1;
static int myrank = -1;
static int avCalls = 0;		  // Total number of alltoallv calls that we went through (indexed on 0, not 1)
static int avCallsLogged = 0; // Total number of alltoallv calls for which we gathered data
static int avCallStart = -1;  // Number of alltoallv call during which we started to gather data
//char myhostname[HOSTNAME_LEN];
//char *hostnames = NULL; // Only used by rank0

static int _num_call_start_profiling = NUM_CALL_START_PROFILING;
static int _limit_av_calls = DEFAULT_LIMIT_ALLTOALLV_CALLS;

// Buffers used to store data through all alltoallv calls
int *sbuf = NULL;
int *rbuf = NULL;
double *op_exec_times = NULL;
double *late_arrival_timings = NULL;

static logger_t *logger = NULL;

/* FORTRAN BINDINGS */
extern int mpi_fortran_in_place_;
#define OMPI_IS_FORTRAN_IN_PLACE(addr) \
	(addr == (void *)&mpi_fortran_in_place_)
extern int mpi_fortran_bottom_;
#define OMPI_IS_FORTRAN_BOTTOM(addr) \
	(addr == (void *)&mpi_fortran_bottom_)
#define OMPI_INT_2_FINT(a) a
#define OMPI_FINT_2_INT(a) a
#define OMPI_F2C_IN_PLACE(addr) (OMPI_IS_FORTRAN_IN_PLACE(addr) ? MPI_IN_PLACE : (addr))
#define OMPI_F2C_BOTTOM(addr) (OMPI_IS_FORTRAN_BOTTOM(addr) ? MPI_BOTTOM : (addr))

static int *lookupRankSendCounters(avSRCountNode_t *call_data, int rank)
{
	return lookup_rank_counters(call_data->send_data_size, call_data->send_data, rank);
}

static int *lookupRankRecvCounters(avSRCountNode_t *call_data, int rank)
{
	return lookup_rank_counters(call_data->recv_data_size, call_data->recv_data, rank);
}

// Compare if two arrays are identical.
static bool same_call_counters(avSRCountNode_t *call_data, int *send_counts, int *recv_counts, int size)
{
	int num = 0;
	int rank, count_num;
	int *_counts;

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Comparing data with existing data...\n", __FILE__, __LINE__);
	DEBUG_ALLTOALLV_PROFILING("[%s:%d] -> Comparing send counts...\n", __FILE__, __LINE__);
	// First compare the send counts
	for (rank = 0; rank < size; rank++)
	{
		int *_counts = lookupRankSendCounters(call_data, rank);
		assert(_counts);
		for (count_num = 0; count_num < size; count_num++)
		{
			if (_counts[count_num] != send_counts[num])
			{
				DEBUG_ALLTOALLV_PROFILING("[%s:%d] Data differs\n", __FILE__, __LINE__);
				return false;
			}
			num++;
		}
	}
	DEBUG_ALLTOALLV_PROFILING("[%s:%d] -> Send counts are the same\n", __FILE__, __LINE__);

	// Then the receive counts
	DEBUG_ALLTOALLV_PROFILING("[%s:%d] -> Comparing recv counts...\n", __FILE__, __LINE__);
	num = 0;
	for (rank = 0; rank < size; rank++)
	{
		int *_counts = lookupRankRecvCounters(call_data, rank);
		for (count_num = 0; count_num < size; count_num++)
		{
			if (_counts[count_num] != recv_counts[num])
			{
				DEBUG_ALLTOALLV_PROFILING("[%s:%d] Data differs\n", __FILE__, __LINE__);
				return false;
			}
			num++;
		}
	}

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Data is the same\n", __FILE__, __LINE__);
	return true;
}

static counts_data_t *lookupCounters(int size, int num, counts_data_t **list, int *count)
{
	int i, j;
	for (i = 0; i < num; i++)
	{
		for (j = 0; j < size; j++)
		{
			if (count[j] != list[i]->counters[j])
			{
				break;
			}
		}

		if (j == size)
		{
			return list[i];
		}
	}

	return NULL;
}

static int extract_patterns_from_counts(int *send_counts, int *recv_counts, int size)
{
	int i, j, num;
	int src_ranks = 0;
	int dst_ranks = 0;
	int send_patterns[size + 1];
	int recv_patterns[size + 1];

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Extracting patterns\n", __FILE__, __LINE__);

	for (i = 0; i < size; i++)
	{
		send_patterns[i] = 0;
	}

	for (i = 0; i < size; i++)
	{
		recv_patterns[i] = 0;
	}

	num = 0;
	for (i = 0; i < size; i++)
	{
		dst_ranks = 0;
		src_ranks = 0;
		for (j = 0; j < size; j++)
		{
			if (send_counts[num] != 0)
			{
				dst_ranks++;
			}
			if (recv_counts[num] != 0)
			{
				src_ranks++;
			}
			num++;
		}
		// We know the current rank sends data to <dst_ranks> ranks
		if (dst_ranks > 0)
		{
			send_patterns[dst_ranks - 1]++;
		}

		// We know the current rank receives data from <src_ranks> ranks
		if (src_ranks > 0)
		{
			recv_patterns[src_ranks - 1]++;
		}
	}

	// From here we know who many ranks send to how many ranks and how many ranks receive from how many rank
	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Handling send patterns\n", __FILE__, __LINE__);
	for (i = 0; i < size; i++)
	{
		if (send_patterns[i] != 0)
		{
			DEBUG_ALLTOALLV_PROFILING("[%s:%d] Add pattern where %d ranks sent data to %d other ranks\n", __FILE__, __LINE__, send_patterns[i], i + 1);
#if COMMSIZE_BASED_PATTERNS
			spatterns = add_pattern_for_size(spatterns, send_patterns[i], i + 1, size);
#else
			spatterns = add_pattern(spatterns, send_patterns[i], i + 1);
#endif // COMMSIZE_BASED_PATTERNS
		}
	}
	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Handling receive patterns\n", __FILE__, __LINE__);
	for (i = 0; i < size; i++)
	{
		if (recv_patterns[i] != 0)
		{
			DEBUG_ALLTOALLV_PROFILING("[%s:%d] Add pattern where %d ranks received data from %d other ranks\n", __FILE__, __LINE__, recv_patterns[i], i + 1);
#if COMMSIZE_BASED_PATTERNS
			rpatterns = add_pattern_for_size(rpatterns, recv_patterns[i], i + 1, size);
#else
			rpatterns = add_pattern(rpatterns, recv_patterns[i], i + 1);
#endif // COMMSIZE_BASED_PATTERNS
		}
	}

	return 0;
}

int extract_call_patterns_from_counts(int callID, int *send_counts, int *recv_counts, int size)
{
	avCallPattern_t *cp = extract_call_patterns(callID, send_counts, recv_counts, size);
	if (call_patterns == NULL)
	{
		call_patterns = cp;
	}
	else
	{
		avCallPattern_t *existing_cp = lookup_call_patterns(call_patterns);
		if (existing_cp == NULL)
		{
			avCallPattern_t *ptr = call_patterns;
			while (ptr->next != NULL)
			{
				ptr = ptr->next;
			}
			ptr->next = cp;
		}
		else
		{
			existing_cp->n_calls++;
			free_patterns(cp->spatterns);
			free_patterns(cp->rpatterns);
			free(cp);
		}
	}

	return 0;
}

static int commit_pattern_from_counts(int callID, int *send_counts, int *recv_counts, int size)
{
#if TRACK_PATTERNS_ON_CALL_BASIS
	return extract_call_patterns_from_counts(callID, send_counts, recv_counts, size);
#else
	return extract_patterns_from_counts(send_counts, recv_counts, size);
#endif
}

static counts_data_t *lookupSendCounters(int *counts, avSRCountNode_t *call_data)
{
	return lookupCounters(call_data->size, call_data->send_data_size, call_data->send_data, counts);
}

static counts_data_t *lookupRecvCounters(int *counts, avSRCountNode_t *call_data)
{
	return lookupCounters(call_data->size, call_data->recv_data_size, call_data->recv_data, counts);
}

static int add_rank_to_counters_data(int rank, counts_data_t *counters_data)
{
	if (counters_data->num_ranks >= counters_data->max_ranks)
	{
		counters_data->max_ranks = counters_data->num_ranks + MAX_TRACKED_RANKS;
		counters_data->ranks = (int *)realloc(counters_data->ranks, counters_data->max_ranks * sizeof(int));
	}

	counters_data->ranks[counters_data->num_ranks] = rank;
	counters_data->num_ranks++;
	return 0;
}

static void delete_counter_data(counts_data_t **data)
{
	if (*data)
	{
		if ((*data)->ranks)
		{
			free((*data)->ranks);
		}
		if ((*data)->counters)
		{
			free((*data)->counters);
		}
		free(*data);
		*data = NULL;
	}
}

static counts_data_t *new_counter_data(int size, int rank, int *counts)
{
	int i;
	counts_data_t *new_data = (counts_data_t *)malloc(sizeof(counts_data_t));
	assert(new_data);
	new_data->counters = (int *)malloc(size * sizeof(int));
	assert(new_data->counters);
	new_data->num_ranks = 0;
	new_data->max_ranks = MAX_TRACKED_RANKS;
	new_data->ranks = (int *)malloc(new_data->max_ranks * sizeof(int));
	assert(new_data->ranks);

	for (i = 0; i < size; i++)
	{
		new_data->counters[i] = counts[i];
	}
	new_data->ranks[new_data->num_ranks] = rank;
	new_data->num_ranks++;

	return new_data;
}

static int add_new_send_counters_to_counters_data(avSRCountNode_t *call_data, int rank, int *counts)
{
	counts_data_t *new_data = new_counter_data(call_data->size, rank, counts);
	call_data->send_data[call_data->send_data_size] = new_data;
	call_data->send_data_size++;

	return 0;
}

static int add_new_recv_counters_to_counters_data(avSRCountNode_t *call_data, int rank, int *counts)
{
	counts_data_t *new_data = new_counter_data(call_data->size, rank, counts);
	call_data->recv_data[call_data->recv_data_size] = new_data;
	call_data->recv_data_size++;

	return 0;
}

static int compareAndSaveSendCounters(int rank, int *counts, avSRCountNode_t *call_data)
{
	counts_data_t *ptr = lookupSendCounters(counts, call_data);
	if (ptr)
	{
		DEBUG_ALLTOALLV_PROFILING("[%s:%d] Add send rank %d to existing count data\n", __FILE__, __LINE__, rank);
		if (add_rank_to_counters_data(rank, ptr))
		{
			fprintf(stderr, "[%s:%d][ERROR] unable to add rank counters (rank: %d)\n", __FILE__, __LINE__, rank);
			return -1;
		}
	}
	else
	{
		DEBUG_ALLTOALLV_PROFILING("[%s:%d] Add send new count data for rank %d\n", __FILE__, __LINE__, rank);
		if (add_new_send_counters_to_counters_data(call_data, rank, counts))
		{
			fprintf(stderr, "[%s:%d][ERROR] unable to add new send counters\n", __FILE__, __LINE__);
			return -1;
		}
	}

	return 0;
}

static int compareAndSaveRecvCounters(int rank, int *counts, avSRCountNode_t *call_data)
{
	counts_data_t *ptr = lookupRecvCounters(counts, call_data);
	if (ptr)
	{
		DEBUG_ALLTOALLV_PROFILING("[%s:%d] Add recv rank %d to existing count data\n", __FILE__, __LINE__, rank);
		if (add_rank_to_counters_data(rank, ptr))
		{
			fprintf(stderr, "[ERROR] unable to add rank counters\n");
			return -1;
		}
	}
	else
	{
		DEBUG_ALLTOALLV_PROFILING("[%s:%d] Add recv new count data for rank %d\n", __FILE__, __LINE__, rank);
		if (add_new_recv_counters_to_counters_data(call_data, rank, counts))
		{
			fprintf(stderr, "[ERROR] unable to add new recv counters\n");
			return -1;
		}
	}

	return 0;
}

// Compare new send count data with existing data.
// If there is a match, increas the counter. Add new data, otherwise.
// recv count was not compared.
static int insert_sendrecv_data(int *sbuf, int *rbuf, int size, int sendtype_size, int recvtype_size)
{
	int i, j, num = 0;
	struct avSRCountNode *newNode = NULL;
	struct avSRCountNode *temp;

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Insert data for a new alltoallv call...\n", __FILE__, __LINE__);

	assert(logger);
#if DEBUG
	assert(logger->f);
#endif

	temp = head;
	while (temp != NULL)
	{
		if (temp->size != size || temp->recvtype_size != recvtype_size || temp->sendtype_size != sendtype_size || !same_call_counters(temp, sbuf, rbuf, size))
		{
			// New data
#if DEBUG
			fprintf(logger->f, "new data: %d\n", size);
#endif
			if (temp->next != NULL)
				temp = temp->next;
			else
				break;
		}
		else
		{
			// Data exist, adding call info to it
			DEBUG_ALLTOALLV_PROFILING("[%s:%d] Data already exists, updating metadata...\n", __FILE__, __LINE__);
			if (temp->count < MAX_TRACKED_CALLS)
			{
				temp->calls[temp->count] = avCalls; // Note: count starts at 1, not 0
			}
			temp->count++;
#if DEBUG
			fprintf(logger->f, "old data: %d --> %d --- %d\n", size, temp->size, temp->count);
#endif
			DEBUG_ALLTOALLV_PROFILING("[%s:%d] Metadata successfully updated\n", __FILE__, __LINE__);
			return 0;
		}
	}

#if DEBUG
	fprintf(logger->f, "no data: %d \n", size);
#endif
	newNode = (struct avSRCountNode *)malloc(sizeof(avSRCountNode_t));
	assert(newNode);

	newNode->size = size;
	newNode->count = 1;
	// We have at most <size> different counts (one per rank) and we just allocate pointers of pointers here, not much space used
	newNode->send_data = (counts_data_t **)malloc(size * sizeof(counts_data_t));
	assert(newNode->send_data);
	newNode->send_data_size = 0;
	newNode->recv_data = (counts_data_t **)malloc(size * sizeof(counts_data_t));
	assert(newNode->recv_data);
	newNode->recv_data_size = 0;

	// We add rank's data one by one so we can compress the data when possible
	num = 0;
	int _rank;

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] handling send counts...\n", __FILE__, __LINE__);
	for (_rank = 0; _rank < size; _rank++)
	{
		if (compareAndSaveSendCounters(_rank, &(sbuf[num * size]), newNode))
		{
			fprintf(stderr, "[%s:%d][ERROR] unable to add send counters\n", __FILE__, __LINE__);
			return -1;
		}
		num++;
	}

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] handling recv counts...\n", __FILE__, __LINE__);
	num = 0;
	for (_rank = 0; _rank < size; _rank++)
	{
		if (compareAndSaveRecvCounters(_rank, &(rbuf[num * size]), newNode))
		{
			fprintf(stderr, "[%s:%d][ERROR] unable to add recv counters\n", __FILE__, __LINE__);
			return -1;
		}
		num++;
	}

	newNode->sendtype_size = sendtype_size;
	newNode->recvtype_size = recvtype_size;
	newNode->calls[0] = avCalls;
	newNode->next = NULL;
#if DEBUG
	fprintf(logger->f, "new entry: %d --> %d --- %d\n", size, newNode->size, newNode->count);
#endif

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Data for the new alltoallv call has %d unique series for send counts and %d for recv counts\n", __FILE__, __LINE__, newNode->recv_data_size, newNode->send_data_size);

	/*
	for (i = 0; i < size; i++)
	{
		for (j = 0; j < size; j++)
		{
			newNode->send_data[j] = sbuf[num];
			newNode->recv_data[j] = rbuf[num];
			num++;
		}
	}
*/

	if (head == NULL)
	{
		head = newNode;
	}
	else
	{
		temp->next = newNode;
	}

	return 0;
}

static void insert_op_exec_times_data(double *timings, double *t_arrivals, int size)
{
	assert(timings);
	struct avTimingsNode *newNode = (struct avTimingsNode *)calloc(1, sizeof(struct avTimingsNode));
	newNode->timings = (double *)malloc(size * sizeof(double));
	newNode->t_arrivals = (double *)malloc(size * sizeof(double));
	assert(newNode);

	newNode->size = size;
	int i;
	for (i = 0; i < size; i++)
	{
		newNode->timings[i] = timings[i];
	}

	for (i = 0; i < size; i++)
	{
		newNode->t_arrivals[i] = t_arrivals[i];
	}

	if (op_timing_exec_head == NULL)
	{
		op_timing_exec_head = newNode;
		op_timing_exec_tail = newNode;
	}
	else
	{
		op_timing_exec_tail->next = newNode;
		op_timing_exec_tail = newNode;
	}
}

static void display_per_host_data(int size)
{
	int i;
	for (i = 0; i < world_size; i++)
	{
	}
}

static void _save_patterns(FILE *fh, avPattern_t *p, char *ctx)
{
	avPattern_t *ptr = p;
	while (ptr != NULL)
	{
#if COMMSIZE_BASED_PATTERNS || TRACK_PATTERNS_ON_CALL_BASIS
		fprintf(fh, "During %d alltoallv calls, %d ranks %s %d other ranks; comm size: %d\n", ptr->n_calls, ptr->n_ranks, ctx, ptr->n_peers, ptr->comm_size);
#else
		fprintf(fh, "During %d alltoallv calls, %d ranks %s %d other ranks\n", ptr->n_calls, ptr->n_ranks, ctx, ptr->n_peers);
#endif // COMMSIZE_BASED_PATTERNS
		ptr = ptr->next;
	}
}

static void save_call_patterns(int uniqueID)
{
	char filename[MAX_PATH_LEN];

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Saving call patterns...\n", __FILE__, __LINE__);

	if (getenv(OUTPUT_DIR_ENVVAR))
	{
		sprintf(filename, "%s/call-patterns-pid%d.txt", getenv(OUTPUT_DIR_ENVVAR), uniqueID);
	}
	else
	{
		sprintf(filename, "call-patterns-pid%d.txt", uniqueID);
	}

	FILE *fh = fopen(filename, "w");
	assert(fh);

	avCallPattern_t *ptr = call_patterns;
	while (ptr != NULL)
	{
		fprintf(fh, "For %d call(s):\n", ptr->n_calls);
		_save_patterns(fh, ptr->spatterns, "sent to");
		_save_patterns(fh, ptr->rpatterns, "recv'd from");
		ptr = ptr->next;
	}
	fclose(fh);
}

static void save_patterns(int uniqueID)
{
	char spatterns_filename[MAX_PATH_LEN];
	char rpatterns_filename[MAX_PATH_LEN];

	DEBUG_ALLTOALLV_PROFILING("[%s:%d] Saving patterns...\n", __FILE__, __LINE__);

	if (getenv(OUTPUT_DIR_ENVVAR))
	{
		sprintf(spatterns_filename, "%s/patterns-send-pid%d.txt", getenv(OUTPUT_DIR_ENVVAR), uniqueID);
		sprintf(rpatterns_filename, "%s/patterns-recv-pid%d.txt", getenv(OUTPUT_DIR_ENVVAR), uniqueID);
	}
	else
	{
		sprintf(spatterns_filename, "patterns-send-pid%d.txt", uniqueID);
		sprintf(rpatterns_filename, "patterns-recv-pid%d.txt", uniqueID);
	}

	FILE *spatterns_fh = fopen(spatterns_filename, "w");
	assert(spatterns_fh);
	FILE *rpatterns_fh = fopen(rpatterns_filename, "w");
	assert(rpatterns_fh);
	avPattern_t *ptr;

	_save_patterns(spatterns_fh, spatterns, "sent to");
	_save_patterns(rpatterns_fh, rpatterns, "recv'd from");

	fclose(spatterns_fh);
	fclose(rpatterns_fh);
}

static void save_counters_for_validation(int uniqueID, int myrank, int avCalls, int size, const int *sendcounts, const int *recvcounts)
{
	char filename[MAX_PATH_LEN];

	if (getenv(OUTPUT_DIR_ENVVAR))
	{
		sprintf(filename, "%s/validation_data-pid%d-rank%d-call%d.txt", getenv(OUTPUT_DIR_ENVVAR), uniqueID, myrank, avCalls);
	}
	else
	{
		sprintf(filename, "validation_data-pid%d-rank%d-call%d.txt", uniqueID, myrank, avCalls);
	}

	FILE *fh = fopen(filename, "w");
	assert(fh);
	int i;
	for (i = 0; i < size; i++)
	{
		fprintf(fh, "%d ", sendcounts[i]);
	}

	fprintf(fh, "\n\n");

	for (i = 0; i < size; i++)
	{
		fprintf(fh, "%d ", recvcounts[i]);
	}

	fclose(fh);
}

int _mpi_init(int *argc, char ***argv)
{
	int ret;
	char buf[200];
	//gethostname(myhostname, HOSTNAME_LEN);

	ret = PMPI_Init(argc, argv);

	MPI_Comm_rank(MPI_COMM_WORLD, &myrank);
	MPI_Comm_size(MPI_COMM_WORLD, &world_size);

	if (myrank == 0)
	{
		logger = logger_init();
	}

	// Allocate buffers reused between alltoallv calls
	// Note the buffer may be used on a communicator that is not comm_world
	// but in any case, it will be smaller or of the same size than comm_world.
	// So we allocate the biggest buffers possible but reuse them during the
	// entire execution of the application.
	sbuf = (int *)malloc(world_size * world_size * (sizeof(int)));
	assert(sbuf);
	rbuf = (int *)malloc(world_size * world_size * (sizeof(int)));
	assert(rbuf);
#if ENABLE_TIMING
	op_exec_times = (double *)malloc(world_size * sizeof(double));
	assert(op_exec_times);
	late_arrival_timings = (double *)malloc(world_size * sizeof(double));
	assert(late_arrival_timings);
#endif

#if ENABLE_VALIDATION
	srand((unsigned)getpid());
#endif

	// Make sure we do not create an articial imbalance between ranks.
	MPI_Barrier(MPI_COMM_WORLD);

	return ret;
}

int MPI_Init(int *argc, char ***argv)
{
	return _mpi_init(argc, argv);
}

int mpi_init_(MPI_Fint *ierr)
{
	int c_ierr;
	int argc = 0;
	char **argv = NULL;

	char *num_call_envvar = getenv(NUM_CALL_START_PROFILING_ENVVAR);
	if (num_call_envvar != NULL)
	{
		_num_call_start_profiling = atoi(num_call_envvar);
	}

	char *limit_a2a_calls = getenv(LIMIT_ALLTOALLV_CALLS_ENVVAR);
	if (limit_a2a_calls != NULL)
	{
		_limit_av_calls = atoi(limit_a2a_calls);
	}

	c_ierr = _mpi_init(&argc, &argv);
	if (NULL != ierr)
		*ierr = OMPI_INT_2_FINT(c_ierr);
}

// During Finalize, it prints all stored data to a file.
int _mpi_finalize()
{
	if (myrank == 0)
	{
#if ENABLE_RAW_DATA || ENABLE_VALIDATION
		DEBUG_ALLTOALLV_PROFILING("[%s:%d] Logging profiling data...\n", __FILE__, __LINE__);
		log_profiling_data(logger, avCalls, avCallStart, avCallsLogged, head, op_timing_exec_head);
		DEBUG_ALLTOALLV_PROFILING("[%s:%d] Logging completed\n", __FILE__, __LINE__);

		// All data has been handled, now we can clean up
		int i;
		while (head != NULL)
		{
			avSRCountNode_t *c_ptr = head->next;

			for (i = 0; i < head->send_data_size; i++)
			{
				delete_counter_data(&(head->send_data[i]));
			}

			for (i = 0; i < head->recv_data_size; i++)
			{
				delete_counter_data(&(head->recv_data[i]));
			}

			free(head->recv_data);
			free(head->send_data);

			free(head);
			head = c_ptr;
		}
#endif // ENABLE_RAW_DATA || ENABLE_VALIDATION

#if ENABLE_PATTERN_DETECTION && !TRACK_PATTERNS_ON_CALL_BASIS
		save_patterns(getpid());
#endif // ENABLE_PATTERN_DETECTION && !TRACK_PATTERNS_ON_CALL_BASIS

#if ENABLE_PATTERN_DETECTION && TRACK_PATTERNS_ON_CALL_BASIS
		save_call_patterns(getpid());
#endif // ENABLE_PATTERN_DETECTION && TRACK_PATTERNS_ON_CALL_BASIS
		while (spatterns != NULL)
		{
			avPattern_t *sp = spatterns->next;
			free(spatterns);
			spatterns = sp;
		}

		while (rpatterns != NULL)
		{
			avPattern_t *rp = rpatterns->next;
			free(rpatterns);
			rpatterns = rp;
		}

		while (op_timing_exec_head != NULL)
		{
			avTimingsNode_t *t_ptr = op_timing_exec_head->next;
			free(op_timing_exec_head->timings);
			free(op_timing_exec_head);
			op_timing_exec_head = t_ptr;
		}
		op_timing_exec_tail = NULL;

#if 0
		fprintf(f, "# Hostnames\n");
                int i;
		for (i = 0; i < world_size; i++)
		{
			char h[HOSTNAME_LEN];
			int offset = HOSTNAME_LEN * i;
                        int j;
			for (j = 0; j < HOSTNAME_LEN; j++)
			{
				h[j] = hostnames[offset + j];
			}
			fprintf(f, "Rank %d: %s\n", i, h);
		}
#endif

		// Free all the memory allocated during MPI_Init() for profiling purposes
		if (rbuf != NULL)
		{
			free(rbuf);
		}
		if (sbuf != NULL)
		{
			free(sbuf);
		}
		if (op_exec_times != NULL)
		{
			free(op_exec_times);
		}
		if (late_arrival_timings != NULL)
		{
			free(late_arrival_timings);
		}
#if 0
		if (hostnames)
		{
			free(hostnames);
		}
#endif

		if (myrank == 0)
		{
			logger_fini(&logger);
		}
	}
	return PMPI_Finalize();
}

int MPI_Finalize()
{
	return _mpi_finalize();
}

void mpi_finalize_(MPI_Fint *ierr)
{
	int c_ierr = _mpi_finalize();
	if (NULL != ierr)
		*ierr = OMPI_INT_2_FINT(c_ierr);
}

int _mpi_alltoallv(const void *sendbuf, const int *sendcounts, const int *sdispls,
				   MPI_Datatype sendtype, void *recvbuf, const int *recvcounts,
				   const int *rdispls, MPI_Datatype recvtype, MPI_Comm comm)
{
	int size;
	int i, j;
	int localrank;
	int ret;
	bool need_profile = true;

	// Check if we need to profile that specific call
	if (avCalls < _num_call_start_profiling)
	{
		need_profile = false;
	}
	else
	{
		if (-1 != _limit_av_calls && avCallsLogged >= _limit_av_calls)
		{
			need_profile = false;
		}
	}

	if (need_profile)
	{

		if (avCallStart == -1)
		{
			avCallStart = avCalls;
		}
		MPI_Comm_rank(comm, &localrank);
		MPI_Comm_size(comm, &size);

#if 0
	if (myrank == 0)
	{
		hostnames = (char *)malloc(size * HOSTNAME_LEN * sizeof(char));
	}
#endif

#if ENABLE_VALIDATION
		// Quite simple: rank 0 broadcast a unique ID (its PID) to other rank so we have a unique way to group files
		// Then we randomly save counters in separate file on a per-rank and per-alltoallv-call basis.
		// A tool is available to compare that data with the counter file we have, using the underlying infrastructure
		// we have to gather stats
		int uniqueID;
		if (myrank == 0)
		{
			uniqueID = getpid();
			MPI_Bcast(&uniqueID, 1, MPI_INT, 0, comm);
		}
		else
		{
			MPI_Bcast(&uniqueID, 1, MPI_INT, 0, comm);
		}

		if (get_remainder(rand(), 100) <= VALIDATION_THRESHOLD)
		{
			save_counters_for_validation(uniqueID, myrank, avCalls, size, sendcounts, recvcounts);
		}
#endif // ENABLE_VALIDATION

#if ENABLE_TIMING
		double t_barrier_start = MPI_Wtime();
		PMPI_Barrier(comm);
		double t_barrier_end = MPI_Wtime();
		double t_start = MPI_Wtime();
#endif // ENABLE_TIMING
		ret = PMPI_Alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
#if ENABLE_TIMING
		double t_end = MPI_Wtime();
		double t_op = t_end - t_start;
		double t_arrival = t_barrier_end - t_barrier_start;
#endif // ENABLE_TIMING

		// Gather a bunch of counters
		MPI_Gather(sendcounts, size, MPI_INT, sbuf, size, MPI_INT, 0, comm);
		MPI_Gather(recvcounts, size, MPI_INT, rbuf, size, MPI_INT, 0, comm);
#if ENABLE_TIMING
		MPI_Gather(&t_op, 1, MPI_DOUBLE, op_exec_times, 1, MPI_DOUBLE, 0, comm);
		MPI_Gather(&t_arrival, 1, MPI_DOUBLE, late_arrival_timings, 1, MPI_DOUBLE, 0, comm);
#endif // ENABLE_TIMING \
	//MPI_Gather(myhostname, HOSTNAME_LEN, MPI_CHAR, hostnames, HOSTNAME_LEN, MPI_CHAR, 0, comm);

		if (myrank == 0)
		{
#if DEBUG
			fprintf(logger->f, "Root: global %d - %d   local %d - %d\n", world_size, myrank, size, localrank);
#endif
#if ENABLE_RAW_DATA || ENABLE_PER_RANK_STATS || ENABLE_VALIDATION
			if (insert_sendrecv_data(sbuf, rbuf, size, sizeof(sendtype), sizeof(recvtype)))
			{
				fprintf(stderr, "[%s:%d][ERROR] unable to insert send/recv counts\n", __FILE__, __LINE__);
				MPI_Abort(MPI_COMM_WORLD, 1);
			}
#endif
#if ENABLE_PATTERN_DETECTION
			commit_pattern_from_counts(avCalls, sbuf, rbuf, size);
#endif
#if ENABLE_TIMING
			insert_op_exec_times_data(op_exec_times, late_arrival_timings, size);
#endif
		}
		avCalls++;
		avCallsLogged++;
	}
	else
	{
		// No need to profile that call but we still count the number of alltoallv calls
		ret = PMPI_Alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
		avCalls++;
	}

#if SYNC
	// We sync all the ranks again to make sure that rank 0, who does some calculations,
	// does not artificially fall behind.
	MPI_Barrier(comm);
#endif

	return ret;
}

int MPI_Alltoallv(const void *sendbuf, const int *sendcounts, const int *sdispls,
				  MPI_Datatype sendtype, void *recvbuf, const int *recvcounts,
				  const int *rdispls, MPI_Datatype recvtype, MPI_Comm comm)
{
	return _mpi_alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
}

void mpi_alltoallv_(void *sendbuf, MPI_Fint *sendcount, MPI_Fint *sdispls, MPI_Fint *sendtype,
					void *recvbuf, MPI_Fint *recvcount, MPI_Fint *rdispls, MPI_Fint *recvtype,
					MPI_Fint *comm, MPI_Fint *ierr)
{
	int c_ierr;
	MPI_Comm c_comm;
	MPI_Datatype c_sendtype, c_recvtype;

	c_comm = PMPI_Comm_f2c(*comm);
	c_sendtype = PMPI_Type_f2c(*sendtype);
	c_recvtype = PMPI_Type_f2c(*recvtype);

	sendbuf = (char *)OMPI_F2C_IN_PLACE(sendbuf);
	sendbuf = (char *)OMPI_F2C_BOTTOM(sendbuf);
	recvbuf = (char *)OMPI_F2C_BOTTOM(recvbuf);

	c_ierr = MPI_Alltoallv(sendbuf,
						   (int *)OMPI_FINT_2_INT(sendcount),
						   (int *)OMPI_FINT_2_INT(sdispls),
						   c_sendtype,
						   recvbuf,
						   (int *)OMPI_FINT_2_INT(recvcount),
						   (int *)OMPI_FINT_2_INT(rdispls),
						   c_recvtype, c_comm);
	if (NULL != ierr)
		*ierr = OMPI_INT_2_FINT(c_ierr);
}

/*************************************************************************
 * Copyright (c) 2019-2010, Mellanox Technologies, Inc. All rights reserved.
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <unistd.h>

#define DEBUG 0
#define HOSTNAME_LEN 16
#define DEFAULT_MSG_SIZE_THRESHOLD 200 // The default threshold between small and big messages
#define SYNC 0						   // Force the ranks to sync after each alltoallv operations to ensure rank 0 does not artifically fall behind

// Data type for storing comm size, alltoallv counts, send/recv count, etc
typedef struct avSRCountNode
{
	int size;
	int count;
	int comm;
	int sendtype_size;
	int recvtype_size;
	int *send_data;
	int *recv_data;
	double *op_exec_times;
	struct avSRCountNode *next;
} avSRCountNode_t;

typedef struct avTimingsNode
{
	int size;
	double *timings;
	struct avTimingsNode *next;
} avTimingsNode_t;

static FILE *f = NULL;
static avSRCountNode_t *head = NULL;
static avTimingsNode_t *op_timing_exec_head = NULL;
static avTimingsNode_t *op_timing_exec_tail = NULL;
static int world_size = -1;
static int myrank = -1;
static int avCalls = 0;
char myhostname[HOSTNAME_LEN];
char *hostnames = NULL; // Only used by rank0

// Buffers used to store data through all alltoallv calls
int *sbuf = NULL;
int *rbuf = NULL;
double *op_exec_times = NULL;

// Compare if two arrays are identical.
static int same_data(int *dest, int *src, int size)
{
	int i, j, num = 0;
	for (i = 0; i < size; i++)
	{
		for (j = 0; j < size; j++)
		{
			if (dest[num] != src[num])
				return 0;
			num++;
		}
	}
	return 1;
}

// Compare new send count data with existing data.
// If there is a match, increas the counter. Add new data, otherwise.
// recv count was not compared.
static void insert_sendrecv_data(int *sbuf, int *rbuf, int size, int sendtype_size, int recvtype_size)
{
	int i, j, num = 0;
	struct avSRCountNode *newNode = NULL;
	struct avSRCountNode *temp;

	temp = head;
	while (temp != NULL)
	{
		if (same_data(temp->send_data, sbuf, size) == 0 || temp->size != size || temp->recvtype_size != recvtype_size || temp->sendtype_size != sendtype_size)
		{
			if (DEBUG)
				fprintf(f, "new data: %d\n", size);
			if (temp->next != NULL)
				temp = temp->next;
			else
				break;
		}
		else
		{
			temp->count++;
			if (DEBUG)
				fprintf(f, "old data: %d --> %d --- %d\n", size, temp->size, temp->count);
			return;
		}
	}

	if (DEBUG)
		fprintf(f, "no data: %d \n", size);
	newNode = (struct avSRCountNode *)malloc(sizeof(avSRCountNode_t));
	assert(newNode != NULL);

	newNode->size = size;
	newNode->count = 1;
	newNode->send_data = (int *)malloc(size * size * (sizeof(int)));
	newNode->recv_data = (int *)malloc(size * size * (sizeof(int)));
	newNode->sendtype_size = sendtype_size;
	newNode->recvtype_size = recvtype_size;
	newNode->next = NULL;
	if (DEBUG)
		fprintf(f, "new entry: %d --> %d --- %d\n", size, newNode->size, newNode->count);

	for (i = 0; i < size; i++)
	{
		for (j = 0; j < size; j++)
		{
			newNode->send_data[num] = sbuf[num];
			newNode->recv_data[num] = rbuf[num];
			num++;
		}
	}

	if (head == NULL)
	{
		head = newNode;
	}
	else
	{
		temp->next = newNode;
	}
}

static void insert_op_exec_times_data(double *timings, int size)
{
	assert(timings);
	struct avTimingsNode *newNode = (struct avTimingsNode *)calloc(1, sizeof(struct avTimingsNode));
	newNode->timings = (double *)malloc(size * sizeof(double));
	assert(newNode);

	newNode->size = size;
	for (int i = 0; i < size; i++)
	{
		newNode->timings[i] = timings[i];
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

static void print_data(int *buf, int size, int type_size)
{
	int i, j, num = 0;

	int *zeros = (int *)calloc(size, sizeof(int));
	int *sums = (int *)calloc(size, sizeof(int));
	int *mins = (int *)calloc(size, sizeof(int));
	int *maxs = (int *)calloc(size, sizeof(int));
	int *small_messages = (int *)calloc(size, sizeof(int));
	int msg_size_threshold = DEFAULT_MSG_SIZE_THRESHOLD;

	assert(f);
	assert(zeros);
	assert(sums);
	assert(mins);
	assert(maxs);
	assert(small_messages);

	char *env_var = getenv("MSG_SIZE_THRESHOLD");
	if (env_var != NULL)
	{
		msg_size_threshold = atoi(env_var);
	}

	fprintf(f, "### Raw counters\n");
	for (i = 0; i < size; i++)
	{
		mins[i] = buf[num];
		maxs[i] = buf[num];
		for (j = 0; j < size; j++)
		{
			sums[i] += buf[num];
			if (buf[num] == 0)
			{
				zeros[i]++;
			}
			if (buf[num] < mins[i])
			{
				mins[i] = buf[num];
			}
			if (maxs[i] < buf[num])
			{
				maxs[i] = buf[num];
			}
			if ((buf[num] * type_size) < msg_size_threshold)
			{
				small_messages[i]++;
			}

			fprintf(f, "%d ", buf[num]);
			num++;
		}
		fprintf(f, "\n");
	}
	fprintf(f, "\n");

	fprintf(f, "### Amount of data per rank\n");
	for (i = 0; i < size; i++)
	{
		fprintf(f, "Rank %d: %d bytes\n", i, sums[i] * type_size);
	}
	fprintf(f, "\n");

	fprintf(f, "### Number of zeros\n");
	int total_zeros = 0;
	for (i = 0; i < size; i++)
	{
		total_zeros += zeros[i];
		double ratio_zeros = zeros[i] * 100 / size;
		fprintf(f, "Rank %d: %d/%d (%f%%) zero(s)\n", i, zeros[i], size, ratio_zeros);
	}
	double ratio_zeros = (total_zeros*100)/(size*size);
	fprintf(f, "Total: %d/%d (%f%%)\n", total_zeros, size*size, ratio_zeros);
	fprintf(f, "\n");

	fprintf(f, "### Data size min/max\n");
	for (i = 0; i < size; i++)
	{
		fprintf(f, "Rank %d: Min = %d bytes; max = %d bytes\n", i, mins[i] * type_size, maxs[i] * type_size);
	}
	fprintf(f, "\n");

	fprintf(f, "### Small vs. large messages\n");
	int total_small_msgs = 0;
	for (i = 0; i < size; i++)
	{
		total_small_msgs += small_messages[i];
		float ratio = small_messages[i] * 100 / size;
		fprintf(f, "Rank %d: %f%% small messages; %f%% large messages\n", i, ratio, 100 - ratio);
	}
	double total_ratio_small_msgs = (total_small_msgs * 100) / (size * size);
	fprintf(f, "Total small messages: %d/%d (%f%%)", total_small_msgs, size * size, total_ratio_small_msgs);
	fprintf(f, "\n");

	free(sums);
	free(zeros);
	free(mins);
	free(maxs);
	free(small_messages);
}

static void display_data()
{
	int i;
	avSRCountNode_t *srCountPtr;
	avTimingsNode_t *tPtr;

	// Display the send/receive counts data
	srCountPtr = head;
	fprintf(f, "# Send/recv counts for alltoallv operations:\n");
	while (srCountPtr != NULL)
	{
		fprintf(f, "comm size = %d, alltoallv calls = %d\n\n", srCountPtr->size, srCountPtr->count);

		fprintf(f, "## Data sent per rank - Type size: %d\n\n", srCountPtr->sendtype_size);
		print_data(srCountPtr->send_data, srCountPtr->size, srCountPtr->sendtype_size);
		fprintf(f, "## Data received per rank - Type size: %d\n\n", srCountPtr->recvtype_size);
		print_data(srCountPtr->recv_data, srCountPtr->size, srCountPtr->recvtype_size);
		srCountPtr = srCountPtr->next;
	}

	// Display the timing data
	tPtr = op_timing_exec_head;
	i = 0;
	fprintf(f, "# Execution times of Alltoallv operations");
	while (tPtr != NULL)
	{
		fprintf(f, "## Alltoallv call #%d\n", i);
		for (int i = 0; i < tPtr->size; i++)
		{
			fprintf(f, "Rank %d: %f\n", i, tPtr->timings[i]);
		}
		fprintf(f, "\n");
		tPtr = tPtr->next;
	}
}

static void display_per_host_data(int size)
{
	for (int i = 0; i < world_size; i++)
	{
	}
}

int MPI_Init(int *argc, char ***argv)
{
	int ret;
	char buf[200];

	gethostname(myhostname, HOSTNAME_LEN);

	ret = PMPI_Init(argc, argv);
	MPI_Comm_rank(MPI_COMM_WORLD, &myrank);
	MPI_Comm_size(MPI_COMM_WORLD, &world_size);

	// Allocate buffers reused between alltoallv calls
	// Note the buffer may be used on a communicator that is not comm_world
	// but in any case, it will be smaller or of the same size than comm_world.
	// So we allocate the biggest buffers possible but reuse them during the
	// entire execution of the application.
	sbuf = (int *)malloc(world_size * world_size * (sizeof(int)));
	assert(sbuf);
	rbuf = (int *)malloc(world_size * world_size * (sizeof(int)));
	assert(rbuf);
	op_exec_times = (double *)malloc(world_size * sizeof(double));
	assert(op_exec_times);

	if (f == NULL && myrank == 0)
	{
		sprintf(buf, "profile_alltoallv.%d.md", myrank);
		f = fopen(buf, "w");
		assert(f != NULL);
	}

	// Make sure we do not create an articial imbalance between ranks.
	MPI_Barrier(MPI_COMM_WORLD);

	return ret;
}

// During Finalize, it prints all stored data to a file.
int MPI_Finalize()
{
	if (myrank == 0)
	{
		if (f != NULL)
		{
			fprintf(f, "# Summary\n");
			fprintf(f, "Total number of alltoallv calls = %d\n\n", avCalls);
			display_data();
		}

		while (head != NULL)
		{
			avSRCountNode_t *c_ptr = head->next;
			free(head->recv_data);
			free(head->send_data);
			free(head);
			head = c_ptr;
		}

		while (op_timing_exec_head != NULL)
		{
			avTimingsNode_t *t_ptr = op_timing_exec_head->next;
			free(op_timing_exec_head->timings);
			free(op_timing_exec_head);
			op_timing_exec_head = t_ptr;
		}
		op_timing_exec_tail = NULL;

		fprintf(f, "# Hostnames\n");
		for (int i = 0; i < world_size; i++)
		{
			char h[HOSTNAME_LEN];
			int offset = HOSTNAME_LEN * i;
			for (int j = 0; j < HOSTNAME_LEN; j++)
			{
				h[j] = hostnames[offset + j];
			}
			fprintf(f, "Rank %d: %s\n", i, h);
		}

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
		if (hostnames)
		{
			free(hostnames);
		}

		if (f != NULL)
		{
			fclose(f);
		}
	}
	return PMPI_Finalize();
}

int MPI_Alltoallv(const void *sendbuf, const int *sendcounts, const int *sdispls,
				  MPI_Datatype sendtype, void *recvbuf, const int *recvcounts,
				  const int *rdispls, MPI_Datatype recvtype, MPI_Comm comm)
{
	int size;
	int i, j;
	int localrank;
	int ret;

	MPI_Comm_rank(comm, &localrank);
	MPI_Comm_size(comm, &size);
	avCalls++;

	if (myrank == 0)
	{
		hostnames = (char *)malloc(size * HOSTNAME_LEN * sizeof(char));
	}

	double t_start = MPI_Wtime();
	ret = PMPI_Alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
	double t_end = MPI_Wtime();
	double t_op = t_end - t_start;

	// Gather a bunch of counters
	MPI_Gather(sendcounts, size, MPI_INT, sbuf, size, MPI_INT, 0, comm);
	MPI_Gather(recvcounts, size, MPI_INT, rbuf, size, MPI_INT, 0, comm);
	MPI_Gather(&t_op, 1, MPI_DOUBLE, op_exec_times, 1, MPI_DOUBLE, 0, comm);
	MPI_Gather(myhostname, HOSTNAME_LEN, MPI_CHAR, hostnames, HOSTNAME_LEN, MPI_CHAR, 0, comm);

	if (myrank == 0)
	{
		if (DEBUG)
			fprintf(f, "Root: global %d - %d   local %d - %d\n", world_size, myrank, size, localrank);

		insert_sendrecv_data(sbuf, rbuf, size, sizeof(sendtype), sizeof(recvtype));
		insert_op_exec_times_data(op_exec_times, size);
		fflush(f);
	}

#if SYNC
	// We sync all the ranks again to make sure that rank 0, who does some calculations,
	// does not artificially fall behind.
	MPI_Barrier(comm);
#endif

	return ret;
}

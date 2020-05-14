/*************************************************************************
 * Copyright (c) 2019-2010, Mellanox Technologies, Inc. All rights reserved.
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <mpi.h>
#include <stdbool.h>
#include <assert.h>

#include "alltoallv_profiler.h"
#include "logger.h"
#include "grouping.h"

avSRCountNode_t *head = NULL;
avTimingsNode_t *op_timing_exec_head = NULL;
avTimingsNode_t *op_timing_exec_tail = NULL;

static int world_size = -1;
static int myrank = -1;
static int avCalls = 0;		  // Total number of alltoallv calls that we went through
static int avCallsLogged = 0; // Total number of alltoallv calls for which we gathered data
static int avCallStart = -1;  // Number of alltoallv call during which we started to gather data
//char myhostname[HOSTNAME_LEN];
//char *hostnames = NULL; // Only used by rank0

// Buffers used to store data through all alltoallv calls
int *sbuf = NULL;
int *rbuf = NULL;
double *op_exec_times = NULL;

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

	assert(logger);
#if DEBUG
	asert(logger->f);
#endif

	temp = head;
	while (temp != NULL)
	{
		if (same_data(temp->send_data, sbuf, size) == 0 || temp->size != size || temp->recvtype_size != recvtype_size || temp->sendtype_size != sendtype_size)
		{
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
			temp->count++;
#if DEBUG
			fprintf(logger->f, "old data: %d --> %d --- %d\n", size, temp->size, temp->count);
#endif
			return;
		}
	}

#if DEBUG
	fprintf(logger->f, "no data: %d \n", size);
#endif
	newNode = (struct avSRCountNode *)malloc(sizeof(avSRCountNode_t));
	assert(newNode != NULL);

	newNode->size = size;
	newNode->count = 1;
	newNode->send_data = (int *)malloc(size * size * (sizeof(int)));
	newNode->recv_data = (int *)malloc(size * size * (sizeof(int)));
	newNode->sendtype_size = sendtype_size;
	newNode->recvtype_size = recvtype_size;
	newNode->next = NULL;
#if DEBUG
	fprintf(logger->f, "new entry: %d --> %d --- %d\n", size, newNode->size, newNode->count);
#endif

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
	int i;
	for (i = 0; i < size; i++)
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

static void display_per_host_data(int size)
{
	int i;
	for (i = 0; i < world_size; i++)
	{
	}
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
	op_exec_times = (double *)malloc(world_size * sizeof(double));
	assert(op_exec_times);

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

	c_ierr = _mpi_init(&argc, &argv);
	if (NULL != ierr)
		*ierr = OMPI_INT_2_FINT(c_ierr);
}

// During Finalize, it prints all stored data to a file.
int _mpi_finalize()
{
	if (myrank == 0)
	{
		log_profiling_data(logger, avCalls, avCallStart, avCallsLogged);

		// All data has been handled, now we can clean up
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
	if (avCalls < NUM_CALL_START_PROFILING)
	{
		need_profile = false;
	}
	else
	{
		if (-1 != DEFAULT_LIMIT_ALLTOALLV_CALLS && avCallsLogged >= DEFAULT_LIMIT_ALLTOALLV_CALLS)
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

		double t_start = MPI_Wtime();
		ret = PMPI_Alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
		double t_end = MPI_Wtime();
		double t_op = t_end - t_start;

		// Gather a bunch of counters
		MPI_Gather(sendcounts, size, MPI_INT, sbuf, size, MPI_INT, 0, comm);
		MPI_Gather(recvcounts, size, MPI_INT, rbuf, size, MPI_INT, 0, comm);
		MPI_Gather(&t_op, 1, MPI_DOUBLE, op_exec_times, 1, MPI_DOUBLE, 0, comm);
		//MPI_Gather(myhostname, HOSTNAME_LEN, MPI_CHAR, hostnames, HOSTNAME_LEN, MPI_CHAR, 0, comm);

		if (myrank == 0)
		{
#if DEBUG
			fprintf(logger->f, "Root: global %d - %d   local %d - %d\n", world_size, myrank, size, localrank);
#endif

			insert_sendrecv_data(sbuf, rbuf, size, sizeof(sendtype), sizeof(recvtype));
			insert_op_exec_times_data(op_exec_times, size);
		}
		avCalls++;
		avCallsLogged++;
	}
	else
	{
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

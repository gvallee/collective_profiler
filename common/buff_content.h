/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/


#ifndef MPI_COLLECTIVE_PROFILER_BUFFCONTENT_H
#define MPI_COLLECTIVE_PROFILER_BUFFCONTENT_H

#include <inttypes.h>
#include <stdbool.h>

#include "mpi.h"

#define COLLECTIVE_PROFILER_MAX_CALL_CHECK_BUFF_CONTENT_ENVVAR "COLLECTIVE_PROFILER_MAX_CALL_CHECK_BUFF_CONTENT"
#define COLLECTIVE_PROFILER_CHECK_SEND_BUFF_ENVVAR "COLLECTIVE_PROFILER_CHECK_SEND_BUFF"

// buffcontent_logger is the central structure to track and profile backtrace in
// the context of MPI collective. We track in a unique manner each trace but for each
// trace, multiple contexts can be tracked. A context is the tuple communictor id/rank/call.
typedef struct buffcontent_logger
{
    char *collective_name;
    uint64_t id;
    int world_rank;
    FILE *fd;
    char *filename;
    uint64_t comm_id;
    MPI_Comm comm;
    struct buffcontent_logger *next;
    struct buffcontent_logger *prev;
} buffcontent_logger_t;

int store_call_data(char *collective_name, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void* buf, int counts[], int displs[], int dtsize);
int read_and_compare_call_data(char *collective_name, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], int dtsize, bool check);
int release_buffcontent_loggers();

#endif // MPI_COLLECTIVE_PROFILER_BUFFCONTENT_H

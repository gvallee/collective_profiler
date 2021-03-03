/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef COLLECTIVE_PROFILER_TIMINGS_H
#define COLLECTIVE_PROFILER_TIMINGS_H

#include <inttypes.h>
#include "mpi.h"

typedef struct comm_timing_logger
{
    uint32_t comm_id;
    FILE *fd;
    char *filename;
    struct comm_timing_logger *next;
    struct comm_timing_logger *prev;
} comm_timing_logger_t;

int init_time_tracking(MPI_Comm comm, char *collective_name, int world_rank, int comm_rank, int jobid, comm_timing_logger_t **logger);
int fini_time_tracking(comm_timing_logger_t **logger);
int release_time_loggers();
int commit_timings(MPI_Comm comm, char *collective_name, int world_rank, int comm_rank, int jobid, double *times, int comm_size, uint64_t n_call);

#endif // COLLECTIVE_PROFILER_TIMINGS_H
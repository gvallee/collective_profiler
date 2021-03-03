/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef MPI_COLLECTIVE_PROFILER_LOCATION_H
#define MPI_COLLECTIVE_PROFILER_LOCATION_H

#include <inttypes.h>
#include <stdio.h>

#include "mpi.h"

// location_logger is the central structure to track and profile locations of ranks in
// the context of MPI collective. 
typedef struct location_logger
{
    char *collective_name;
    int world_rank;
    FILE *fd; // File descriptor to write the trace
    char *filename; // Filename for the trace
    int *world_comm_ranks;
    size_t calls_count;
    size_t calls_max;
    uint64_t *calls;
    uint64_t commid;
    int comm_size;
    char *locations;
    int *pids;
    struct location_logger *next;
    struct location_logger *prev;
} location_logger_t;

int commit_rank_locations(char *collective_name, MPI_Comm comm, int comm_size, int world_rank, int comm_rank, int *pids, int *world_comm_ranks, char *hostnames, uint64_t n_call);
int release_location_loggers();


#endif // MPI_COLLECTIVE_PROFILER_LOCATION_H
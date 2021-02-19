/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef MPI_COLLECTIVE_PROFILER_BACKTRACE_H
#define MPI_COLLECTIVE_PROFILER_BACKTRACE_H

#include <stdlib.h>
#include <inttypes.h>
#include <stdio.h>

#include "mpi.h"

typedef struct trace_context 
{
    uint32_t comm_id; // Communicator ID for the associated trace
    int comm_rank; // Rank on the communicator
    int world_rank;
    uint64_t *calls; // All the calls associated to this backtrace
    size_t calls_count;
    size_t max_calls;
    struct trace_context *next;
    struct trace_context *prev;
} trace_context_t;

// backtrace_logger is the central structure to track and profile backtrace in
// the context of MPI collective. We track in a unique manner each trace but for each
// trace, multiple contexts can be tracked. A context is the tuple communictor id/rank/call.
typedef struct backtrace_logger
{
    char *collective_name;
    trace_context_t *contexts;
    uint64_t id;
    int world_rank;
    size_t num_contexts;
    size_t max_contexts;
    char **trace;
    size_t trace_size;
    FILE *fd; // File descriptor to write the trace
    char *filename; // Filename for the trace
    struct backtrace_logger *next;
    struct backtrace_logger *prev;
} backtrace_logger_t;

int insert_caller_data(char *collective_name, char **trace, size_t trace_size, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call);
int release_backtrace_loggers();

#endif // MPI_COLLECTIVE_PROFILER_BACKTRACE_H
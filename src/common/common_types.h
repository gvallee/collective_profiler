/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdint.h>
#include <stdlib.h>

#ifndef _COLLECTIVE_PROFILER_COMMON_TYPES_H
#define _COLLECTIVE_PROFILER_COMMON_TYPES_H

// Compact way to save send/recv counts of ranks within a single alltoallv call
typedef struct counts_data
{
    int *counters; // the actual counters (i.e., send/recv counts)
    int num_ranks; // The number of ranks having that series of counters
    int max_ranks; // The current size of the ranks array
    int *ranks;    // The list of ranks having that series of counters
} counts_data_t;

// Data type for storing comm size, alltoallv counts, send/recv count, etc
typedef struct avSRCountNode
{
    int size;
    int rank_vec_len; // =1 for alltoall, = comm_size for alltoallv
    uint64_t count; // How many time we detected the pattern; also size of list_calls
    uint64_t max_calls;
    uint64_t *list_calls; // Which calls produced the pattern
    int comm;
    int sendtype_size;
    int recvtype_size;
    int send_data_size;        // Size of the array of unique series of send counters
    int recv_data_size;        // Size of the array of unique series of recv counters
    counts_data_t **send_data; // Array of unique series of send counters
    counts_data_t **recv_data; // Array of unique series of recv counters
    double *op_exec_times;
    double *late_arrival_timings;
    struct avSRCountNode *next;
} avSRCountNode_t;

typedef struct avTimingsNode
{
    int size;
    double *timings; // Time spent in the alltoallv function
    struct avTimingsNode *next;
} avTimingsNode_t;

typedef struct avPattern
{
    // <n_ranks> ranks send to or receive from <n_peers> other ranks
    int n_ranks;
    int n_peers;
    uint64_t n_calls;   // How many collective calls have that pattern
    int comm_size; // Size of the communicator for which the pattern was detected. Not always used.
    struct avPattern *next;
} avPattern_t;

typedef struct avCallPattern  // TODO is this anything to do with counts? What is an av_Pattern? See next def above. So this pattern is to do with rank patterns.
{
    uint64_t n_calls;
    uint64_t *calls;
    avPattern_t *spatterns;
    avPattern_t *rpatterns;
    struct avCallPattern *next;
} avCallPattern_t;

typedef struct caller_info
{
    uint64_t n_calls;
    uint64_t *calls;
    char *caller;
    struct caller_info *next;
} caller_info_t;

typedef char* (*get_full_filename_fn_t)(int, char *, int, int);

typedef struct logger_config
{
    get_full_filename_fn_t get_full_filename;
    char *collective_name;
    uint64_t limit_number_calls;
} logger_config_t;

enum
{
    MAIN_CTX = 0,
    SEND_CTX,
    RECV_CTX
};

#endif // _COLLECTIVE_PROFILER_COMMON_TYPES_H
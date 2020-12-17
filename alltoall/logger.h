/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/
/******************************************************************************************************
 * Copyright (c) 2020, University College London and Mellanox Technolgies Limited. All rights reserved.
 * - for further contributions 
 ******************************************************************************************************/


#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <assert.h>
#include <string.h>
#include <inttypes.h>

#include "alltoall_profiler.h"
#include "common_types.h"

#ifndef LOGGER_H
#define LOGGER_H

typedef struct logger
{
    int world_size;            // COMM_WORLD size
    int rank;                  // Rank that is handling the current logger.
    char *main_filename;       // Path to the main profile file.
    FILE *f;                   // File handle to save general profile data. Other files are created for specific data.
    char *sendcounts_filename; // Path of the send counts profile.
    FILE *sendcounters_fh;     // File handle used to save send counters.
    char *recvcounts_filename; // Path of the receive counts profile.
    FILE *recvcounters_fh;     // File handle used to save recv counters.
    char *sums_filename;       // Path of the sums profiles.
    FILE *sums_fh;             // File handle used to save data related to amount of data exchanged.
    char *timing_filename;     // Path of the timing profile.
    FILE *timing_fh;           // File handle used to save data related to timing of operations.
    get_full_filename_fn_t get_full_filename;
} logger_t;

extern logger_t *logger_init();
extern void logger_fini(logger_t **l);
extern void log_profiling_data(logger_t *logger, uint64_t avCalls, uint64_t avCallStart, uint64_t avCallsLogged, avSRCountNode_t *counters_list, avTimingsNode_t *times_list);
extern void log_timing_data(logger_t *logger, avTimingsNode_t *times_list);
extern int *lookup_rank_counters(int data_size, counts_data_t **data, int rank);
extern char *compress_int_array(int *array, int size);

#endif // LOGGER_H
/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <assert.h>
#include <string.h>

#include "alltoallv_profiler.h"

#ifndef LOGGER_H
#define LOGGER_H

typedef struct logger
{
    FILE *f;               // File handle to save general profile data. Other files are created for specific data
    FILE *sendcounters_fh; // File handle used to save send counters
    FILE *recvcounters_fh; // File handle used to save recv counters
    FILE *sums_fh;         // File handle used to save data related to amount of data exchanged
    FILE *timing_fh;       // File handle used to save data related to timing of operations
} logger_t;

extern logger_t *logger_init();
extern void logger_fini(logger_t **l);
extern void log_profiling_data(logger_t *logger, int avCalls, int avCallStart, int avCallsLogged, avSRCountNode_t *counters_list, avTimingsNode_t *times_list);
extern int *lookup_rank_counters(int data_size, counts_data_t **data, int rank);

#endif // LOGGER_H
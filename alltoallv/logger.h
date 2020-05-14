/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <assert.h>

#ifndef LOGGER_H
#define LOGGER_H

typedef struct logger
{
    FILE *f;               // File handle to save general profile data. Other files are created for specific data
    FILE *sendcounters_fh; // File handle used to save send counters
    FILE *recvcounters_fh; // File handle used to save recv counters
} logger_t;

extern logger_t *logger_init();
extern void logger_fini(logger_t **l);
extern void log_profiling_data(logger_t *logger, int avCalls, int avCallStart, int avCallsLogged);

#endif // LOGGER_H
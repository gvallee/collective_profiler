/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/
/******************************************************************************************************
 * Copyright (c) 2020, University College London and Mellanox Technolgies Limited. All rights reserved.
 * - for further contributions 
 ******************************************************************************************************/


#ifndef ALLTOALL_PROFILER_H
#define ALLTOALL_PROFILER_H

#include <stdbool.h>
#include <assert.h>
#include <stdarg.h>
#include <stdlib.h>
#include <stdint.h>

#include "config.h"

#if DEBUG
#if DEBUG_FLUSH
#define DEBUG_ALLTOALL_PROFILING(fmt, ...) \
    fprintf(stdout, "A2A - [%s:%d]" fmt, __FILE__, __LINE__, __VA_ARGS__); \
    fprintf(stdout)
#else
#define DEBUG_ALLTOALL_PROFILING(fmt, ...) \
    fprintf(stdout, "A2A - [%s:%d]" fmt, __FILE__, __LINE__, __VA_ARGS__)
#endif //DEBUG_FLUSH
#else
#define DEBUG_ALLTOALL_PROFILING(fmt, ...) \
    do                                      \
    {                                       \
    } while (0)
#endif // DEBUG

#endif // ALLTOALL_PROFILER_H

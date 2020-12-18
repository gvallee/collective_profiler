/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef ALLTOALLV_PROFILER_H
#define ALLTOALLV_PROFILER_H

#include <stdbool.h>
#include <assert.h>
#include <stdarg.h>
#include <stdlib.h>
#include <stdint.h>

#include "common_types.h"
#include "config.h"

#if DEBUG
#define DEBUG_ALLTOALLV_PROFILING(fmt, ...) \
    fprintf(stdout, "A2A - [%s:%d]" fmt, __FILE__, __LINE__, __VA_ARGS__)
#else
#define DEBUG_ALLTOALLV_PROFILING(fmt, ...) \
    do                                      \
    {                                       \
    } while (0)
#endif // DEBUG

#endif // ALLTOALLV_PROFILER_H

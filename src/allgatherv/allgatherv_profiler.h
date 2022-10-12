/*************************************************************************
 * Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef ALLGATHERV_PROFILER_H
#define ALLGATHERV_PROFILER_H

#include <stdbool.h>
#include <assert.h>
#include <stdarg.h>
#include <stdlib.h>
#include <stdint.h>

#include "common_types.h"
#include "config.h"

#if DEBUG
#define DEBUG_ALLGATHERV_PROFILING(fmt, ...) \
    fprintf(stdout, "ALLGATHERV - [%s:%d]" fmt, __FILE__, __LINE__, __VA_ARGS__)
#else
#define DEBUG_ALLGATHERV_PROFILING(fmt, ...) \
    do                                      \
    {                                       \
    } while (0)
#endif // DEBUG

#endif // ALLGATHERV_PROFILER_H

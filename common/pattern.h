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
#include <stdbool.h>

#include "collective_profiler_config.h"
#include "common_types.h"

#ifndef PATTERN_H
#define PATTERN_H

#define ENABLE_PATTERN_DEBUGING (0)
#if ENABLE_PATTERN_DEBUGING
#define DEBUG_PATTERN(fmt, ...) \
    fprintf(stdout, "A2A - [%s:%d]" fmt, __FILE__, __LINE__, __VA_ARGS__)
#else
#define DEBUG_PATTERN(fmt, ...) \
    do                         \
    {                          \
    } while (0)
#endif // ENABLE_PATTERN_DEBUGING

extern avPattern_t *add_pattern(avPattern_t *patterns, int num_ranks, int num_peers);
extern avCallPattern_t *extract_call_patterns(int callID, int *send_counts, int *recv_counts, int size);
extern avCallPattern_t *lookup_call_patterns(avCallPattern_t *call_patterns);
extern void free_patterns(avPattern_t *p);
extern avPattern_t *add_pattern_for_size(avPattern_t *patterns, int num_ranks, int num_peers, int size);
extern int get_size_patterns(avPattern_t *p);
extern bool compare_patterns(avPattern_t *p1, avPattern_t *p2);

#endif // PATTERN_H
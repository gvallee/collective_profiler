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

#include "alltoall_profiler.h"

#ifndef PATTERN_H
#define PATTERN_H

extern avPattern_t *add_pattern(avPattern_t *patterns, int num_ranks, int num_peers);
extern avCallPattern_t *extract_call_patterns(int callID, int *send_counts, int *recv_counts, int size);
extern avCallPattern_t *lookup_call_patterns(avCallPattern_t *call_patterns);
extern void free_patterns(avPattern_t *p);
extern avPattern_t *add_pattern_for_size(avPattern_t *patterns, int num_ranks, int num_peers, int size);
extern int get_size_patterns(avPattern_t *p);
extern bool compare_patterns(avPattern_t *p1, avPattern_t *p2);

#endif // PATTERN_H
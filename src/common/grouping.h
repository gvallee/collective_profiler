/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include "common_utils.h"

#ifndef COLLECTIVE_PROFILER_GROUPING_H
#define COLLECTIVE_PROFILER_GROUPING_H

#define GROUPING_DEBUG (0)
#define DEBUG_GROUPING(fmt, ...)               \
    do                                         \
    {                                          \
        if (GROUPING_DEBUG > 0)                \
        {                                      \
            fprintf(stdout, fmt, __VA_ARGS__); \
        }                                      \
    } while (0);

/*
 * data_point represents a data point that belongs to a group.
 * We do not actually store the values here because we assume that
 * all data points are also stored in a separate array. As a result,
 * we only store the rank and to get the value, we get the value
 * from the array using the rank as the index.
 */
typedef struct data_point
{
    int rank;
    struct data_point *prev;
    struct data_point *next;
} data_point_t;

typedef struct group
{
    struct group *prev;
    struct group *next;
    int size;
    int max_size;
    int *elts;
    int min;
    int max;
    int cached_sum;
} group_t;

/*
 * grouping_engine_t is an opaque handle that handles
 * grouping. Practically, it allows us to avoid global
 * variables and make it easier to use.
 */
typedef struct grouping_engine
{
    group_t *head_gp;
    group_t *tail_gp;
} grouping_engine_t;

extern int add_datapoint(grouping_engine_t *e, int rank, int *values);
extern int get_groups(grouping_engine_t *e, group_t **gps, int *num_group);
extern int grouping_init(grouping_engine_t **e);
extern int grouping_fini(grouping_engine_t **e);
extern int get_remainder(int n, int d);

#endif // COLLECTIVE_PROFILER_GROUPING_H
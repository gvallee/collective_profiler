/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#define DEBUG (0)
#define DEBUG_GROUPING(fmt, ...)               \
    do                                         \
    {                                          \
        if (DEBUG > 0)                         \
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
} group_t;

extern int add_datapoint(int rank, int *values);
extern int get_groups(group_t **gps, int *num_group);
extern int grouping_fini();

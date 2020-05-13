/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

/*
 * This algorithm is grouping data points that are practically a rank and an
 * associated value. In this software package, the value is the amount of data
 * that a rank is sending or receiving.
 * The grouping algorithm is quite simple:
 *  - We compare the median and the mean of the values and if they are too much
 *    appart (10% of the highest value by default), the group is removed and
 *    individual data point put back into the group to the left or the right,
 *    whichever the closer.
 *  - When checking if a group needs to be dismantled, we also check if we would
 *    have a better repartition of the data points by splitting the group in two.
 *  - The algorithm is recursive so when a group is dismantled and data points
 *    added to the group to the left or the right, these groups can end up 
 *    behind dismantled too. The aglorithm is supposed to stabilize since a group
 *    can be composed of a single data point.
 */

#include <stdlib.h>
#include <stdio.h>
#include <stdbool.h>

#include "grouping.h"

#define DEFAULT_GP_SIZE (20)

const float DEFAULT_MEAN_MEDIAN_DEVIATION = 0.1; // max of 10% of deviation

group_t *head_gp = NULL;
group_t *tail_gp = NULL;

int add_datapoint(int rank, int *values);

static int count_groups()
{
    int count = 0;
    group_t *ptr = head_gp;

    if (ptr == NULL)
    {
        return 0;
    }
    while (ptr != NULL)
    {
        ptr = ptr->next;
        count++;
    }

    return count;
}

static int get_value(int rank, int *values)
{
    return values[rank];
}

static int get_distance_from_group(int val, group_t *gp)
{
    if (gp->max > val && val > gp->min)
    {
        // something wrong, the value belong to the group
        return -1;
    }

    if (gp->max <= val)
    {
        return val - gp->max;
    }

    if (gp->min >= val)
    {
        return gp->min - val;
    }

    return -1;
}

/**
 * lookup_group finds the group that is the most likely to accept the data
 * point. For that we scan the min/max of each group, if the value is within
 * the min/max, the group is selected. If the value is between the max of a
 * group and the min of another group, we calculate the distance to each and
 * select the closest group
 */
static group_t *lookup_group(int val)
{
    group_t *ptr = head_gp;

    while (ptr != NULL)
    {
        // the value is within the range of a group
        if (ptr->min <= val && ptr->max >= val)
        {
            return ptr;
        }

        // the value is beyond the last group
        if (ptr->max < val && ptr == tail_gp)
        {
            return ptr;
        }

        // the value is before the first group
        if (ptr->min > val && ptr == head_gp)
        {
            return ptr;
        }

        if (ptr->max < val && ptr->next != NULL && ptr->next->min > val)
        {
            int d1 = get_distance_from_group(val, ptr);
            int d2 = get_distance_from_group(val, ptr->next);
            if (d1 >= d2)
            {
                return ptr;
            }
            return ptr->next;
        }

        ptr = ptr->next;
    }

    return NULL;
}

static int add_and_shift(group_t *gp, int rank, int index)
{
    int temp_rank;

    int i = gp->size;
    while (i > index)
    {
        gp->elts[i] = gp->elts[i - 1];
        i--;
    }
    gp->elts[index] = rank;
    return 0;
}

static int add_elt_to_group(group_t *gp, int rank, int *values)
{
    DEBUG_GROUPING("[%s:%d] Adding element %d-%d to group\n", __FILE__, __LINE__, rank, values[rank]);

    // When necessary, initialize the group
    if (gp->elts == NULL)
    {
        DEBUG_GROUPING("[%s:%d] Initializing group's elements\n", __FILE__, __LINE__);
        gp->elts = (int *)calloc(DEFAULT_GP_SIZE, sizeof(int));
        if (gp->elts == NULL)
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to allocate group's elements\n", __FILE__, __LINE__);
            return -1;
        }
        gp->max_size = DEFAULT_GP_SIZE;
        gp->size = 0;
    }

    // When necessary, grow the group
    if (gp->size == gp->max_size)
    {
        DEBUG_GROUPING("[%s:%d] Growing group to %d\n", __FILE__, __LINE__, gp->max_size + DEFAULT_GP_SIZE)
        gp->elts = (int *)realloc(gp->elts, gp->max_size + DEFAULT_GP_SIZE);
        if (gp->elts == NULL)
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to grow group\n", __FILE__, __LINE__);
            return -1;
        }
        gp->max_size += DEFAULT_GP_SIZE;
    }

    // The array is ordered
    DEBUG_GROUPING("[%s:%d] Inserting new element in group's elements\n", __FILE__, __LINE__);
    int i = 0;
    while (i < gp->size && values[gp->elts[i]] < values[rank])
    {
        i++;
    }
    if (i == gp->size)
    {
        // We add the new value at the end of the array
        gp->elts[i] = rank;
    }
    else
    {
        DEBUG_GROUPING("[%s:%d] Shifting elements within the group...\n", __FILE__, __LINE__);
        add_and_shift(gp, rank, i);
    }

    DEBUG_GROUPING("[%s:%d] Updating group's metadata (first rank is %d)...\n", __FILE__, __LINE__, rank);
    gp->size++;
    gp->min = values[gp->elts[0]];
    gp->max = values[gp->elts[gp->size - 1]];
    DEBUG_GROUPING("[%s:%d] Element successfully added (size: %d; min: %d, max: %d)\n", __FILE__, __LINE__, gp->size, gp->min, gp->max);
    return 0;
}

static group_t *create_group(int rank, int val, int *values)
{
    group_t *new_group = calloc(1, sizeof(group_t));
    new_group->min = val;
    new_group->max = val;
    if (add_elt_to_group(new_group, rank, values))
    {
        fprintf(stderr, "[%s:%d][ERROR] unable to actually add the rank to the group\n", __FILE__, __LINE__);
        return NULL;
    }

    return new_group;
}

static int add_group(group_t *gp)
{
    group_t *ptr = head_gp;

    if (head_gp == NULL)
    {
        // No group yet
        head_gp = tail_gp = gp;
        return 0;
    }

    while (ptr != NULL && (ptr->max < gp->min))
    {
        ptr = ptr->next;
    }

    if (ptr == NULL)
    {
        // the new group goes to the tail
        tail_gp->next = gp;
        tail_gp = gp;
    }
    else
    {
        // the new group is between two groups
        if (gp->next != NULL)
        {
            gp->next->prev = gp;
        }
        gp->next = gp;
    }

#if GROUPING_DEBUG
    fprintf(stdout, "[%s:%d] Number of groups: %d\n", __FILE__, __LINE__, count_groups());
#endif

    return 0;
}

static int
get_remainder(int n, int d)
{
    return (n - d * (n / d));
}

static double get_median(int size, int *data, int *values)
{
    int idx1, idx2;

    if (size == 1)
    {
        return values[data[0]];
    }

    if (get_remainder(size, 2) == 1)
    {
        idx1 = data[size / 2];
        DEBUG_GROUPING("[%s:%d] odd number of elements (%d), returning element %d\n", __FILE__, __LINE__, size, idx1);
        return (double)values[idx1];
    }

    idx1 = size / 2 - 1;
    idx2 = size / 2;
    int sum = values[data[idx1]] + values[data[idx2]];
    double median = ((double)(sum)) / 2;
    DEBUG_GROUPING("[%s:%d] even number of elements (%d), returning element between %d (val=%d) and %d (val=%d) - sum: %d - median: %f\n",
                   __FILE__, __LINE__, size, idx1, values[data[idx1]], idx2, values[data[idx2]], sum, median);
    return median;
}

static double get_median_with_additional_point(group_t *gp, int rank, int val, int *values)
{
    int candidate_rank;
    int middle = gp->size / 2;

    DEBUG_GROUPING("[%s:%d] Concidering add rank %d (value: %d) to group with min=%d and max=%d\n", __FILE__, __LINE__, rank, val, gp->min, gp->max);
    if (gp->size == 1) {

        return ((double)(values[gp->elts[0]] + val)) / 2;
    }

    if (get_remainder(gp->size + 1, 2) == 0)
    {
        // Odd total number of data points, even number of elements already in the group
        // Regardless of where the extra data point would land in the sorted list
        // of the group's elements, the point in the middle of the group's elements
        // will always be used.
        int value1 = -1, value2 = -1; // the two values used to calculate the median
        int index = middle;
        candidate_rank = gp->elts[index];
        DEBUG_GROUPING("[%s:%d] Odd total number of points, even number of elements already in group\n", __FILE__, __LINE__);
        DEBUG_GROUPING("[%s:%d] Pivot is at index %d; rank: %d, value: %d (middle: %d)\n", __FILE__, __LINE__, index, candidate_rank, values[candidate_rank], middle);
        if (values[candidate_rank] > val && val > values[candidate_rank - 1])
        {
            // The extra element goes in between the middle of the group's elements and the element to its left.
            // It shifts the element to the left to calculate the median
            value1 = val;
            value2 = values[candidate_rank];
        }
        if (values[candidate_rank] > val && val < values[candidate_rank - 1])
        {
            // The extra element falls toward the begining of the group's elements; it shifts the two elements
            // required to calculate the median
            value1 = values[candidate_rank - 1];
            value2 = values[candidate_rank];
        }
        if (values[candidate_rank] < val && val < values[candidate_rank + 1])
        {
            // The extra element falls in between the middle of the group's elements and the element to its right.
            value1 = values[candidate_rank];
            value2 = val;
        }
        if (values[candidate_rank] < val && val > values[candidate_rank + 1])
        {
            // The extra element falls toward the end of the group's elements.
            value1 = values[candidate_rank];
            value2 = values[candidate_rank + 1];
        }
        if (values[candidate_rank] == val) {
            // If the extra element has the same value than the middle of the group's elements, it will be added
            // right in the middle, shifting the second half of the group's elements starting by the elements to
            // the left
            value1 = val;
            value2 = values[candidate_rank]; 
        }
        if (values[candidate_rank + 1] == val) {
            // If the extra elements has the same value then the middle + 1 of the group's elements, it will be
            // added to the right of the middle, shifting the second half of the group's elements starting by the
            // elements to the right
            value1 = val;
            value2 = values[candidate_rank + 1];
        }
        if (value1 == -1 || value2 == -1)
        {
            int i;
            fprintf(stderr, "[%s:%d] unable to calculate median for the following group and new point rank: %d/value: %d\n", __FILE__, __LINE__, rank, val);
            for(i = 0; i <gp->size; i++)
            {
                fprintf(stderr, "-> elt %d: rank: %d, value: %d\n", i, gp->elts[i], values[gp->elts[i]]);
            }
        }

        int sum = value1 + value2;
        double median = ((double)(sum)) / 2;
        DEBUG_GROUPING("[%s:%d] With new data point, even number of elements (%d), returning element between value %d and %d - sum: %d - median: %f\n",
                       __FILE__, __LINE__, gp->size + 1, value1, value2, sum, median);
        return median;
    }
    else
    {
        // Even total number of data points, odd number of elements already in group
        int index = middle - 1;
        candidate_rank = gp->elts[index];
        DEBUG_GROUPING("[%s:%d] Even total number of points, odd number of elements already in group\n", __FILE__, __LINE__);
        DEBUG_GROUPING("[%s:%d] Pivot is at index %d; rank: %d, value: %d\n", __FILE__, __LINE__, index, candidate_rank, values[candidate_rank]);
        if (values[candidate_rank] > val) {
            // The new value falls to the left of the two elements from the original group that are candidate
            return (double)values[candidate_rank];
        }
        if (values[candidate_rank + 1] < val) {
            // The new value falls to the right of the two elements from the original group that are candidate
            return (double)values[candidate_rank + 1];
        }
        if (values[candidate_rank] <= val && values[candidate_rank + 1] >= val)
        {
            // The new element falls right in the middle of the new group
            return (double)val;
        }
    }

    // We should not actually get here
    return -1;
}

static double get_gp_median(group_t *gp, int *values)
{
    return get_median(gp->size, gp->elts, values);
}

static double get_mean(int size, int *data, int *values)
{
    int sum = 0;
    int i;

    for (i = 0; i < size; i++)
    {
        sum += values[data[i]];
    }

    double mean = (double)((double)(sum) / (double)(size));
    DEBUG_GROUPING("[%s:%d] Sum = %d; size = %d, mean = %f\n", __FILE__, __LINE__, sum, size, mean);
    return mean;
}

static double get_gp_mean(group_t *gp, int *values)
{
    return get_mean(gp->size, gp->elts, values);
}

static bool affinity_is_okay(double mean, double median)
{
    // If the mean and median do not deviate too much, we add the new data point to the group
    // Once the new data point is added to the group, we check the group to see if it needs
    // to be split.
    int max_mean_median;
    int min_mean_median;
    bool affinity_okay = false; // true when the mean and median are in acceptable range
    if (median > mean)
    {
        max_mean_median = median;
        min_mean_median = mean;
    }
    else
    {
        max_mean_median = mean;
        min_mean_median = median;
    }

    double a = (double)(max_mean_median * (1 - DEFAULT_MEAN_MEDIAN_DEVIATION));
    if (a <= (double)min_mean_median)
    {
        DEBUG_GROUPING("[%s:%d] Group is balanced\n", __FILE__, __LINE__);
        affinity_okay = true;
    }

    return affinity_okay;
}

static bool group_is_balanced(group_t *gp, int *values)
{
    // We calculate the mean and median.
    double median = get_gp_median(gp, values);
    double mean = get_gp_mean(gp, values);

    DEBUG_GROUPING("[%s:%d] Group has %d elements - Group median = %f; group mean = %f\n", __FILE__, __LINE__, gp->size, median, mean);
    return affinity_is_okay(mean, median);
}

static int unlink_gp(group_t *gp)
{
    if (gp == head_gp)
    {
        if (gp->next != NULL)
        {
            gp->next->prev = NULL;
        }
        head_gp = gp->next;
        return 0;
    }

    if (gp == tail_gp)
    {
        if (gp->prev != NULL)
        {
            gp->prev->next = NULL;
        }
        tail_gp = gp->prev;
        return 0;
    }

    gp->prev->next = gp->next;
    gp->next->prev = gp->prev;
    return 0;
}

static void free_gp(group_t *gp)
{
    if (gp->elts != NULL)
    {
        free(gp->elts);
        gp->elts = NULL;
    }
    free(gp);
}

static int add_datapoint_to_group(group_t *gp, int rank, int val, int *values)
{
    DEBUG_GROUPING("[%s:%d] Adding data point %d-%d\n", __FILE__, __LINE__, rank, val)
    // The group's array of values is ordered
    if (add_elt_to_group(gp, rank, values))
    {
        fprintf(stderr, "[%s:%d][ERROR] unable to add new data point to group\n", __FILE__, __LINE__);
        return -1;
    }

    if (group_is_balanced(gp, values))
    {
        // If the group is still balanced, we are done.
        // If not, we remove the continue with the second half of the
        // function which deals with an imbalanced group.
        return 0;
    }

    DEBUG_GROUPING("[%s:%d] Group is unbalanced\n", __FILE__, __LINE__);

    // If the mean and median deviate too much, we split the group: the group
    // is destroyed and we add the data points one by one to the other groups
    // on each side. The algorithm will converge with eventually groups with
    // only a few data points.
    int *data_points = gp->elts;
    if (unlink_gp(gp))
    {
        fprintf(stderr, "[%s:%d][ERROR] unable to unlink group\n", __FILE__, __LINE__);
        return -1;
    }

    // We add points again but in reverse order of what was previously done
    int i;
    for (i = gp->size - 1; i >= 0; i++)
    {
        if (add_datapoint(data_points[i], values))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to add data point while rebalancing group\n", __FILE__, __LINE__);
            return -1;
        }
    }

    free_gp(gp);
}

static int get_gp_sum(group_t *gp, int *values)
{
    int sum = 0;
    int i;

    for (i = 0; i < gp->size; i++)
    {
        sum += values[gp->elts[i]];
    }

    return sum;
}

static group_t *split_group(group_t *gp, int index_split, int *values)
{
    // Create the new group
    group_t *ng = create_group(gp->elts[index_split], values[gp->elts[index_split]], values);
    ng->prev = gp;
    ng->next = gp->next;
    if (gp->next != NULL)
    {
        gp->next->prev = ng;
    }
    gp->next = ng;
    gp->min = values[gp->elts[0]];
    gp->max = values[gp->elts[gp->size - 1]];

    if (tail_gp == gp)
    {
        tail_gp = ng;
    }

    int i;
    for (i = index_split + 1; i < gp->size; i++)
    {
        DEBUG_GROUPING("[%s:%d] Adding %d to new group (value = %d)...\n", __FILE__, __LINE__, gp->elts[i], values[gp->elts[i]]);
        add_elt_to_group(ng, gp->elts[i], values);
    }

    gp->size = index_split;
    gp->max = values[gp->elts[index_split - 1]];
    DEBUG_GROUPING("[%s:%d] Split successful\n", __FILE__, __LINE__);
#if GROUPING_DEBUG
    fprintf(stdout, "[%s:%d] Number of groups: %d\n", __FILE__, __LINE__, count_groups());
#endif

    return ng;
}

static int balance_group_with_new_element(group_t *gp, int rank, int val, int *values)
{
    DEBUG_GROUPING("[%s:%d] Balancing group with new element (rank/value = %d/%d)...\n", __FILE__, __LINE__, rank, val);
    //int vals[gp->size + 1];

    // We calculate the mean for the group + the element
    double sum = (double)get_gp_sum(gp, values);
    sum += val;
    double mean = sum / (gp->size + 1);

    // Now we calculate the median
    DEBUG_GROUPING("[%s:%d] Getting the median for the potential new group...\n", __FILE__, __LINE__);
    double median = get_median_with_additional_point(gp, rank, val, values);
    DEBUG_GROUPING("[%s:%d] Mean: %f; median: %f\n", __FILE__, __LINE__, mean, median);
    if (affinity_is_okay(mean, median))
    {
        add_elt_to_group(gp, rank, values);
    }
    else
    {
        // We figure out where we need to split the group
        int i = 0;
        while (i < gp->size && values[gp->elts[i]] < values[rank])
        {
            i++;
        }

        if (i < gp->size)
        {
            DEBUG_GROUPING("[%s:%d] Splitting group at index %d\n", __FILE__, __LINE__, i);
            group_t *new_group = split_group(gp, i, values);
            add_elt_to_group(new_group, rank, values);
        }
        else
        {
            DEBUG_GROUPING("[%s:%d] Adding new group to the right...\n", __FILE__, __LINE__);
            group_t *new_group = create_group(rank, val, values);
            if (add_group(new_group))
            {
                fprintf(stderr, "[%s:%d][ERROR] unable to add new group to the right of group\n", __FILE__, __LINE__);
                return -1;
            }
        }
    }
    return 0;
}

int add_datapoint(int rank, int *values)
{
    int val = get_value(rank, values);
    group_t *gp = NULL;

    DEBUG_GROUPING("[%s:%d] ***** Adding new data points *****\n", __FILE__, __LINE__);

    // We scan the groups to see which group is the most likely to be suitable
    gp = lookup_group(val);

    if (gp == NULL)
    {
        gp = create_group(rank, val, values);
        if (add_group(gp))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to add group\n", __FILE__, __LINE__);
            return -1;
        }
    }
    else
    {
        DEBUG_GROUPING("[%s:%d] Adding element (val:%d) to existing group (min: %d; max: %d)\n", __FILE__, __LINE__, val, gp->min, gp->max);
        if (balance_group_with_new_element(gp, rank, val, values))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to balance group\n", __FILE__, __LINE__);
            return -1;
        }
    }

    return 0;
}

int grouping_fini()
{
    group_t *ptr = head_gp;
    while (ptr != NULL)
    {
        group_t *temp = ptr;
        free(ptr->elts);
        ptr = ptr->next;
        free(temp);
    }

    head_gp = NULL;
    tail_gp = NULL;
    return 0;
}

int get_groups(group_t **gps, int *num_groups)
{
    group_t *ptr;

    if (head_gp == NULL)
    {
        *gps = NULL;
        *num_groups = 0;
    }

    *gps = head_gp;
    *num_groups = count_groups();

    return 0;
}

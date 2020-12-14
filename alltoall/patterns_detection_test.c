/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdio.h>
#include "pattern.h"

#define MAX_ELTS (50)

typedef struct pattern_test
{
    int n_ranks;
    int n_peers;
} pattern_test_t;

typedef struct pd_test
{
    int s_counts[MAX_ELTS];
    int r_counts[MAX_ELTS];
    int size;
    int expected_spatterns_size;
    int expected_rpatterns_size;
    pattern_test_t expected_spatterns[MAX_ELTS];
    pattern_test_t expected_rpatterns[MAX_ELTS];
} pd_test_t;

static bool _compare_patterns(int *expected_patterns, int size, avPattern_t *p)
{
    if (p == NULL)
    {
        return false;
    }

    int i;

    for (i = 0; i < size; i++)
    {
        avPattern_t *ptr = p;
        while (ptr != NULL)
        {
            if (expected_patterns[i * 2] == ptr->n_ranks && expected_patterns[i * 2 + 1] == ptr->n_peers)
            {
                break;
            }
            ptr = ptr->next;
        }
        if (ptr == NULL)
        {
            return false;
        }
    }
    return true;
}

int patterns_detection_test(void)
{
    pd_test_t tests[] = {
        {
            s_counts : {
                1, 1, 1, 0, 0, 0,
                1, 1, 1, 0, 0, 0,
                1, 1, 1, 0, 0, 0,
                0, 0, 0, 1, 1, 1,
                0, 0, 0, 1, 1, 1,
                0, 0, 0, 1, 1, 1},
            r_counts : {
                1, 1, 1, 0, 0, 0,
                1, 1, 1, 0, 0, 0,
                1, 1, 1, 0, 0, 0,
                0, 0, 0, 1, 1, 1,
                0, 0, 0, 1, 1, 1,
                0, 0, 0, 1, 1, 1},
            size : 6,
            expected_spatterns_size : 1,
            expected_rpatterns_size : 1,
            expected_spatterns : {
                {6, 3}, // 6 ranks are sending to 3 other ranks
            },
            expected_rpatterns : {
                {6, 3}, // 6 ranks are receiving from 3 other ranks
            }
        },
        {
            s_counts : {
                1, 1, 1, 0, 0, 0,
                1, 1, 1, 0, 0, 0,
                1, 1, 1, 0, 0, 0,
                0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 1, 1},
            r_counts : {
                1, 1, 1, 0, 0, 0,
                0, 0, 0, 0, 0, 0,
                0, 1, 0, 0, 0, 0,
                0, 0, 0, 0, 1, 0,
                0, 3, 2, 1, 1, 1,
                0, 0, 0, 0, 0, 0},
            size : 6,
            expected_spatterns_size : 2,
            expected_rpatterns_size : 3,
            expected_spatterns : {
                {3, 3}, // 3 ranks are sending to 3 other ranks
                {1, 2}, // 1 rank is sending to 2 other ranks
            },
            expected_rpatterns : {
                {1, 3}, // 1 rank is receiving from 3 other ranks
                {2, 1}, // 2 ranks are receiving from 1 other ranks
                {1, 5}, // 1 rank is receiving from 3 other ranks
            }
        },
    };

    int i;
    for (i = 0; i < 2; i++)
    {
        fprintf(stdout, "*** Running test %d\n", i);
        avCallPattern_t *call_pattern = extract_call_patterns(i, (int *)(tests[i].s_counts), (int *)(tests[i].r_counts), tests[i].size);
        if (call_pattern == NULL ||
            get_size_patterns(call_pattern->rpatterns) != tests[i].expected_rpatterns_size ||
            get_size_patterns(call_pattern->spatterns) != tests[i].expected_spatterns_size ||
            _compare_patterns((int *)tests[i].expected_spatterns, tests[i].expected_spatterns_size, call_pattern->spatterns) == false ||
            _compare_patterns((int *)tests[i].expected_rpatterns, tests[i].expected_rpatterns_size, call_pattern->rpatterns) == false)
        {
            fprintf(stderr, "Test %d failed\n", i);
        }
        else
        {
            fprintf(stdout, "Test %d succeeded\n", i);
        }
    }
    return 0;
}

int main(int argc, char **argv)
{
    if (patterns_detection_test())
    {
        fprintf(stderr, "[ERROR] compressing array test failed\n");
    }
    else
    {
        fprintf(stdout, "compressing array test succeeded\n");
    }

    return 0;
}

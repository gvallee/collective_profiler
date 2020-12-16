/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdio.h>
#include "logger.h"

#define MAX_ELTS (20)
#define MAX_STRLEN (128)

typedef struct ca_test
{
    int array[MAX_ELTS];
    int size;
    char expected_result[MAX_STRLEN];
} ca_test_t;

static int compress_array_test(void)
{
    ca_test_t tests[] = {
        {
            array : {0, 1, 2, 3, 4, 5, 6},
            size : 7,
            expected_result : "0-6",
        },
        {
            array : {0, 1, 2, 3, 4, 5, 7},
            size : 7,
            expected_result : "0-5, 7",
        },
        {
            array : {0, 2, 3, 4, 5, 6},
            size : 6,
            expected_result : "0, 2-6",
        },
        {
            array : {0, 2, 3, 5, 6, 7, 8},
            size : 7,
            expected_result : "0, 2-3, 5-8",
        },
        {
            array : {0, 1, 2, 3, 5, 6, 7, 8},
            size : 8,
            expected_result : "0-3, 5-8",
        },
        {
            array : {4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 32, 33, 34, 35, 36, 64, 65, 66},
            size : 20,
            expected_result : "4-15, 32-36, 64-66",
        },
    };

    int i;
    for (i = 0; i < 6; i++)
    {
        fprintf(stdout, "*** Running test %d\n", i);
        char *str = compress_int_array(tests[i].array, tests[i].size);

        if (strcmp(str, tests[i].expected_result) != 0)
        {
            fprintf(stderr, "[ERROR] test #%d expected %s but got %s\n", i, tests[i].expected_result, str);
            return 1;
        }
        else
        {

            fprintf(stdout, "*** Test %d successful\n", i);
        }
        free(str);
    }
    return 0;
}

int main(int argc, char **argv)
{
    if (compress_array_test())
    {
        fprintf(stderr, "[ERROR] compressing array test failed\n");
    }
    else
    {
        fprintf(stdout, "compressing array test succeeded\n");
    }

    return 0;
}

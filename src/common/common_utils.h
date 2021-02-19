/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef COLLECTIVE_PROFILER_COMM_UTILS_H
#define COLLECTIVE_PROFILER_COMM_UTILS_H

#include <assert.h>

#include "common_types.h"

static int
get_remainder(int n, int d)
{
    return (n - d * (n / d));
}

#define _asprintf(str, ret, fmt, ...)                               \
    do                                                              \
    {                                                               \
        assert(str == NULL);                                        \
        int __asprintf_size = MAX_STRING_LEN;                       \
        ret = __asprintf_size;                                      \
        while (ret >= __asprintf_size)                              \
        {                                                           \
            if (str == NULL)                                        \
            {                                                       \
                str = (char *)malloc(__asprintf_size);              \
                assert(str);                                        \
            }                                                       \
            else                                                    \
            {                                                       \
                __asprintf_size += MAX_STRING_LEN;                  \
                str = (char *)realloc(str, __asprintf_size);        \
                assert(str);                                        \
            }                                                       \
            ret = snprintf(str, __asprintf_size, fmt, __VA_ARGS__); \
        }                                                           \
    } while (0)

static char *ctx_to_string(int ctx)
{
    char *context;
    switch (ctx)
    {
    case MAIN_CTX:
        context = "main";
        break;

    case SEND_CTX:
        context = "send";
        break;

    case RECV_CTX:
        context = "recv";
        break;

    default:
        context = "main";
        break;
    }
    return context;
}

static int get_job_id()
{
    char *jobid = NULL;
    if (getenv("SLURM_JOB_ID"))
    {
        jobid = getenv("SLURM_JOB_ID");
    }
    else
    {
        if (getenv("LSB_JOBID"))
        {
            jobid = getenv("LSB_JOBID");
        }
        else
        {
            jobid = "0";
        }
    }

    return atoi(jobid);
}

#endif // COLLECTIVE_PROFILER_COMM_UTILS_H
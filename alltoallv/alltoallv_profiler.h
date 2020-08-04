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

#define DEBUG (0)
#define HOSTNAME_LEN 16
#define MAX_FILENAME_LEN (32)
#define MAX_PATH_LEN (128)
#define MAX_STRING_LEN (64)
#define SYNC 0 // Force the ranks to sync after each alltoallv operations to ensure rank 0 does not artifically fall behind

// A few environment variables to control a few things at runtime
#define MSG_SIZE_THRESHOLD_ENVVAR "MSG_SIZE_THRESHOLD" // Name of the environment variable to change the value used to differentiate small and large messages
#define OUTPUT_DIR_ENVVAR "A2A_PROFILING_OUTPUT_DIR"   // Name of the environment variable to specify where output files will be created
#define NUM_CALL_START_PROFILING_ENVVAR "A2A_NUM_CALL_START_PROFILING"
#define LIMIT_ALLTOALLV_CALLS_ENVVAR "A2A_LIMIT_ALLTOALLV_CALLS_ENVVAR"
#define A2A_COMMIT_PROFILER_DATA_AT_ENVVAR "A2A_COMMIT_PROFILER_DATA_AT"
#define A2A_RELEASE_RESOURCES_AFTER_DATA_COMMIT_ENVVAR "A2A_RELEASE_RESOURCES_AFTER_DATA_COMMIT"

#define DEFAULT_MSG_SIZE_THRESHOLD 200     // The default threshold between small and big messages
#define DEFAULT_LIMIT_ALLTOALLV_CALLS (-1) // Maximum number of alltoallv calls that we profile (-1 means no limit)
#define NUM_CALL_START_PROFILING (0)       // During which call do we start profiling? By default, the very first one. Note that once started, DEFAULT_LIMIT_ALLTOALLV_CALLS says when we stop profiling
#define DEFAULT_TRACKED_CALLS (10)

// A few switches to enable/disable a bunch of capabilities

// Note that we check whether it is already set so we can define it while compiling and potentially generate multiple shared libraries

// Switch to enable/disable getting the backtrace safely to get data about the alltoallv caller
#ifndef ENABLE_BACKTRACE
#define ENABLE_BACKTRACE (0)
#endif // ENABLE_BACKTRACE

// Switch to enable/disable the display of raw data (can be very time consuming)
#ifndef ENABLE_RAW_DATA
#define ENABLE_RAW_DATA (0)
#endif // ENABLE_RAW_DATA

// Switch to enable/disable to individual storage of calls' counts. To be used in conjuction with ENABLE_RAW_DATA
#ifndef ENABLE_COMPACT_FORMAT
#define ENABLE_COMPACT_FORMAT (1)
#endif // ENABLE_COMPACT_FORMAT

// Switch to enable/disable timing of alltoallv operations
#ifndef ENABLE_A2A_TIMING
#define ENABLE_A2A_TIMING (0)
#endif // ENABLE_A2A_TIMING

// Switch to enable/disable timing of late arrivals
#ifndef ENABLE_LATE_ARRIVAL_TIMING
#define ENABLE_LATE_ARRIVAL_TIMING (0)
#endif // ENABLE_LATE_ARRIVAL_TIMING

// Switch to enable/disable tracking of the ranks' location
#ifndef ENABLE_LOCATION_TRACKING
#define ENABLE_LOCATION_TRACKING (0)
#endif // ENABLE_LOCATION_TRACKING

// A few switches that are less commonly used by users and that cannot be set a compiling time from the compiler command
#define ENABLE_LIVE_GROUPING (0)         // Switch to enable/disable live grouping (can be very time consuming)
#define ENABLE_POSTMORTEM_GROUPING (0)   // Switch to enable/disable post-mortem grouping analysis (when enabled, data will be saved to a file)
#define ENABLE_MSG_SIZE_ANALYSIS (0)     // Switch to enable/disable live analysis of message size
#define ENABLE_PER_RANK_STATS (0)        // SWitch to enable/disable per-rank data (can be very expensive)
#define ENABLE_VALIDATION (0)            // Switch to enable/disable gathering of extra data for validation. Be carefull when enabling it in addition of other features.
#define ENABLE_PATTERN_DETECTION (0)     // Switch to enable/disable pattern detection using the number of zero counts
#define COMMSIZE_BASED_PATTERNS (0)      // Do we want to differentiate patterns based on the communication size?
#define TRACK_PATTERNS_ON_CALL_BASIS (0) // Do we want to differentiate patterns on a per-call basis

#define MAX_TRACKED_RANKS (1024)

#define VALIDATION_THRESHOLD (1) // The lower, the less validation data

#if DEBUG
#define DEBUG_ALLTOALLV_PROFILING(fmt, ...) \
    fprintf(stdout, "A2A - [%s:%d]" fmt, __FILE__, __LINE__, __VA_ARGS__)
#else
#define DEBUG_ALLTOALLV_PROFILING(fmt, ...) \
    do                                      \
    {                                       \
    } while (0)
#endif // DEBUG

enum
{
    MAIN_CTX = 0,
    SEND_CTX,
    RECV_CTX
};

// Compact way to save send/recv counts of ranks within a single alltoallv call
typedef struct counts_data
{
    int *counters; // the actual counters (i.e., send/recv counts)
    int num_ranks; // The number of ranks having that series of counters
    int max_ranks; // The current size of the ranks array
    int *ranks;    // The list of ranks having that series of counters
} counts_data_t;

// Data type for storing comm size, alltoallv counts, send/recv count, etc
typedef struct avSRCountNode
{
    int size;
    int count; // How many time we detected the pattern; also size of list_calls
    int max_calls;
    int *list_calls; // Which calls produced the pattern
    int comm;
    int sendtype_size;
    int recvtype_size;
    int send_data_size;        // Size of the array of unique series of send counters
    int recv_data_size;        // Size of the array of unique series of recv counters
    counts_data_t **send_data; // Array of unique series of send counters
    counts_data_t **recv_data; // Array of unique series of recv counters
    double *op_exec_times;
    double *late_arrival_timings;
    struct avSRCountNode *next;
} avSRCountNode_t;

typedef struct avTimingsNode
{
    int size;
    double *timings; // Time spent in the alltoallv function
    struct avTimingsNode *next;
} avTimingsNode_t;

typedef struct avPattern
{
    // <n_ranks> ranks send to or receive from <n_peers> other ranks
    int n_ranks;
    int n_peers;
    int n_calls;   // How many alltoallv calls have that pattern
    int comm_size; // Size of the communicator for which the pattern was detected. Not always used.
    struct avPattern *next;
} avPattern_t;

typedef struct avCallPattern
{
    int n_calls;
    int *calls;
    avPattern_t *spatterns;
    avPattern_t *rpatterns;
    struct avCallPattern *next;
} avCallPattern_t;

#define BACKTRACE_LEN
typedef struct caller_info
{
    int n_calls;
    int *calls;
    char *caller;
    struct caller_info *next;
} caller_info_t;

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

#endif // ALLTOALLV_PROFILER_H

/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef ALLTOALLV_PROFILER_H
#define ALLTOALLV_PROFILER_H

#define DEBUG (0)
#define HOSTNAME_LEN 16
#define MAX_FILENAME_LEN (32)
#define MAX_PATH_LEN (128)
#define SYNC 0 // Force the ranks to sync after each alltoallv operations to ensure rank 0 does not artifically fall behind

#define DEFAULT_MSG_SIZE_THRESHOLD 200     // The default threshold between small and big messages
#define DEFAULT_LIMIT_ALLTOALLV_CALLS (-1) // Maximum number of alltoallv calls that we profile (-1 means no limit)
#define NUM_CALL_START_PROFILING (0)       // During which call do we start profiling? By default, the very first one. Note that once started, DEFAULT_LIMIT_ALLTOALLV_CALLS says when we stop profiling

// A few switches to enable/disable a bunch of capabilities
#define ENABLE_LIVE_GROUPING (0)       // Switch to enable/disable live grouping (can be very time consuming)
#define ENABLE_POSTMORTEM_GROUPING (0) // Switch to enable/disable post-mortem grouping analysis (when enabled, data will be saved to a file)
#define ENABLE_MSG_SIZE_ANALYSIS (0)   // Switch to enable/disable live analysis of message size
#define ENABLE_RAW_DATA (0)            // Switch to enable/disable the display of raw data (can be very time consuming)
#define ENABLE_PER_RANK_STATS (0)      // SWitch to enable/disable per-rank data (can be very expensive)
#define ENABLE_TIMING (0)              // Switch to enable/disable timing of various operations
#define ENABLE_VALIDATION (1)          // Switch to enable/disable gathering of extra data for validation. Be carefull when enabling it in addition of other features.

// A few environment variables to control a few things at runtime
#define MSG_SIZE_THRESHOLD_ENVVAR "MSG_SIZE_THRESHOLD" // Name of the environment variable to change the value used to differentiate small and large messages
#define OUTPUT_DIR_ENVVAR "A2A_PROFILING_OUTPUT_DIR"   // Name of the environment variable to specify where output files will be created

#define MAX_TRACKED_CALLS (5)
#define MAX_TRACKED_RANKS (1024)

#define VALIDATION_THRESHOLD (1)

#define DEBUG_ALLTOALLV_PROFILING(fmt, ...)    \
    do                                         \
    {                                          \
        if (DEBUG > 0)                         \
        {                                      \
            fprintf(stdout, fmt, __VA_ARGS__); \
        }                                      \
    } while (0)

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
    int count;                    // How many time we detected the pattern
    int calls[MAX_TRACKED_CALLS]; // Which calls produced the pattern
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
    double *timings;    // Time spent in the alltoallv function
    double *t_arrivals; // Arrival time (used to track late arrival)
    struct avTimingsNode *next;
} avTimingsNode_t;

#endif // ALLTOALLV_PROFILER_H

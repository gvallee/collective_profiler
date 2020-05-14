/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef ALLTOALLV_PROFILER_H
#define ALLTOALLV_PROFILER_H

#define DEBUG 0
#define HOSTNAME_LEN 16
#define MAX_FILENAME_LEN (32)
#define MAX_PATH_LEN (128)
#define SYNC 0							   // Force the ranks to sync after each alltoallv operations to ensure rank 0 does not artifically fall behind

#define DEFAULT_MSG_SIZE_THRESHOLD 200	   // The default threshold between small and big messages
#define DEFAULT_LIMIT_ALLTOALLV_CALLS (-1) // Maximum number of alltoallv calls that we profile (-1 means no limit)
#define NUM_CALL_START_PROFILING (0)	   // During which call do we start profiling? By default, the very first one. Note that once started, DEFAULT_LIMIT_ALLTOALLV_CALLS says when we stop profiling

// A few switches to enable/disable a bunch of capabilities
#define ENABLE_LIVE_GROUPING (0)	   // Switch to enable/disable live grouping (can be very time consuming)
#define POSTMORTEM_GROUPING (0)		   // Switch to enable/disable post-mortem grouping analysis (when enabled, data will be saved to a file)
#define ENABLE_MSG_SIZE_ANALYSIS (0)   // Switch to enable/disable live analysis of message size
#define ENABLE_DISPLAY_OF_RAW_DATA (0) // Switch to enable/disable the display of raw data (can be very time consuming)
#define ENABLE_PER_RANK_STATS (0)	   // SWitch to enable/disable per-rank data (can be very expensive)

// A few environment variables to control a few things at runtime
#define MSG_SIZE_THRESHOLD_ENVVAR "MSG_SIZE_THRESHOLD"
#define OUTPUT_DIR_ENVVAR "A2A_PROFILING_OUTPUT_DIR" // Name of the environment variable to specify where output files will be created

enum
{
    MAIN_CTX = 0,
    SEND_CTX,
    RECV_CTX
};

// Data type for storing comm size, alltoallv counts, send/recv count, etc
typedef struct avSRCountNode
{
	int size;
	int count;
	int comm;
	int sendtype_size;
	int recvtype_size;
	int *send_data;
	int *recv_data;
	double *op_exec_times;
	struct avSRCountNode *next;
} avSRCountNode_t;

typedef struct avTimingsNode
{
	int size;
	double *timings;
	struct avTimingsNode *next;
} avTimingsNode_t;

extern avSRCountNode_t *head;
extern avTimingsNode_t *op_timing_exec_head;
extern avTimingsNode_t *op_timing_exec_tail;

#endif // ALLTOALLV_PROFILER_H
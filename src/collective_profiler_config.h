/*************************************************************************
 * Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#define HOSTNAME_LEN 16
#define MAX_FILENAME_LEN (32)
#define MAX_PATH_LEN (128)
#define MAX_STRING_LEN (64)
#define SYNC 0 // Force the ranks to sync after each collective operations to ensure rank 0 does not artifically fall behind
#define DEFAULT_MSG_SIZE_THRESHOLD 200     // The default threshold between small and big messages

// A few environment variables to control a few things at runtime
#define MSG_SIZE_THRESHOLD_ENVVAR "MSG_SIZE_THRESHOLD" // Name of the environment variable to change the value used to differentiate small and large messages
#define OUTPUT_DIR_ENVVAR "A2A_PROFILING_OUTPUT_DIR"   // Name of the environment variable to specify where output files will be created

#ifndef FORMAT_VERSION
#define FORMAT_VERSION (0)
#endif // FORMAT_VERSION

// A few switches to enable/disable a bunch of capabilities

// Note that we check whether it is already set so we can define it while compiling and potentially generate multiple shared libraries

// Switch to enable/disable getting the backtrace safely to get data about the collective caller
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

// Switch to enable/disable timing of collective operations
#ifndef ENABLE_EXEC_TIMING
#define ENABLE_EXEC_TIMING (0)
#endif // ENABLE_EXEC_TIMING

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

#include "alltoallv/config.h"
#include "alltoall/config.h"
/*************************************************************************
 * Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef _COLLECTIVE_PROFILER_ALLGATHERV_CONFIG_H
#define _COLLECTIVE_PROFILER_ALLGATHERV_CONFIG_H

#define DEBUG (0)

// A few environment variables to control a few things at runtime
#define ALLGATHERV_NUM_CALL_START_PROFILING_ENVVAR "ALLGATHERV_NUM_CALL_START_PROFILING"
#define ALLGATHERV_LIMIT_CALLS_ENVVAR "ALLGATHERV_LIMIT_CALLS_ENVVAR"
#define ALLGATHERV_COMMIT_PROFILER_DATA_AT_ENVVAR "ALLGATHERV_COMMIT_PROFILER_DATA_AT"
#define ALLGATHERV_RELEASE_RESOURCES_AFTER_DATA_COMMIT_ENVVAR "ALLGATHERV_RELEASE_RESOURCES_AFTER_DATA_COMMIT"

#define DEFAULT_LIMIT_ALLGATHERV_CALLS (-1) // Maximum number of alltoallv calls that we profile (-1 means no limit)
#define ALLGATHERV_NUM_CALL_START_PROFILING (0)       // During which call do we start profiling? By default, the very first one. Note that once started, DEFAULT_LIMIT_ALLGATHERV_CALLS says when we stop profiling
#define ALLGATHERV_DEFAULT_TRACKED_CALLS (10)


#endif // _COLLECTIVE_PROFILER_ALLGATHERV_CONFIG_H

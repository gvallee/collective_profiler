/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef _COLLECTIVE_PROFILER_ALLTOALLV_CONFIG_H
#define _COLLECTIVE_PROFILER_ALLTOALLV_CONFIG_H

#define DEBUG (0)

// A few environment variables to control a few things at runtime
#define NUM_CALL_START_PROFILING_ENVVAR "A2A_NUM_CALL_START_PROFILING"
#define LIMIT_ALLTOALLV_CALLS_ENVVAR "A2A_LIMIT_ALLTOALLV_CALLS_ENVVAR"
#define A2A_COMMIT_PROFILER_DATA_AT_ENVVAR "A2A_COMMIT_PROFILER_DATA_AT"
#define A2A_RELEASE_RESOURCES_AFTER_DATA_COMMIT_ENVVAR "A2A_RELEASE_RESOURCES_AFTER_DATA_COMMIT"

#define DEFAULT_LIMIT_ALLTOALLV_CALLS (-1) // Maximum number of alltoallv calls that we profile (-1 means no limit)
#define NUM_CALL_START_PROFILING (0)       // During which call do we start profiling? By default, the very first one. Note that once started, DEFAULT_LIMIT_ALLTOALLV_CALLS says when we stop profiling
#define DEFAULT_TRACKED_CALLS (10)


#endif // _COLLECTIVE_PROFILER_ALLTOALLV_CONFIG_H

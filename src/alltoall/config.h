/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef _COLLECTIVE_PROFILER_ALLTOALL_CONFIG_H
#define _COLLECTIVE_PROFILER_ALLTOALL_CONFIG_H

#define DEBUG (0)

// A few environment variables to control a few things at runtime
#define NUM_CALL_START_PROFILING_ENVVAR "A2A_NUM_CALL_START_PROFILING"
#define LIMIT_ALLTOALL_CALLS_ENVVAR "A2A_LIMIT_ALLTOALL_CALLS_ENVVAR"
#define A2A_COMMIT_PROFILER_DATA_AT_ENVVAR "A2A_COMMIT_PROFILER_DATA_AT"
#define A2A_RELEASE_RESOURCES_AFTER_DATA_COMMIT_ENVVAR "A2A_RELEASE_RESOURCES_AFTER_DATA_COMMIT"

#define DEFAULT_LIMIT_ALLTOALL_CALLS (-1) // Maximum number of alltoall calls that we profile (-1 means no limit)
#define NUM_CALL_START_PROFILING (0)       // During which call do we start profiling? By default, the very first one. Note that once started, DEFAULT_LIMIT_ALLTOALL_CALLS says when we stop profiling
#define DEFAULT_TRACKED_CALLS (10)

#ifndef ASSUME_COUNTS_EQUAL_ALL_RANKS
#define ASSUME_COUNTS_EQUAL_ALL_RANKS (1) // MPI_Alltoall can use different send (or recv) counts on different nodes if send (or recv) type is different, subject to same amount of data sent per rank pair
                                          // but his arrangement is not likely, so assume sendcounts equal on all nodes for performance reasons
#endif

#endif // _COLLECTIVE_PROFILER_ALLTOALL_CONFIG_H
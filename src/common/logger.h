/*************************************************************************
 * Copyright (c) 2020-2022, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <assert.h>
#include <inttypes.h>

#include "collective_profiler_config.h"
#include "common_types.h"

#ifndef LOGGER_H
#define LOGGER_H

#define ENABLE_LOGGER_DEBUGING (0)

#if ENABLE_LOGGER_DEBUGING
#define DEBUG_LOGGER(fmt, ...) \
    fprintf(stdout, "Common - [%s:%d]" fmt, __FILE__, __LINE__, __VA_ARGS__)
#else
#define DEBUG_LOGGER(fmt, ...) \
    do                         \
    {                          \
    } while (0)
#endif // ENABLE_LOGGER_DEBUGGING

#if ENABLE_LOGGER_DEBUGING
#define DEBUG_LOGGER_NOARGS(str) \
    fprintf(stdout, "Common - [%s:%d] %s", __FILE__, __LINE__, str)
#else
#define DEBUG_LOGGER_NOARGS(str) \
    do                         \
    {                          \
    } while (0)
#endif // ENABLE_LOGGER_DEBUGGING


typedef struct logger
{
    char *collective_name;     // Name of the collective, mainly used to enable nice output text.
    int world_size;            // COMM_WORLD size.
    int rank;                  // Rank that is handling the current logger.
    int jobid;                 // Job identifier.
    char *main_filename;       // Path to the main profile file.
    FILE *f;                   // File handle to save general profile data. Other files are created for specific data.
    char *sendcounts_filename; // Path of the send counts profile.
    FILE *sendcounters_fh;     // File handle used to save send counters.
    char *recvcounts_filename; // Path of the receive counts profile.
    FILE *recvcounters_fh;     // File handle used to save recv counters.
    char *senddispls_filename; // Path of the send displacements profile.
    FILE *senddispls_fh;       // File handle used to save send displacements.
    char *recvdispls_filename; // Path of the receive displacements profile.
    FILE *recvdispls_fh;       // File handle used to save recv displacements.
    char *sums_filename;       // Path of the sums profiles.
    FILE *sums_fh;             // File handle used to save data related to amount of data exchanged.
    char *timing_filename;     // Path of the timing profile.
    FILE *timing_fh;           // File handle used to save data related to timing of operations.
    get_full_filename_fn_t get_full_filename;
    uint64_t limit_number_calls;
} logger_t;

extern logger_t *logger_init();
extern void logger_fini(logger_t **l);

/**
 * @brief log_profiling_data goes through all the data gathered during profiling to save it to files.
 * 
 * @param logger Pointer to the logger object that gathers all the data to write to files
 * @param coll_calls Number of collective calls gathered by the logger
 * @param callStart Identifier of the first call being logged
 * @param callsLogged Number of calls being logged
 * @param counters_list List of the collective counters
 * @param displs_list List of the collective displacements
 * @param times_list List of timings associated to the collective executions
 */
extern void log_profiling_data(logger_t *logger, uint64_t coll_calls, uint64_t callStart, uint64_t callsLogged, SRCountNode_t *counters_list, SRDisplNode_t *displs_list, avTimingsNode_t *times_list);
extern void log_timing_data(logger_t *logger, avTimingsNode_t *times_list);
extern int *lookup_rank_counters(int data_size, counts_data_t **data, int rank);
extern int *lookup_rank_displs(int data_size, displs_data_t **data, int rank);

/**
 * get_output_dir checks the environment variable used to specify a output directory.
 * If the environment variable is set, it checks whether the directory exists and if
 * not, tries to create it and return the value of the environment variable.
 * If the environment variable is not set, it returns NULL.
 */
extern char *get_output_dir();

#endif // LOGGER_H
/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/
/******************************************************************************************************
 * Copyright (c) 2020, University College London and Mellanox Technolgies Limited. All rights reserved.
 * - for further contributions 
 ******************************************************************************************************/


#include "logger.h"
#include "alltoall_profiler.h"
#include "grouping.h"
#include "common_utils.h"
#include "common_types.h"
#include "collective_profiler_config.h"


static void _log_data(logger_t *logger, int startcall, int endcall, int ctx, int count, int *calls, int num_counts_data, counts_data_t **counters, int size, int type_size)
{
    int i, j, num = 0;
    FILE *fh = NULL;

    if (counters == NULL)
    {
        // Nothing to log, we exit
        return;
    }

#if ENABLE_PER_RANK_STATS
    int *zeros = (int *)calloc(size, sizeof(int));
    int *sums = (int *)calloc(size, sizeof(int));
    assert(zeros);
    assert(sums);
#endif
#if ENABLE_MSG_SIZE_ANALYSIS
    int *mins = (int *)calloc(size, sizeof(int));
    int *maxs = (int *)calloc(size, sizeof(int));
    int *small_messages = (int *)calloc(size, sizeof(int));
    int msg_size_threshold = DEFAULT_MSG_SIZE_THRESHOLD;

    assert(mins);
    assert(maxs);
    assert(small_messages);

    if (getenv(MSG_SIZE_THRESHOLD_ENVVAR) != NULL)
    {
        msg_size_threshold = atoi(getenv(MSG_SIZE_THRESHOLD_ENVVAR));
    }
#endif

    assert(logger);

    if (logger->f == NULL)
    {
        logger->main_filename = get_full_filename(MAIN_CTX, NULL, logger->rank);
        logger->f = fopen(logger->main_filename, "w");
    }
    assert(logger->f);

#if ENABLE_RAW_DATA || ENABLE_VALIDATION
    switch (ctx)
    {
    case RECV_CTX:
        if (logger->recvcounters_fh == NULL)
        {
            logger->recvcounts_filename = get_full_filename(RECV_CTX, "counters", logger->rank);
            logger->recvcounters_fh = fopen(logger->recvcounts_filename, "w");
        }
        fh = logger->recvcounters_fh;
        break;

    case SEND_CTX:
        if (logger->sendcounters_fh == NULL)
        {
            logger->sendcounts_filename = get_full_filename(SEND_CTX, "counters", logger->rank);
            logger->sendcounters_fh = fopen(logger->sendcounts_filename, "w");
        }
        fh = logger->sendcounters_fh;
        break;

    default:
        fh = logger->f;
        break;
    }

    assert(fh);
    fprintf(fh, "# Raw counters\n\n");
    fprintf(fh, "Number of ranks: %d\n", size);
    fprintf(fh, "Datatype size: %d\n", type_size);
    fprintf(fh, "Alltoall calls %d-%d\n", startcall, endcall - 1); // endcall is one ahead so we substract 1
    char *calls_str = compress_int_array(calls, count);
    fprintf(fh, "Count: %d calls - %s\n", count, calls_str);
    fprintf(fh, "\n\nBEGINNING DATA\n");
    DEBUG_ALLTOALL_PROFILING("Saving counts...\n");
    // Save the compressed version of the data
    int count_data_number, _num_ranks, n;
    for (count_data_number = 0; count_data_number < num_counts_data; count_data_number++)
    {
        DEBUG_ALLTOALL_PROFILING("Number of ranks: %d\n", (counters[count_data_number])->num_ranks);

        char *str = compress_int_array((counters[count_data_number])->ranks, (counters[count_data_number])->num_ranks);
        fprintf(fh, "Rank(s) %s: ", str);
        if (str != NULL)
        {
            free(str);
            str = NULL;
        }

        for (n = 0; n < size; n++)
        {
            fprintf(fh, "%d ", (counters[count_data_number])->counters[n]);
        }
        fprintf(fh, "\n");
    }
    DEBUG_ALLTOALL_PROFILING("Counts saved\n");
    fprintf(fh, "END DATA\n");
#endif

#if ENABLE_PER_RANK_STATS || ENABLE_MSG_SIZE_ANALYSIS
    // Go through the data to gather some stats
    int rank;
    for (rank = 0; rank < size; rank++)
    {
        int *_counters = lookupRankCounters(int data_size, count_data_t *data, rank);
        assert(_counters);
#if ENABLE_MSG_SIZE_ANALYSIS
        mins[i] = _counters[0];
        maxs[i] = _counters[0];
#endif
        int num_counter;
        for (num_counter = 0; num_counter < size; num_counter++)
        {
            sums[rank] += _counters[num];
            if (_counters[num_counter] == 0)
            {
                zeros[rank]++;
            }
#if ENABLE_MSG_SIZE_ANALYSIS
            if (_counters[num_counter] < mins[rank])
            {
                mins[rank] = _counters[num_counter];
            }
            if (maxs[rank] < _counters[num_counter])
            {
                maxs[rank] = _counters[num_counter];
            }
            if ((_counters[num_counter] * type_size) < msg_size_threshold)
            {
                small_messages[rank]++;
            }
#endif
        }
    }
#endif
    fprintf(logger->f, "#### Amount of data per rank\n");
#if ENABLE_PER_RANK_STATS
    for (i = 0; i < size; i++)
    {
        fprintf(logger->f, "Rank %d: %d bytes\n", i, sums[i] * type_size);
    }
#else
    fprintf(logger->f, "Per-rank data is disabled\n");
#endif
    fprintf(logger->f, "\n");

    fprintf(logger->f, "#### Number of zeros\n");
    int total_zeros = 0;
#if ENABLE_PER_RANK_STATS
    for (i = 0; i < size; i++)
    {
        total_zeros += zeros[i];
        double ratio_zeros = zeros[i] * 100 / size;
        fprintf(logger->f, "Rank %d: %d/%d (%f%%) zero(s)\n", i, zeros[i], size, ratio_zeros);
    }
#else
    fprintf(logger->f, "Per-rank data is disabled\n");
#endif
    double ratio_zeros = (total_zeros * 100) / (size * size);
    fprintf(logger->f, "Total: %d/%d (%f%%)\n", total_zeros, size * size, ratio_zeros);
    fprintf(logger->f, "\n");

    fprintf(logger->f, "#### Data size min/max\n");
#if ENABLE_MSG_SIZE_ANALYSIS
    for (i = 0; i < size; i++)
    {
        fprintf(logger->f, "Rank %d: Min = %d bytes; max = %d bytes\n", i, mins[i] * type_size, maxs[i] * type_size);
    }
#else
    fprintf(logger->f, "DISABLED\n");
#endif
    fprintf(logger->f, "\n");

    fprintf(logger->f, "#### Small vs. large messages\n");
#if ENABLE_MSG_SIZE_ANALYSIS
    int total_small_msgs = 0;
    for (i = 0; i < size; i++)
    {
        total_small_msgs += small_messages[i];
        float ratio = small_messages[i] * 100 / size;
        fprintf(logger->f, "Rank %d: %f%% small messages; %f%% large messages\n", i, ratio, 100 - ratio);
    }
    double total_ratio_small_msgs = (total_small_msgs * 100) / (size * size);
    fprintf(logger->f, "Total small messages: %d/%d (%f%%)", total_small_msgs, size * size, total_ratio_small_msgs);
#else
    fprintf(logger->f, "DISABLED\n");
#endif
    fprintf(logger->f, "\n");

    // Group information for the send data (using the sums)
    fprintf(logger->f, "\n#### Grouping based on the total amount per ranks\n\n");
#if ENABLE_POSTMORTEM_GROUPING
    log_sums(logger, ctx, sums, size);
#endif
#if ENABLE_LIVE_GROUPING
    grouping_engine_t *e;
    if (grouping_init(&e))
    {
        fprintf(stderr, "[ERROR] unable to initialize grouping\n");
    }
    else
    {
        for (j = 0; j < size; j++)
        {
            if (add_datapoint(e, j, sums))
            {
                fprintf(stderr, "[ERROR] unable to group send data\n");
                return;
            }
        }
        int num_gps = 0;
        group_t *gps = NULL;
        if (get_groups(e, &gps, &num_gps))
        {
            fprintf(stderr, "[ERROR] unable to get groups\n");
            return;
        }
        log_groups(logger, gps, num_gps);
        grouping_fini(&e);
        fprintf(logger->f, "\n");
    }
#else
    fprintf(logger->f, "DISABLED\n\n");
#endif

#if ENABLE_PER_RANK_STATS
    free(sums);
    free(zeros);
#endif
#if ENABLE_MSG_SIZE_ANALYSIS
    free(mins);
    free(maxs);
    free(small_messages);
#endif
}

static void log_timings(logger_t *logger, int num_call, double *timings, int size)
{
    int j;

    if (logger->timing_fh == NULL)
    {
        // Default filename that we overwrite based on enabled features
        logger->timing_filename = get_full_filename(MAIN_CTX, "timings", logger->rank);
#if ENABLE_A2A_TIMING
        logger->timing_filename = get_full_filename(MAIN_CTX, "a2a-timings", logger->rank);
#endif // ENABLE_A2A_TIMING
#if ENABLE_LATE_ARRIVAL_TIMING
        logger->timing_filename = get_full_filename(MAIN_CTX, "late-arrivals-timings", logger->rank);
#endif // ENABLE_LATE_ARRIVAL_TIMING
        logger->timing_fh = fopen(logger->timing_filename, "w");
    }

    fprintf(logger->timing_fh, "Alltoall call #%d\n", num_call);
    for (j = 0; j < size; j++)
    {
        fprintf(logger->timing_fh, "Rank %d: %f\n", j, timings[j]);
    }
    fprintf(logger->timing_fh, "\n");
}

static void log_data(logger_t *logger, uint64_t startcall, uint64_t endcall, avSRCountNode_t *counters_list, avTimingsNode_t *times_list)
{
    assert(logger);
#if ENABLE_RAW_DATA

    // Display the send/receive counts data
    if (counters_list != NULL)
    {
        avSRCountNode_t *srCountPtr = counters_list;
        if (logger->f == NULL)
        {
            logger->main_filename = get_full_filename(MAIN_CTX, NULL, logger->rank);
            logger->f = fopen(logger->main_filename, "w");
        }
        assert(logger->f);
        fprintf(logger->f, "# Send/recv counts for alltoall operations:\n");
        uint64_t count = 0;
        while (srCountPtr != NULL)
        {
            fprintf(logger->f, "\n## Data set #%" PRIu64 "\n\n", count);
            fprintf(logger->f, "comm size = %d; alltoall calls = %" PRIu64 "\n\n", srCountPtr->size, srCountPtr->count);

            DEBUG_ALLTOALL_PROFILING("Logging alltoall call %" PRIu64 "\n", srCountPtr->count);
            DEBUG_ALLTOALL_PROFILING("Logging send counts\n");
            fprintf(logger->f, "### Data sent per rank - Type size: %d\n\n", srCountPtr->sendtype_size);

            _log_data(logger, startcall, endcall,
                      SEND_CTX, srCountPtr->count, srCountPtr->list_calls,
                      srCountPtr->send_data_size, srCountPtr->send_data, srCountPtr->size, srCountPtr->sendtype_size);

            DEBUG_ALLTOALL_PROFILING("Logging recv counts (number of count series: %d)\n", srCountPtr->recv_data_size);
            fprintf(logger->f, "### Data received per rank - Type size: %d\n\n", srCountPtr->recvtype_size);

            _log_data(logger, startcall, endcall,
                      RECV_CTX, srCountPtr->count, srCountPtr->list_calls,
                      srCountPtr->recv_data_size, srCountPtr->recv_data, srCountPtr->size, srCountPtr->recvtype_size);

            DEBUG_ALLTOALL_PROFILING("alltoall call %" PRIu64 " logged\n", srCountPtr->count);
            srCountPtr = srCountPtr->next;
            count++;
        }
    }
#endif

#if ENABLE_A2A_TIMING || ENABLE_LATE_ARRIVAL_TIMING
    // Handle the timing data
    if (times_list != NULL)
    {
        avTimingsNode_t *tPtr = times_list;
        int i = 0;
        while (tPtr != NULL)
        {
            log_timings(logger, i, tPtr->timings, tPtr->size);
            tPtr = tPtr->next;
            i++;
        }
    }
#endif // ENABLE_A2A_TIMING || ENABLE_LATE_ARRIVAL_TIMING
}

void log_profiling_data(logger_t *logger, uint64_t avCalls, uint64_t avCallStart, uint64_t avCallsLogged, avSRCountNode_t *counters_list, avTimingsNode_t *times_list)
{
    // We log the data most of the time right before unloading our shared
    // library, and it includes the mpirun process. So the logger may be NULL.
    if (logger == NULL)
        return;

    // We check if we actually have data to save or not
    if (avCallsLogged > 0 && (counters_list != NULL || times_list != NULL))
    {
        if (logger->f == NULL)
        {
            logger->main_filename = get_full_filename(MAIN_CTX, NULL, logger->rank);
            logger->f = fopen(logger->main_filename, "w");
        }
        fprintf(logger->f, "# Summary\n");
        fprintf(logger->f, "COMM_WORLD size: %d\n", logger->world_size);
        fprintf(logger->f, "Total number of alltoall calls = %" PRIu64 " (limit is %d; -1 means no limit)\n", avCalls, DEFAULT_LIMIT_ALLTOALL_CALLS);
        //fprintf(logger->f, "Alltoall call range: [%d-%d]\n\n", avCallStart, avCallStart + avCallsLogged - 1); // Note that we substract 1 because we are 0 indexed
        log_data(logger, avCallStart, avCallStart + avCallsLogged, counters_list, times_list);
    }
}
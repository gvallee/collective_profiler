/*************************************************************************
 * Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <sys/stat.h>
#include <errno.h>
#include <sys/types.h>
#include <dirent.h>

#include "logger.h"
#include "grouping.h"
#include "format.h"
#include "comm.h"
#include "timings.h"
#include "backtrace.h"
#include "location.h"

char *get_output_dir()
{
    char *dirpath = NULL;
    if (getenv(OUTPUT_DIR_ENVVAR))
	{
		dirpath = getenv(OUTPUT_DIR_ENVVAR);
		// if the output directory does not exist, we create it
		DIR *dir = opendir(dirpath);
		if (dir == NULL && errno == ENOENT)
		{
			// The directory does not exist, we try to create it.
            // We do not check the return code because this is best
            // effort the value of the environment variable is set
            // by the user.
			mkdir(dirpath, 0744);
		}
	}
    return dirpath;
}

void log_groups(logger_t *logger, group_t *gps, int num_gps)
{
    group_t *ptr = gps;

    assert(logger);
    assert(logger->f);

    fprintf(logger->f, "Number of groups: %d\n\n", num_gps);
    int i;
    for (i = 0; i < num_gps; i++)
    {
        fprintf(logger->f, "#### Group %d\n", i);
        fprintf(logger->f, "Number of ranks: %d\n", ptr->size);
        fprintf(logger->f, "Smaller data size: %d\n", ptr->min);
        fprintf(logger->f, "Bigger data size: %d\n", ptr->max);
        fprintf(logger->f, "Ranks: ");
        int i;
        for (i = 0; i < ptr->size; i++)
        {
            fprintf(logger->f, "%d ", ptr->elts[i]);
        }
        fprintf(logger->f, "\n");
        i++;
        ptr = ptr->next;
    }
}

static void log_sums(logger_t *logger, int ctx, int *sums, int size)
{
    int i;

    assert(logger);

    if (logger->sums_fh == NULL)
    {
        logger->sums_filename = logger->get_full_filename(MAIN_CTX, "sums", logger->jobid, logger->rank);
        logger->sums_fh = fopen(logger->sums_filename, "w");
    }

    fprintf(logger->sums_fh, "# Rank\tAmount of data (bytes)\n");
    for (i = 0; i < size; i++)
    {
        fprintf(logger->sums_fh, "%d\t%d\n", i, sums[i]);
    }
}

int *lookup_rank_counters(int data_size, counts_data_t **data, int rank)
{
    assert(data);
    DEBUG_LOGGER("Looking up counts for rank %d (%d data elements to scan)\n", rank, data_size);
    int i, j;
    for (i = 0; i < data_size; i++)
    {
        assert(data[i]);
        DEBUG_LOGGER("Pattern %d has %d ranks associated to it\n", i, data[i]->num_ranks);
        for (j = 0; j < data[i]->num_ranks; j++)
        {
            assert(data[i]->ranks);
            DEBUG_LOGGER("Scan previous counts for rank %d\n", data[i]->ranks[j]);
            if (rank == data[i]->ranks[j])
            {
                return data[i]->counters;
            }
        }
    }
    DEBUG_LOGGER("Could not find data for rank %d\n", rank);
    return NULL;
}

static void _log_data(logger_t *logger,
                      uint64_t startcall,
                      uint64_t endcall,
                      int ctx,
                      uint64_t count,
                      uint64_t *calls,
                      uint64_t num_counts_data,
                      counts_data_t **counters,
                      int size,
                      int rank_vec_len,
                      int type_size)
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
        logger->main_filename = logger->get_full_filename(MAIN_CTX, NULL, logger->jobid, logger->rank);
        logger->f = fopen(logger->main_filename, "w");
    }
    assert(logger->f);

#if ENABLE_RAW_DATA || ENABLE_VALIDATION
    switch (ctx)
    {
    case RECV_CTX:
        if (logger->recvcounters_fh == NULL)
        {
            logger->recvcounts_filename = logger->get_full_filename(RECV_CTX, "counters", logger->jobid, logger->rank);
            logger->recvcounters_fh = fopen(logger->recvcounts_filename, "w");
        }
        fh = logger->recvcounters_fh;
        break;

    case SEND_CTX:
        if (logger->sendcounters_fh == NULL)
        {
            logger->sendcounts_filename = logger->get_full_filename(SEND_CTX, "counters", logger->jobid, logger->rank);
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
    fprintf(fh, "%s calls %"PRIu64"-%"PRIu64"\n", logger->collective_name, startcall, endcall - 1); // endcall is one ahead so we substract 1
    char *calls_str = compress_uint64_array(calls, count, 1);
    fprintf(fh, "Count: %"PRIu64" calls - %s\n", count, calls_str);
    fprintf(fh, "\n\nBEGINNING DATA\n");
    DEBUG_LOGGER_NOARGS("Saving counts...\n");
    // Save the compressed version of the data
    int count_data_number, _num_ranks, n;
    for (count_data_number = 0; count_data_number < num_counts_data; count_data_number++)
    {
        DEBUG_LOGGER("Number of ranks: %d\n", (counters[count_data_number])->num_ranks);

        char *str = compress_int_array((counters[count_data_number])->ranks, (counters[count_data_number])->num_ranks, 1);
        fprintf(fh, "Rank(s) %s: ", str);
        if (str != NULL)
        {
            free(str);
            str = NULL;
        }

        for (n = 0; n < rank_vec_len; n++)
        {
            fprintf(fh, "%d ", (counters[count_data_number])->counters[n]);
        }
        fprintf(fh, "\n");
    }
    DEBUG_LOGGER_NOARGS("Counts saved\n");
    fprintf(fh, "END DATA\n");
#endif
//TO DO check the rest of this function for alltoallv to alltoall conversion
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
        logger->timing_filename = logger->get_full_filename(MAIN_CTX, "timings", logger->jobid, logger->rank);
#if ENABLE_EXEC_TIMING
        logger->timing_filename = logger->get_full_filename(MAIN_CTX, "a2a-timings", logger->jobid, logger->rank);
#endif // ENABLE_EXEC_TIMING
#if ENABLE_LATE_ARRIVAL_TIMING
        logger->timing_filename = logger->get_full_filename(MAIN_CTX, "late-arrivals-timings", logger->jobid, logger->rank);
#endif // ENABLE_LATE_ARRIVAL_TIMING
        logger->timing_fh = fopen(logger->timing_filename, "w");
    }

    fprintf(logger->timing_fh, "%s call #%d\n", logger->collective_name, num_call);
    for (j = 0; j < size; j++)
    {
        fprintf(logger->timing_fh, "Rank %d: %f\n", j, timings[j]);
    }
    fprintf(logger->timing_fh, "\n");
}
// called with log_data(logger, avCallStart, avCallStart + avCallsLogged, counters_list, times_list);
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
            logger->main_filename = logger->get_full_filename(MAIN_CTX, NULL, logger->jobid, logger->rank);
            logger->f = fopen(logger->main_filename, "w");
        }
        assert(logger->f);
        fprintf(logger->f, "# Send/recv counts for %s operations:\n", logger->collective_name);
        uint64_t count = 0;
        while (srCountPtr != NULL)
        {
            fprintf(logger->f, "\n## Data set #%" PRIu64 "\n\n", count);
            fprintf(logger->f,
                    "comm size = %d; %s calls = %" PRIu64 "\n\n",
                    srCountPtr->size,
                    logger->collective_name,
                    srCountPtr->count);

            DEBUG_LOGGER("Logging %s call %" PRIu64 "\n", logger->collective_name, srCountPtr->count);
            DEBUG_LOGGER_NOARGS("Logging send counts\n");
            fprintf(logger->f, "### Data sent per rank - Type size: %d\n\n", srCountPtr->sendtype_size);

            _log_data(logger, startcall, endcall,
                      SEND_CTX, srCountPtr->count, srCountPtr->list_calls,
                      srCountPtr->send_data_size, srCountPtr->send_data, srCountPtr->size, srCountPtr->rank_vec_len, srCountPtr->sendtype_size);

            DEBUG_LOGGER("Logging recv counts (number of count series: %d)\n", srCountPtr->recv_data_size);
            fprintf(logger->f, "### Data received per rank - Type size: %d\n\n", srCountPtr->recvtype_size);

            _log_data(logger, startcall, endcall,
                      RECV_CTX, srCountPtr->count, srCountPtr->list_calls,
                      srCountPtr->recv_data_size, srCountPtr->recv_data, srCountPtr->size, srCountPtr->rank_vec_len, srCountPtr->recvtype_size);

            DEBUG_LOGGER("%s call %" PRIu64 " logged\n", logger->collective_name, srCountPtr->count);
            srCountPtr = srCountPtr->next;
            count++;
        }
    }
#endif

#if ENABLE_EXEC_TIMING || ENABLE_LATE_ARRIVAL_TIMING
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
#endif // ENABLE_EXEC_TIMING || ENABLE_LATE_ARRIVAL_TIMING
}

logger_t *logger_init(int jobid, int world_rank, int world_size, logger_config_t *cfg)
{
    if (cfg == NULL)
    {
        fprintf(stderr, "logger configuration is undefined\n");
        return NULL;
    }

    if (cfg->get_full_filename == NULL || strlen(cfg->collective_name) == 0)
    {
        fprintf(stderr, "invalid logger configuration\n");
        return NULL;
    }

    char filename[128];
    logger_t *l = calloc(1, sizeof(logger_t));
    if (l == NULL)
    {
        return NULL;
    }

    l->rank = world_rank;
    l->world_size = world_size;
    l->f = NULL;
    l->main_filename = NULL;
    l->recvcounters_fh = NULL;
    l->recvcounts_filename = NULL;
    l->sendcounters_fh = NULL;
    l->sendcounts_filename = NULL;
    l->sums_fh = NULL;
    l->sums_filename = NULL;
    l->timing_fh = NULL;
    l->timing_filename = NULL;

    l->get_full_filename = cfg->get_full_filename;
    l->collective_name = strdup(cfg->collective_name);

    return l;
}

void logger_fini(logger_t **l)
{
    int rc;

    // Calling the function multiple times is allowed
    if (l == NULL || *l == NULL)
        return;

    rc = release_time_loggers();
    if (rc)
    {
        fprintf(stderr, "fini_time_tracking() failed: %d\n", rc);
    }

    rc = release_backtrace_loggers();
    if (rc)
    {
        fprintf(stderr, "release_backtrace_loggers() failed: %d\n", rc);
    }

    rc = release_location_loggers();
    if (rc)
    {
        fprintf(stderr, "release_location_loggers() failed: %d\n", rc);
    }

    rc = release_comm_data((*l)->collective_name, (*l)->rank);
    if (rc)
    {
        fprintf(stderr, "release_comm_data() failed: %d\n", rc);
    }

    if (l != NULL)
    {
        if (*l != NULL)
        {
            if ((*l)->f)
                fclose((*l)->f);
            if ((*l)->main_filename)
                free((*l)->main_filename);
            if ((*l)->sendcounters_fh)
                fclose((*l)->sendcounters_fh);
            if ((*l)->sendcounts_filename)
                free((*l)->sendcounts_filename);
            if ((*l)->recvcounters_fh)
                fclose((*l)->recvcounters_fh);
            if ((*l)->recvcounts_filename)
                free((*l)->recvcounts_filename);
            if ((*l)->timing_fh)
                fclose((*l)->timing_fh);
            if ((*l)->timing_filename)
                free((*l)->timing_filename);
            if ((*l)->sums_fh)
                fclose((*l)->sums_fh);
            if ((*l)->sums_filename)
                free((*l)->sums_filename);
            if ((*l)->collective_name)
                free((*l)->collective_name);
            free(*l);
            *l = NULL;
        }
    }
}

void log_timing_data(logger_t *logger, avTimingsNode_t *times_list)
{
    avTimingsNode_t *tPtr;
    int i;

    // Handle the timing data
    tPtr = times_list;
    i = 0;
    while (tPtr != NULL)
    {
        log_timings(logger, i, tPtr->timings, tPtr->size);
        tPtr = tPtr->next;
        i++;
    }
}
// called with log_profiling_data(logger, avCalls, avCallStart, avCallsLogged, head, op_timing_exec_head); so counters_list = head, which is global var in mpi_alltoall.c
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
            logger->main_filename = logger->get_full_filename(MAIN_CTX, NULL, logger->jobid, logger->rank);
            logger->f = fopen(logger->main_filename, "w");
        }
        fprintf(logger->f, "# Summary\n");
        fprintf(logger->f, "COMM_WORLD size: %d\n", logger->world_size);
        fprintf(logger->f,
                "Total number of %s calls = %" PRIu64 " (limit is %" PRIu64 "; -1 means no limit)\n",
                logger->collective_name,
                avCalls,
                logger->limit_number_calls);
        //fprintf(logger->f, "%s call range: [%d-%d]\n\n", logger->collective_name, avCallStart, avCallStart + avCallsLogged - 1); // Note that we substract 1 because we are 0 indexed
        log_data(logger, avCallStart, avCallStart + avCallsLogged, counters_list, times_list);
    }
}
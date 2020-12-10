/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include "logger.h"
#include "alltoall_profiler.h"
#include "grouping.h"

static char *ctx_to_string(int ctx)
{
    char *context;
    switch (ctx)
    {
    case MAIN_CTX:
        context = "main";
        break;

    case SEND_CTX:
        context = "send";
        break;

    case RECV_CTX:
        context = "recv";
        break;

    default:
        context = "main";
        break;
    }
    return context;
}

static int get_job_id()
{
    char *jobid = NULL;
    if (getenv("SLURM_JOB_ID"))
    {
        jobid = getenv("SLURM_JOB_ID");
    }
    else
    {
        if (getenv("LSB_JOBID"))
        {
            jobid = getenv("LSB_JOBID");
        }
        else
        {
            jobid = "0";
        }
    }

    return atoi(jobid);
}

static char *get_full_filename(int ctxt, char *id, int world_rank)
{
    char *filename = NULL;
    char *dir = NULL;
    int size;

    int jobid = get_job_id();

    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        dir = getenv(OUTPUT_DIR_ENVVAR);
    }

    if (ctxt == MAIN_CTX)
    {
        if (id == NULL)
        {
            _asprintf(filename, size, "profile_alltoall_job%d.rank%d.md", jobid, world_rank);
            assert(size > 0);
        }
        else
        {
            _asprintf(filename, size, "%s.job%d.rank%d.md", id, jobid, world_rank);
            assert(size > 0);
        }
    }
    else
    {
        char *context = ctx_to_string(ctxt);
        _asprintf(filename, size, "%s-%s.job%d.rank%d.txt", context, id, jobid, world_rank);
        assert(size > 0);
    }

    if (dir != NULL)
    {
        char *path = NULL;
        _asprintf(path, size, "%s/%s", dir, filename);
        assert(size > 0);
        free(filename);
        return path;
    }

    return filename;
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
        logger->sums_filename = get_full_filename(MAIN_CTX, "sums", logger->rank);
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
    DEBUG_ALLTOALL_PROFILING("Looking up counts for rank %d (%d data elements to scan)\n", rank, data_size);
    int i, j;
    for (i = 0; i < data_size; i++)
    {
        assert(data[i]);
        DEBUG_ALLTOALL_PROFILING("Pattern %d has %d ranks associated to it\n", i, data[i]->num_ranks);
        for (j = 0; j < data[i]->num_ranks; j++)
        {
            assert(data[i]->ranks);
            DEBUG_ALLTOALL_PROFILING("Scan previous counts for rank %d\n", data[i]->ranks[j]);
            if (rank == data[i]->ranks[j])
            {
                return data[i]->counters;
            }
        }
    }
    DEBUG_ALLTOALL_PROFILING("Could not find data for rank %d\n", rank);
    return NULL;
}

static char *add_range(char *str, int start, int end)
{
    int size;
    if (str == NULL)
    {
        size = MAX_STRING_LEN;
    }
    else
    {
        size = strlen(str) + (MAX_STRING_LEN - get_remainder(strlen(str), MAX_STRING_LEN));
    }
    int ret = size;

    if (str == NULL)
    {
        str = (char *)malloc(size * sizeof(char));
        assert(str);
        while (ret >= size)
        {
            ret = snprintf(str, size, "%d-%d", start, end);
            if (ret < 0)
            {
                fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
                return NULL;
            }
            if (ret >= size)
            {
                // truncated result, increasing the size of the buffer and trying again
                size = size * 2;
                str = (char *)realloc(str, size);
                assert(str);
            }
        }
        return str;
    }
    else
    {
        // We make sure we do not get a truncated result
        char *s = NULL;
        while (ret >= size)
        {
            if (s == NULL)
            {
                s = (char *)malloc(size * sizeof(char));
                assert(s);
            }
            else
            {
                // truncated result, increasing the size of the buffer and trying again
                size = size * 2;
                s = (char *)realloc(s, size);
                assert(s);
            }
            ret = snprintf(s, size, "%s, %d-%d", str, start, end);
            if (ret < 0)
            {
                fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
                return NULL;
            }
        }

        if (s != NULL)
        {
            if (str != NULL)
            {
                free(str);
            }
            str = s;
        }

        return str;
    }
}

static char *add_singleton(char *str, int n)
{

    int size;
    int rc;
    if (str == NULL)
    {
        size = MAX_STRING_LEN;
    }
    else
    {
        size = strlen(str) + (MAX_STRING_LEN - get_remainder(strlen(str), MAX_STRING_LEN));
    }
    int ret = size;
    if (str == NULL)
    {
        str = (char *)malloc(size * sizeof(char));
        assert(str);
        rc = sprintf(str, "%d", n);
        assert(rc <= size);
        return str;
    }

    // We make sure we do not get a truncated result
    char *s = NULL;
    while (ret >= size)
    {
        if (s == NULL)
        {
            s = (char *)malloc(size * sizeof(char));
            assert(s);
        }
        else
        {
            // truncated result, increasing the size of the buffer and trying again
            size = size * 2;
            s = (char *)realloc(s, size);
            assert(s);
        }
        ret = snprintf(s, size, "%s, %d", str, n);
        if (ret < 0)
        {
            fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
            return NULL;
        }
    }

    if (s != NULL)
    {
        if (str != NULL)
        {
            free(str);
        }
        str = s;
    }

    return str;
}

char *compress_int_array(int *array, int size)
{
    int i, start;
    char *compressedRep = NULL;

#if DEBUG
    fprintf(stderr, "Compressing:");
    for (i = 0; i < size; i++)
    {
        fprintf(stderr, " %d", array[i]);
    }
    fprintf(stderr, "\n");
#endif // DEBUG

    for (i = 0; i < size; i++)
    {
        start = i;
        while (i + 1 < size && array[i] + 1 == array[i + 1])
        {
            i++;
        }
        if (i != start)
        {
            // We found a range
            compressedRep = add_range(compressedRep, array[start], array[i]);
        }
        else
        {
            // We found a singleton
            compressedRep = add_singleton(compressedRep, array[i]);
        }
    }
#if DEBUG
    fprintf(stderr, "Compressed version is: %s\n", compressedRep);
#endif // DEBUG
    return compressedRep;
}

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

logger_t *logger_init(int world_rank, int world_size)
{
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

    return l;
}

void logger_fini(logger_t **l)
{
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
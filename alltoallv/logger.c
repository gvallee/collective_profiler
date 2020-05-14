/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include "logger.h"
#include "alltoallv_profiler.h"
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

static char *get_full_filename(int ctxt, char *id)
{
    char *filename = malloc(MAX_FILENAME_LEN * sizeof(char));
    char *dir = NULL;

    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        dir = getenv(OUTPUT_DIR_ENVVAR);
    }

    if (ctxt == MAIN_CTX)
    {
        sprintf(filename, "profile_alltoallv.pid%d.md", getpid());
    }
    else
    {
        char *context = ctx_to_string(ctxt);
        sprintf(filename, "%s-%s.pid%d.txt", context, id, getpid());
    }

    if (dir != NULL)
    {
        char *path = malloc(MAX_PATH_LEN * sizeof(char));
        sprintf(path, "%s/%s", dir, filename);
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

static FILE *open_log_file(int ctxt, char *id)
{
    FILE *fp = NULL;
    char *path;

    path = get_full_filename(ctxt, id);
    fp = fopen(path, "w");
    free(path);
    return fp;
}

static void _log_data(logger_t *logger, int ctx, int *buf, int size, int type_size)
{
    int i, j, num = 0;
    FILE *fh;

    int *zeros = (int *)calloc(size, sizeof(int));
    int *sums = (int *)calloc(size, sizeof(int));
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
    assert(zeros);
    assert(sums);

#if ENABLE_RAW_DATA
    switch (ctx)
    {
    case RECV_CTX:
        fh = logger->recvcounters_fh;
        break;

    case SEND_CTX:
        fh = logger->sendcounters_fh;
        break;

    default:
        fh = logger->f;
        break;
    }

    fprintf(fh, "### Raw counters\n");
#endif

    for (i = 0; i < size; i++)
    {
#if ENABLE_MSG_SIZE_ANALYSIS
        mins[i] = buf[num];
        maxs[i] = buf[num];
#endif
        for (j = 0; j < size; j++)
        {
            sums[i] += buf[num];
            if (buf[num] == 0)
            {
                zeros[i]++;
            }
#if ENABLE_MSG_SIZE_ANALYSIS
            if (buf[num] < mins[i])
            {
                mins[i] = buf[num];
            }
            if (maxs[i] < buf[num])
            {
                maxs[i] = buf[num];
            }
            if ((buf[num] * type_size) < msg_size_threshold)
            {
                small_messages[i]++;
            }
#endif
#if ENABLE_RAW_DATA
            fprintf(fh, "%d ", buf[num]);
#endif
            num++;
        }
#if ENABLE_RAW_DATA
        fprintf(fh, "\n");
#endif
    }
    fprintf(logger->f, "\n");

    fprintf(logger->f, "### Amount of data per rank\n");
#if ENABLE_PER_RANK_STATS
    for (i = 0; i < size; i++)
    {
        fprintf(logger->f, "Rank %d: %d bytes\n", i, sums[i] * type_size);
    }
#else
    fprintf(logger->f, "Per-rank data is disabled\n");
#endif
    fprintf(logger->f, "\n");

    fprintf(logger->f, "### Number of zeros\n");
    int total_zeros = 0;
    for (i = 0; i < size; i++)
    {
        total_zeros += zeros[i];
        double ratio_zeros = zeros[i] * 100 / size;
#if ENABLE_PER_RANK_STATS
        fprintf(logger->f, "Rank %d: %d/%d (%f%%) zero(s)\n", i, zeros[i], size, ratio_zeros);
    }
#else
    }
    fprintf(logger->f, "Per-rank data is disabled\n");
#endif
    double ratio_zeros = (total_zeros * 100) / (size * size);
    fprintf(logger->f, "Total: %d/%d (%f%%)\n", total_zeros, size * size, ratio_zeros);
    fprintf(logger->f, "\n");

    fprintf(logger->f, "### Data size min/max\n");
#if ENABLE_MSG_SIZE_ANALYSIS
    for (i = 0; i < size; i++)
    {
        fprintf(logger->f, "Rank %d: Min = %d bytes; max = %d bytes\n", i, mins[i] * type_size, maxs[i] * type_size);
    }
#else
    fprintf(logger->f, "DISABLED\n");
#endif
    fprintf(logger->f, "\n");

    fprintf(logger->f, "### Small vs. large messages\n");
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
    fprintf(logger->f, "\n### Grouping based on the total amount per ranks\n\n");
#if POSTMORTEM_GROUPING
    char *filename = save_sums(ctx, sums, size);
    fprintf(logger->f, "Data saved in %s for post-mortem analysis\n", filename);
    free(filename);
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

    free(sums);
    free(zeros);
#if ENABLE_MSG_SIZE_ANALYSIS
    free(mins);
    free(maxs);
    free(small_messages);
#endif
}

static void log_data(logger_t *logger)
{
    int i;
    avSRCountNode_t *srCountPtr;
    avTimingsNode_t *tPtr;

    // Display the send/receive counts data
    srCountPtr = head;
    fprintf(logger->f, "# Send/recv counts for alltoallv operations:\n");
    while (srCountPtr != NULL)
    {
        fprintf(logger->f, "comm size = %d, alltoallv calls = %d\n\n", srCountPtr->size, srCountPtr->count);

        fprintf(logger->f, "## Data sent per rank - Type size: %d\n\n", srCountPtr->sendtype_size);
        _log_data(logger, SEND_CTX, srCountPtr->send_data, srCountPtr->size, srCountPtr->sendtype_size);
        fprintf(logger->f, "## Data received per rank - Type size: %d\n\n", srCountPtr->recvtype_size);
        _log_data(logger, RECV_CTX, srCountPtr->recv_data, srCountPtr->size, srCountPtr->recvtype_size);
        srCountPtr = srCountPtr->next;
    }

    // Display the timing data
    tPtr = op_timing_exec_head;
    i = 0;
    fprintf(logger->f, "# Execution times of Alltoallv operations");
    while (tPtr != NULL)
    {
        fprintf(logger->f, "## Alltoallv call #%d\n", i);
#if ENABLE_PER_RANK_STATS
        int j;
        for (j = 0; j < tPtr->size; j++)
        {
            fprintf(logger->f, "Rank %d: %f\n", i, tPtr->timings[i]);
        }
#else
        fprintf(logger->f, "Per-rank data is disabled\n");
#endif
        fprintf(logger->f, "\n");
        tPtr = tPtr->next;
        i++;
    }
}

logger_t *logger_init()
{
    char filename[128];
    logger_t *l = calloc(1, sizeof(logger_t));
    if (l == NULL)
    {
        return l;
    }

    l->f = open_log_file(MAIN_CTX, NULL);
#if ENABLE_RAW_DATA
    l->recvcounters_fh = open_log_file(RECV_CTX, "counters");
    l->sendcounters_fh = open_log_file(SEND_CTX, "counters");
#endif

    return l;
}

void logger_fini(logger_t **l)
{
    if (l != NULL)
    {
        if (*l != NULL)
        {
            if ((*l)->f != NULL)
            {
                fclose((*l)->f);
            }
            if ((*l)->sendcounters_fh)
            {
                fclose((*l)->sendcounters_fh);
            }
            if ((*l)->recvcounters_fh)
            {
                fclose((*l)->recvcounters_fh);
            }
            free(*l);
            *l = NULL;
        }
    }
}

int save_counter(int ctx, int *conters)
{
    return 0;
}

void log_profiling_data(logger_t *logger, int avCalls, int avCallStart, int avCallsLogged)
{
    assert(logger);
    if (logger->f != NULL)
    {
        fprintf(logger->f, "# Summary\n");
        fprintf(logger->f, "Total number of alltoallv calls = %d (limit is %d; -1 means no limit)\n\n", avCalls, DEFAULT_LIMIT_ALLTOALLV_CALLS);
        fprintf(logger->f, "Alltoallv call range: [%d-%d]\n", avCallStart, avCallStart + avCallsLogged);
        log_data(logger);
    }
}
/*************************************************************************
 * Copyright (c) 2020-2022, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <sys/stat.h>
#include <errno.h>
#include <sys/types.h>
#include <dirent.h>
#include <assert.h>

#include "logger.h"
#include "grouping.h"
#include "format.h"

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

int log_counts(logger_t *logger,
                FILE *fh,
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
    fprintf(fh, "%s calls %" PRIu64 "-%" PRIu64 "\n", logger->collective_name, startcall, endcall - 1); // endcall is one ahead so we substract 1
    char *calls_str = compress_uint64_array(calls, count, 1);
    fprintf(fh, "Count: %" PRIu64 " calls - %s\n", count, calls_str);
    fprintf(fh, "\n\nBEGINNING DATA\n");
    DEBUG_LOGGER_NOARGS("Saving counts...\n");
    // Save the compressed version of the data
    int count_data_number, n;
    for (count_data_number = 0; count_data_number < num_counts_data; count_data_number++)
    {
        DEBUG_LOGGER("Number of ranks: %d\n", (counters[count_data_number])->num_ranks);

        char *str = compress_int_array((counters[count_data_number])->ranks, (counters[count_data_number])->num_ranks, 1);
        assert(str);
        fprintf(fh, "Rank(s) %s: ", str);
        free(str);
        str = NULL;

        for (n = 0; n < rank_vec_len; n++)
        {
            fprintf(fh, "%d ", (counters[count_data_number])->counters[n]);
        }
        fprintf(fh, "\n");
    }
    DEBUG_LOGGER_NOARGS("Counts saved\n");
    fprintf(fh, "END DATA\n");
    return 0;
}

/*************************************************************************
 * Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

// This file gathers all the code required to actually handle profile data for collective displacements.
// Note that a logger is compiled specifically for the context. Practically, it means that logger.c
// is compiled with specific arguments to generate an object file specific for the logging of displacements.
// That specific object must be used in conjunction with the object file generated from this file,
// creating a fully functional logger for collective displacements.

#include <sys/stat.h>
#include <errno.h>
#include <sys/types.h>
#include <dirent.h>
#include <assert.h>

#include "logger.h"
#include "grouping.h"
#include "format.h"

int log_displs(logger_t *logger,
               uint64_t startcall,
               uint64_t endcall,
               int ctx,
               uint64_t count,
               uint64_t *calls,
               uint64_t num_displs_data,
               displs_data_t **displs,
               int size,
               int rank_vec_len,
               int type_size)
{
    FILE *fh = NULL;
    switch (ctx)
    {
    case RECV_CTX:
        if (logger->recvdispls_fh == NULL)
        {
            logger->recvdispls_filename = logger->get_full_filename(RECV_CTX, "counters", logger->jobid, logger->rank);
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
    fprintf(fh, "# Raw displacements\n\n");
    fprintf(fh, "Number of ranks: %d\n", size);
    fprintf(fh, "Datatype size: %d\n", type_size);
    fprintf(fh, "%s calls %" PRIu64 "-%" PRIu64 "\n", logger->collective_name, startcall, endcall - 1); // endcall is one ahead so we substract 1
    char *calls_str = compress_uint64_array(calls, count, 1);
    fprintf(fh, "Count: %" PRIu64 " calls - %s\n", count, calls_str);
    fprintf(fh, "\n\nBEGINNING DATA\n");
    DEBUG_LOGGER_NOARGS("Saving displacements...\n");
    // Save the compressed version of the data
    int count_data_number, n;
    for (count_data_number = 0; count_data_number < num_displs_data; count_data_number++)
    {
        DEBUG_LOGGER("Number of ranks: %d\n", (counters[count_data_number])->num_ranks);

        char *str = compress_int_array((displs[count_data_number])->ranks, (displs[count_data_number])->num_ranks, 1);
        assert(str);
        fprintf(fh, "Rank(s) %s: ", str);
        free(str);
        str = NULL;

        for (n = 0; n < rank_vec_len; n++)
        {
            fprintf(fh, "%d ", (displs[count_data_number])->displs[n]);
        }
        fprintf(fh, "\n");
    }
    DEBUG_LOGGER_NOARGS("Displacements saved\n");
    fprintf(fh, "END DATA\n");
    return 0;
}
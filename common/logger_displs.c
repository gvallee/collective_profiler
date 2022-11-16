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

int *lookup_rank_displs(int data_size, displs_data_t **data, int rank)
{
    assert(data);
    DEBUG_LOGGER("Looking up displacements for rank %d (%d data elements to scan)\n", rank, data_size);
    int i, j;
    for (i = 0; i < data_size; i++)
    {
        assert(data[i]);
        DEBUG_LOGGER("Pattern %d has %d ranks associated to it\n", i, data[i]->num_ranks);
        for (j = 0; j < data[i]->num_ranks; j++)
        {
            assert(data[i]->ranks);
            DEBUG_LOGGER("Scan previous displacements for rank %d\n", data[i]->ranks[j]);
            if (rank == data[i]->ranks[j])
            {
                return data[i]->displs;
            }
        }
    }
    DEBUG_LOGGER("Could not find data for rank %d\n", rank);
    return NULL;
}

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
            logger->recvdispls_filename = logger->get_full_filename(RECV_CTX, "displs", logger->jobid, logger->rank);
            logger->recvdispls_fh = fopen(logger->recvdispls_filename, "w");
        }
        fh = logger->recvdispls_fh;
        break;

    case SEND_CTX:
        if (logger->senddispls_fh == NULL)
        {
            logger->senddispls_filename = logger->get_full_filename(SEND_CTX, "displs", logger->jobid, logger->rank);
            logger->senddispls_fh = fopen(logger->senddispls_filename, "w");
        }
        fh = logger->senddispls_fh;
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

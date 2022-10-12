/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <sys/stat.h>
#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <assert.h>
#include <sys/types.h>
#include <unistd.h>

#include <openssl/sha.h>

#include "buff_content.h"
#include "collective_profiler_config.h"
#include "common_utils.h"

buffcontent_logger_t *buffcontent_loggers_head = NULL;
buffcontent_logger_t *buffcontent_loggers_tail = NULL;
uint64_t buffcontent_id = 0;

static inline int _close_buffcontent_file(buffcontent_logger_t *logger)
{
    if (logger->ctxt[0].fd)
    {
        fclose(logger->ctxt[0].fd);
        logger->ctxt[0].fd = NULL;
    }

    if (logger->ctxt[0].filename)
    {
        free(logger->ctxt[0].filename);
        logger->ctxt[0].filename = NULL;
    }

    if (logger->ctxt[1].fd)
    {
        fclose(logger->ctxt[1].fd);
        logger->ctxt[1].fd = NULL;
    }

    if (logger->ctxt[1].filename)
    {
        free(logger->ctxt[1].filename);
        logger->ctxt[1].filename = NULL;
    }

    return 0;
}

int fini_buffcontent_logger(buffcontent_logger_t **logger)
{
    _close_buffcontent_file(*logger);

    if ((*logger)->collective_name)
    {
        free((*logger)->collective_name);
        (*logger)->collective_name = NULL;
    }

    if ((*logger)->ctxt[0].fd)
    {
        fclose((*logger)->ctxt[0].fd);
        (*logger)->ctxt[0].fd = NULL;
    }

    if ((*logger)->ctxt[0].filename)
    {
        free((*logger)->ctxt[0].filename);
        (*logger)->ctxt[0].filename = NULL;
    }

    if ((*logger)->ctxt[1].fd)
    {
        fclose((*logger)->ctxt[1].fd);
        (*logger)->ctxt[1].fd = NULL;
    }

    if ((*logger)->ctxt[1].filename)
    {
        free((*logger)->ctxt[1].filename);
        (*logger)->ctxt[1].filename = NULL;
    }

    *logger = NULL;
    return 0;
}

int release_buffcontent_loggers()
{
    buffcontent_logger_t *ptr = buffcontent_loggers_head;
    while (ptr)
    {
        buffcontent_logger_t *next = ptr->next;
        int rc = fini_buffcontent_logger(&ptr);
        if (rc)
        {
            fprintf(stderr, "release_buffcontent_loggers() failed: %d\n", rc);
            return rc;
        }
        assert(ptr == NULL);
        ptr = next;
    }
    buffcontent_loggers_head = NULL;
    buffcontent_loggers_tail = NULL;
    return 0;
}

int store_call_data(char *collective_name, int ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], MPI_Datatype dt)
{
    buffcontent_logger_t *buffcontent_logger = NULL;
    int rc = get_buffcontent_logger(collective_name,
                                    ctxt,
                                    "w",
                                    comm,
                                    world_rank,
                                    comm_rank,
                                    &buffcontent_logger);
    if (rc)
    {
        fprintf(stderr, "Impossible to get logger\n");
        return rc;
    }
    assert(buffcontent_logger);
    DT_CHECK(dt);
    int dtsize;
    PMPI_Type_size(dt, &dtsize);

    int i;
    int comm_size;
    PMPI_Comm_size(comm, &comm_size);
    fprintf(buffcontent_logger->ctxt[ctxt].fd, "Call %" PRIu64 "\n", n_call);
    for (i = 0; i < comm_size; i++)
    {
        if (counts[i] == 0)
        {
            continue;
        }

        size_t data_size = counts[i] * dtsize;
        void *ptr = (void *)((uintptr_t)buf + (uintptr_t)(displs[i] * dtsize));
        size_t j;
        unsigned char sha256_buff[32];
        SHA256(ptr, data_size, sha256_buff);
        for (j = 0; j < 32; j++)
        {
            fprintf(buffcontent_logger->ctxt[ctxt].fd, "%02x", sha256_buff[j]);
        }
        fprintf(buffcontent_logger->ctxt[ctxt].fd, "\n");
    }
    fprintf(buffcontent_logger->ctxt[ctxt].fd, "\n");
    // fflush(buffcontent_logger->ctxt[ctxt].fd);

    return 0;
}

int store_call_data_single_count(char *collective_name, int ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int count, MPI_Datatype dt)
{
    buffcontent_logger_t *buffcontent_logger = NULL;
    int rc = get_buffcontent_logger(collective_name,
                                    ctxt,
                                    "w",
                                    comm,
                                    world_rank,
                                    comm_rank,
                                    &buffcontent_logger);
    if (rc)
    {
        fprintf(stderr, "Impossible to get logger\n");
        return rc;
    }
    assert(buffcontent_logger);
    DT_CHECK(dt);
    int dtsize;
    PMPI_Type_size(dt, &dtsize);

    int i;
    int comm_size;
    PMPI_Comm_size(comm, &comm_size);
    fprintf(buffcontent_logger->ctxt[ctxt].fd, "Call %" PRIu64 "\n", n_call);
    if (count != 0)
    {
        size_t data_size = count * dtsize;
        size_t j;
        unsigned char sha256_buff[32];
        SHA256(buf, data_size, sha256_buff);
        for (j = 0; j < 32; j++)
        {
            fprintf(buffcontent_logger->ctxt[ctxt].fd, "%02x", sha256_buff[j]);
        }
        fprintf(buffcontent_logger->ctxt[ctxt].fd, "\n");
    }
    fprintf(buffcontent_logger->ctxt[ctxt].fd, "\n");
    return 0;
}

int read_and_compare_call_data(char *collective_name, int ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], MPI_Datatype dt, bool check)
{
    buffcontent_logger_t *buffcontent_logger = NULL;
    int rc = get_buffcontent_logger(collective_name,
                                    ctxt,
                                    "r",
                                    comm,
                                    world_rank,
                                    comm_rank,
                                    &buffcontent_logger);
    if (rc)
        return rc;
    DT_CHECK(dt);
    int dtsize;
    PMPI_Type_size(dt, &dtsize);

    int i;
    int comm_size;
    PMPI_Comm_size(comm, &comm_size);

    // Read header ("Call X\n")
    uint64_t num_call;
    fscanf(buffcontent_logger->ctxt[ctxt].fd, "Call %" PRIu64 "\n", &num_call);
    for (i = 0; i < comm_size; i++)
    {
        if (counts[i] == 0)
        {
            continue;
        }

        size_t data_size = counts[i] * dtsize;
        void *ptr = (void *)((uintptr_t)buf + (uintptr_t)(displs[i] * dtsize));
        char buff[255];
        fscanf(buffcontent_logger->ctxt[ctxt].fd, "%254s\n", buff);
        if (check)
        {
            unsigned char sha256_buff[32];
            SHA256(ptr, data_size, sha256_buff);
            char data[255];
            size_t j;
            for (j = 0; j < 32; j++)
            {
                // 3 because it adds EOC
                snprintf(&data[j * 2], 3, "%02x", sha256_buff[j]);
            }

            if (strcmp(data, buff) != 0)
            {
                fprintf(stderr, "Rank %d: Content differ for call %" PRIu64 " (%s vs. %s)\n", world_rank, n_call, data, buff);
                PMPI_Abort(comm, 1);
            }
        }
    }
    fscanf(buffcontent_logger->ctxt[ctxt].fd, "\n");
}

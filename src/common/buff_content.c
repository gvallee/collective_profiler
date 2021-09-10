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
#include "comm.h"
#include "format.h"

buffcontent_logger_t *buffcontent_loggers_head = NULL;
buffcontent_logger_t *buffcontent_loggers_tail = NULL;
uint64_t buffcontent_id = 0;

static inline int
_open_content_storage_file(char *collective_name, char **filename, FILE **file, uint64_t comm_id, int world_rank, char *ctxt, char *mode)
{
    char *_filename = NULL;
    int rc;
    if (ctxt == NULL)
    {
        if (getenv(OUTPUT_DIR_ENVVAR))
        {
            _asprintf(_filename, rc, "%s/%s_buffcontent_comm%" PRIu64 "_rank%d.txt", getenv(OUTPUT_DIR_ENVVAR), collective_name, comm_id, world_rank);
            assert(rc > 0);
        }
        else
        {
            _asprintf(_filename, rc, "%s_buffcontent_comm%" PRIu64 "_rank%d.txt", collective_name, comm_id, world_rank);
            assert(rc > 0);
        }
    }
    else
    {
        if (getenv(OUTPUT_DIR_ENVVAR))
        {
            _asprintf(_filename, rc, "%s/%s_buffcontent_comm%" PRIu64 "_rank%d_%s.txt", getenv(OUTPUT_DIR_ENVVAR), collective_name, comm_id, world_rank, ctxt);
            assert(rc > 0);
        }
        else
        {
            _asprintf(_filename, rc, "%s_buffcontent_comm%" PRIu64 "_rank%d_%s.txt", collective_name, comm_id, world_rank, ctxt);
            assert(rc > 0);
        }
    }

    FILE *f = fopen(_filename, mode);
    assert(f);

    *file = f;
    *filename = _filename;
    return 0;
}

static inline int
init_buffcontent_logger(char *collective_name, int world_rank, MPI_Comm comm, uint64_t comm_id, char *mode, char *ctxt, buffcontent_logger_t **buffcontent_logger)
{
    assert(collective_name);
    buffcontent_logger_t *new_logger = malloc(sizeof(buffcontent_logger_t));
    assert(new_logger);
    new_logger->collective_name = strdup(collective_name);
    new_logger->world_rank = world_rank;
    new_logger->comm_id = comm_id;
    new_logger->comm = comm;
    new_logger->fd = NULL;
    new_logger->filename = NULL;
    new_logger->prev = NULL;
    new_logger->next = NULL;

    int rc = _open_content_storage_file(new_logger->collective_name, &new_logger->filename, &new_logger->fd, comm_id, new_logger->world_rank, ctxt, mode);
    if (rc)
    {
        fprintf(stderr, "_open_content_storage_files() failed: %d\n", rc);
        return -1;
    }
    assert(new_logger->filename);
    assert(new_logger->fd);

    if (buffcontent_loggers_head == NULL)
    {
        buffcontent_loggers_head = new_logger;
        buffcontent_loggers_tail = new_logger;
        new_logger->id = 0;
    }
    else
    {
        buffcontent_loggers_tail->next = new_logger;
        new_logger->prev = buffcontent_loggers_tail;
        new_logger->id = buffcontent_loggers_tail->id + 1;
        buffcontent_loggers_tail = new_logger;
    }

    if (strcmp(mode, "w") == 0)
    {
        // Write the format version at the begining of the file
        FORMAT_VERSION_WRITE(new_logger->fd);
    }
    else
    {
        // Read the format version so we can continue to read the file once we return from the function
        int version;
        fscanf(new_logger->fd, "FORMAT_VERSION: %d\n\n", &version);
        if (version != FORMAT_VERSION)
        {
            fprintf(stderr, "incompatible version (%d vs. %d)\n", version, FORMAT_VERSION);
            return -1;
        }
    }
    *buffcontent_logger = new_logger;
    return 0;
}

static inline int _close_buffcontent_file(buffcontent_logger_t *logger)
{
    if (logger->fd)
    {
        fclose(logger->fd);
        logger->fd = NULL;
    }

    if (logger->filename)
    {
        free(logger->filename);
        logger->filename = NULL;
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

    if ((*logger)->fd)
    {
        fclose((*logger)->fd);
        (*logger)->fd = NULL;
    }

    if ((*logger)->filename)
    {
        free((*logger)->filename);
        (*logger)->filename = NULL;
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

int lookup_buffcontent_logger(char *collective_name, MPI_Comm comm, buffcontent_logger_t **logger)
{
    buffcontent_logger_t *ptr = buffcontent_loggers_head;
    while (ptr != NULL)
    {
        if (strcmp(ptr->collective_name, collective_name) == 0 && ptr->comm == comm)
        {

            *logger = ptr;
            return 0;
        }
        ptr = ptr->next;
    }

    *logger = NULL;
    return 0;
}

int store_call_data(char *collective_name, char *ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], MPI_Datatype dt)
{
    buffcontent_logger_t *buffcontent_logger = NULL;
    GET_BUFFCONTENT_LOGGER(collective_name, ctxt, "w", comm, world_rank, comm_rank, buffcontent_logger);
    DT_CHECK(dt);
    int dtsize;
    PMPI_Type_size(dt, &dtsize);

    int i;
    int comm_size;
    PMPI_Comm_size(comm, &comm_size);
    fprintf(buffcontent_logger->fd, "Call %" PRIu64 "\n", n_call);
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
            fprintf(buffcontent_logger->fd, "%02x", sha256_buff[j]);
        }
        fprintf(buffcontent_logger->fd, "\n");
    }
    fprintf(buffcontent_logger->fd, "\n");
    //fflush(buffcontent_logger->fd);

    return 0;
}

int read_and_compare_call_data(char *collective_name, char* ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], MPI_Datatype dt, bool check)
{
    buffcontent_logger_t *buffcontent_logger = NULL;
    GET_BUFFCONTENT_LOGGER(collective_name, ctxt, "r", comm, world_rank, comm_rank, buffcontent_logger);
    DT_CHECK(dt);
    int dtsize;
    PMPI_Type_size(dt, &dtsize);

    int i;
    int comm_size;
    PMPI_Comm_size(comm, &comm_size);

    // Read header ("Call X\n")
    uint64_t num_call;
    fscanf(buffcontent_logger->fd, "Call %" PRIu64 "\n", &num_call);
    for (i = 0; i < comm_size; i++)
    {
        if (counts[i] == 0)
        {
            continue;
        }

        size_t data_size = counts[i] * dtsize;
        void *ptr = (void *)((uintptr_t)buf + (uintptr_t)(displs[i] * dtsize));
        char buff[255];
        fscanf(buffcontent_logger->fd, "%254s\n", buff);
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
    fscanf(buffcontent_logger->fd, "\n");
}

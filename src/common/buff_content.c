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

#include "buff_content.h"
#include "collective_profiler_config.h"
#include "common_utils.h"
#include "comm.h"
#include "format.h"

buffcontent_logger_t *buffcontent_loggers_head = NULL;
buffcontent_logger_t *buffcontent_loggers_tail = NULL;
uint64_t buffcontent_id = 0;

static inline int _open_content_storage_file(char *collective_name, char **filename, FILE **file, uint64_t comm_id, int world_rank, char *mode)
{
    char *_filename = NULL;
    int rc;
    // filename schema: buffcontent_rank<WORLDRANK>.txt
    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        _asprintf(_filename, rc, "%s/%s_buffcontent_comm%"PRIu64"_rank%d.txt", getenv(OUTPUT_DIR_ENVVAR), collective_name, comm_id, world_rank);
        assert(rc > 0);
    }
    else
    {
        _asprintf(_filename, rc, "%s_buffcontent_comm%"PRIu64"_rank%d.txt", collective_name, comm_id, world_rank);
        assert(rc > 0);
    }

    FILE *f = fopen(_filename, mode);
    assert(f);

    *file = f;
    *filename = _filename;
    return 0;

exit_on_error:
    *file = NULL;
    *filename = NULL;
    return -1;
}

static inline int init_buffcontent_logger(char *collective_name, int world_rank, MPI_Comm comm, uint64_t comm_id, char *mode, buffcontent_logger_t **buffcontent_logger)
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

    int rc = _open_content_storage_file(new_logger->collective_name, &new_logger->filename, &new_logger->fd, comm_id, new_logger->world_rank, mode);
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

    if (strcmp(mode, "w") == 0) {
        // Write the format version at the begining of the file
        FORMAT_VERSION_WRITE(new_logger->fd);
    } else {
        // Read the format version so we can continue to read the file once we return from the function
        int version;
        fscanf(new_logger->fd, "FORMAT_VERSION: %d\n\n", &version);
        if (version != FORMAT_VERSION) {
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
    int rc;
    buffcontent_logger_t *ptr = buffcontent_loggers_head;
    while (ptr)
    {
        buffcontent_logger_t *next = ptr->next;
        rc = fini_buffcontent_logger(&ptr);
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
    int i;
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

int store_call_data(char *collective_name, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], int dtsize)
{
    uint32_t comm_id;
    int rc;
    buffcontent_logger_t *buffcontent_logger = NULL;

    rc = lookup_comm(comm, &comm_id);
    if (rc)
    {
        rc = add_comm(comm, world_rank, comm_rank, &comm_id);
        if (rc)
        {
            fprintf(stderr, "add_comm() failed: %d\n", rc);
            return 1;
        }
    }

    rc = lookup_buffcontent_logger(collective_name, comm, &buffcontent_logger);
    if (rc)
    {
        fprintf(stderr, "lookup_buffcontent_logger() failed: %d\n", rc);
        return 1;
    }

    if (buffcontent_logger == NULL)
    {
        rc = init_buffcontent_logger(collective_name, world_rank, comm, comm_id, "w", &buffcontent_logger);
        if (rc)
        {
            fprintf(stderr, "init_buffcontent_logger() failed: %d\n", rc);
            return 1;
        }
    }
    assert(buffcontent_logger);
    assert(buffcontent_logger->fd);

    int i;
    int comm_size;
    MPI_Comm_size(comm, &comm_size);
    fprintf(buffcontent_logger->fd, "Call %"PRIu64"\n", n_call);
    for (i = 0; i < comm_size; i++) 
    {
        void *ptr = (void*)(buf + (uintptr_t)displs[i]);
        size_t data_size = counts[i] * dtsize;
        size_t j;
        uint8_t *b = (uint8_t*)ptr;

        if (counts[i] == 0) {
            continue;
        }

        for (j = 0; j < data_size; j++) {
            fprintf(buffcontent_logger->fd, "%02x", b[j]);
        }     
        fprintf(buffcontent_logger->fd, "\n");
    }
    fprintf(buffcontent_logger->fd, "\n");

    return 0;
}

int read_and_compare_call_data(char *collective_name, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], int dtsize)
{
        uint32_t comm_id;
    int rc;
    buffcontent_logger_t *buffcontent_logger = NULL;

    rc = lookup_comm(comm, &comm_id);
    if (rc)
    {
        rc = add_comm(comm, world_rank, comm_rank, &comm_id);
        if (rc)
        {
            fprintf(stderr, "add_comm() failed: %d\n", rc);
            return 1;
        }
    }

    rc = lookup_buffcontent_logger(collective_name, comm, &buffcontent_logger);
    if (rc)
    {
        fprintf(stderr, "lookup_buffcontent_logger() failed: %d\n", rc);
        return 1;
    }

    if (buffcontent_logger == NULL)
    {
        rc = init_buffcontent_logger(collective_name, world_rank, comm, comm_id, "r", &buffcontent_logger);
        if (rc)
        {
            fprintf(stderr, "init_buffcontent_logger() failed: %d\n", rc);
            return 1;
        }
    }
    assert(buffcontent_logger);
    assert(buffcontent_logger->fd);

    int i;
    int comm_size;
    MPI_Comm_size(comm, &comm_size);
    char buff[255];
    // Read header ("Call X\n")
    uint64_t num_call;
    fscanf(buffcontent_logger->fd, "Call %"PRIu64"\n", &num_call);
    for (i = 0; i < comm_size; i++) 
    {
        void *ptr = (void*)(buf + (uintptr_t)displs[i]);
        size_t data_size = counts[i] * dtsize;
        size_t j;

        if (counts[i] == 0) {
            continue;
        }

        fscanf(buffcontent_logger->fd, "%s\n", buff);
        uint8_t *b = (uint8_t*)ptr;
        char data[255];
        int idx = 0;
        for (j = 0; j < data_size; j++) {
            // 3 because it adds EOC
            snprintf(&data[idx], 3, "%02x", b[j]);
            idx += 2;
        }
        
        if (strcmp(data, buff) != 0) {
                fprintf(stderr, "Rank %d: Content differ (%s vs. %s)\n", world_rank, data, buff);
                MPI_Abort(comm, 1);
        }     
    }
    fscanf(buffcontent_logger->fd, "%s\n", buff);
}

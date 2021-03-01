/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>

#include "location.h"
#include "comm.h"
#include "collective_profiler_config.h"
#include "common_utils.h"
#include "format.h"

location_logger_t *location_loggers_head = NULL;
location_logger_t *location_loggers_tail = NULL;

static inline int _open_location_file(location_logger_t *logger)
{
    int rc;
    assert(logger);
    assert(logger->collective_name);
    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        _asprintf(logger->filename, rc, "%s/%s_locations_comm%" PRIu64 "_rank%d.md", getenv(OUTPUT_DIR_ENVVAR), logger->collective_name, logger->commid, logger->world_rank);
    }
    else
    {
        _asprintf(logger->filename, rc, "%s_locations_comm%" PRIu64 "_rank%d.md", logger->collective_name, logger->commid, logger->world_rank);
    }
    assert(rc > 0);

    logger->fd = fopen(logger->filename, "w");
    assert(logger->fd);
    return 0;
}

int init_location_logger(char *collective_name, int world_rank, uint64_t comm_id, size_t comm_size, char *hostnames, int *pids, int *world_comm_ranks, uint64_t callID, location_logger_t **logger)
{
    int rc;

    location_logger_t *new_logger = malloc(sizeof(location_logger_t));
    assert(new_logger);
    new_logger->world_rank = world_rank;
    new_logger->commid = comm_id;
    new_logger->comm_size = comm_size;
    new_logger->world_comm_ranks = world_comm_ranks;
    new_logger->collective_name = strdup(collective_name);
    new_logger->pids = pids;
    new_logger->calls_max = 2;
    new_logger->calls = malloc(new_logger->calls_max * sizeof(uint64_t));
    assert(new_logger->calls);
    new_logger->calls_count = 1;
    new_logger->calls[0] = callID;
    new_logger->locations = hostnames;
    new_logger->fd = NULL;
    new_logger->filename = NULL;
    new_logger->next = NULL;
    new_logger->prev = NULL;

    rc = _open_location_file(new_logger);
    if (rc)
    {
        fprintf(stderr, "_open_location_file() failed: %d\n", rc);
        return -1;
    }
    assert(new_logger->fd);

    if (location_loggers_head == NULL)
    {
        location_loggers_head = new_logger;
        location_loggers_tail = new_logger;
    }
    else
    {
        location_loggers_tail->next = new_logger;
        new_logger->prev = location_loggers_tail;
        location_loggers_tail = new_logger;
    }

    // Write the format version at the begining of the file
    FORMAT_VERSION_WRITE(new_logger->fd);
    *logger = new_logger;

    return 0;
}

int lookup_location_logger(uint64_t commid, location_logger_t **logger)
{
    location_logger_t *ptr = location_loggers_head;
    while (ptr)
    {
        if (ptr->commid == commid)
        {
            *logger = ptr;
            return 0;
        }
        ptr = ptr->next;
    }

    // We could find data logger
    *logger = NULL;
    return 0;
}

static inline int _close_location_file(location_logger_t *logger)
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

static inline int _write_location_to_file(location_logger_t *logger)
{
    assert(logger);
    assert(logger->fd);

    fprintf(logger->fd, "Communicator ID: %"PRIu64"\n", logger->commid);
    char *strCalls = compress_uint64_array(logger->calls, logger->calls_count, 1);
    assert(strCalls);
    fprintf(logger->fd, "Calls: %s\n", strCalls);
    char *strRanks = compress_int_array(logger->world_comm_ranks, logger->comm_size, 1);
    assert(strRanks);
    fprintf(logger->fd, "COMM_WORLD ranks: %s\n", strRanks);
    char *strPIDs = compress_int_array(logger->pids, logger->comm_size, 1);
    assert(strPIDs);
    fprintf(logger->fd, "PIDs: %s\n", strPIDs);
    fprintf(logger->fd, "Hostnames:\n");
    int i;
    for(i = 0; i < logger->comm_size; i++)
    {
        fprintf(logger->fd, "\tRank %d: %s\n", i, &(logger->locations[i * 256]));
    }
    free(strCalls);
    free(strPIDs);
    free(strRanks);
    return 0;
}

int fini_location_tracking(location_logger_t **logger)
{
    _write_location_to_file(*logger);
    _close_location_file(*logger);

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

    if ((*logger)->pids)
    {
        free((*logger)->pids);
        (*logger)->pids = NULL;
    }

    if ((*logger)->calls)
    {
        free((*logger)->calls);
        (*logger)->calls = NULL;
    }

    assert((*logger)->collective_name);
    free((*logger)->collective_name);
    free(*logger);
    *logger = NULL;

    return 0;
}

int release_location_loggers()
{
    
    while (location_loggers_head)
    {
        location_logger_t *ptr = location_loggers_head->next;
        int rc = fini_location_tracking(&location_loggers_head);
        if (rc)
        {
            fprintf(stderr, "fini_location_tracking() failed: %d\n", rc);
            // Do not return, let's the library try to clean up as much as possible
        }
        location_loggers_head = ptr;
    }
    return 0;
}

int commit_rank_locations(char *collective_name, MPI_Comm comm, int comm_size, int world_rank, int comm_rank, int *pids, int *world_comm_ranks, char *hostnames, uint64_t n_call)
{
    int i;
    int rc;
    location_logger_t *logger;

    uint32_t comm_id;
    rc = lookup_comm(comm, &comm_id);
    if (rc)
    {
        // We save the communicator
        rc = add_comm(comm, world_rank, comm_rank, &comm_id);
        if (rc)
        {
            fprintf(stderr, "unabel to add communicator\n");
            return rc;
        }
    }

    // Do we already have that communicator's data
    rc = lookup_location_logger(comm_id, &logger);
    if (rc)
    {
        fprintf(stderr, "lookup_location_logger() failed: %d\n", rc);
        return rc;
    }

    if (logger == NULL)
    {
        // We have no data about the communicator
        // We check first if the communicator is already known

        rc = init_location_logger(collective_name, world_rank, comm_id, comm_size, hostnames, pids, world_comm_ranks, n_call, &logger);
        if (rc)
        {
            fprintf(stderr, "init_location_logger(): %d\n", rc);
            return rc;
        }
    }
    else
    {
        // Simply add the call to the list of the logger's calls
        if (logger->calls_count == logger->calls_max)
        {
            logger->calls_max *= 2;
            logger->calls = realloc(logger->calls, logger->calls_max * sizeof(uint64_t));
            assert(logger->calls);
        }
        logger->calls[logger->calls_count] = n_call;
        logger->calls_count++;
    }

    return 0;
}

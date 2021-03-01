/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include "timings.h"
#include "comm.h"
#include "collective_profiler_config.h"
#include "common_utils.h"
#include "format.h"

comm_timing_logger_t *timing_loggers_head = NULL;
comm_timing_logger_t *timing_loggers_tail = NULL;

int init_time_tracking(MPI_Comm comm, char *collective_name, int world_rank, int comm_rank, int jobid, comm_timing_logger_t **logger)
{
    int rc;

    uint32_t comm_id;
    GET_COMM_LOGGER(comm, world_rank, comm_rank, comm_id);

    comm_timing_logger_t *new_logger = malloc(sizeof(comm_timing_logger_t));
    assert(new_logger);
    new_logger->filename = NULL;
    new_logger->next = NULL;
    new_logger->prev = NULL;
    new_logger->comm_id = comm_id;

#if ENABLE_EXEC_TIMING
    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        _asprintf(new_logger->filename, rc, "%s/%s_execution_times.rank%d_comm%" PRIu32 "_job%d.md", getenv(OUTPUT_DIR_ENVVAR), collective_name, world_rank, comm_id, jobid);
    }
    else
    {
        _asprintf(new_logger->filename, rc, "%s_execution_times.rank%d_comm%" PRIu32 "_job%d.md", collective_name, world_rank, comm_id, jobid);
    }
#endif // ENABLE_EXEC_TIMING

#if ENABLE_LATE_ARRIVAL_TIMING
    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        _asprintf(new_logger->filename, rc, "%s/%s_late_arrival_times.rank%d_comm%" PRIu32 "_job%d.md", getenv(OUTPUT_DIR_ENVVAR), collective_name, world_rank, comm_id, jobid);
    }
    else
    {
        _asprintf(new_logger->filename, rc, "%s_late_arrival_times.rank%d_comm%" PRIu32 "_job%d.md", collective_name, world_rank, comm_id, jobid);
    }
#endif // ENABLE_LATE_ARRIVAL_TIMING
    assert(rc > 0);
    assert(new_logger->filename);

    if (timing_loggers_head == NULL)
    {
        timing_loggers_head = new_logger;
        timing_loggers_tail = new_logger;
    }
    else
    {
        timing_loggers_tail->next = new_logger;
        new_logger->prev = timing_loggers_tail;
        timing_loggers_tail = new_logger;
    }

    new_logger->fd = fopen(new_logger->filename, "w");
    assert(new_logger->fd);
    // Write the format version at the begining of the file
    FORMAT_VERSION_WRITE(new_logger->fd);
    fclose(new_logger->fd);
    new_logger->fd = NULL;

    *logger = new_logger;

    return 0;
}

int lookup_timing_logger(MPI_Comm comm, comm_timing_logger_t **logger)
{
    comm_timing_logger_t *ptr = timing_loggers_head;
    uint32_t comm_id;

    int rc = lookup_comm(comm, &comm_id);
    if (rc)
    {
        // We try to use a logger for a communicator that we know nothing about
        *logger = NULL;
        return 1;
    }

    while (ptr != NULL)
    {
        if (ptr->comm_id == comm_id)
        {
            *logger = ptr;
            return 0;
        }
        ptr = ptr->next;
    }

    // We could find data about the communicator but no associated logger
    *logger = NULL;
    return 0;
}

int fini_time_tracking(comm_timing_logger_t **logger)
{
    if ((*logger)->fd)
    {
        fclose((*logger)->fd);
        (*logger)->fd = NULL;
    }
    free((*logger)->filename);
    free((*logger));
    *logger = NULL;

    return 0;
}

int release_time_loggers()
{
    while (timing_loggers_head)
    {
        comm_timing_logger_t *ptr = timing_loggers_head->next;
        fini_time_tracking(&timing_loggers_head);
        timing_loggers_head = ptr;
        if (ptr != NULL)
            ptr->prev = NULL;
    }
    return 0;
}

int commit_timings(MPI_Comm comm, char *collective_name, int world_rank, int comm_rank, int jobid, double *times, int comm_size, uint64_t n_call)
{
    assert(times);
    comm_timing_logger_t *logger;
    int rc = lookup_timing_logger(comm, &logger);
    if (rc || logger == NULL)
    {
        // We check first if the communicator is already known
        uint32_t comm_id;
        rc = lookup_comm(comm, &comm_id);
        if (rc)
        {
            rc = add_comm(comm, world_rank, comm_rank, &comm_id);
            if (rc)
            {
                fprintf(stderr, "unable to add communicator\n");
                return rc;
            }
        }

        // Now we know the communicator, create a logger for it
        rc = init_time_tracking(comm, collective_name, world_rank, comm_rank, jobid, &logger);
        if (rc || logger == NULL)
        {
            fprintf(stderr, "unable to initialize time tracking (rc: %d)\n", rc);
            return 1;
        }
    }
    assert(logger);

    if (logger->fd == NULL)
    {
        assert(logger->filename);
        logger->fd = fopen(logger->filename, "a");
    }
    assert(logger->fd);

    // We know from here we have a correct logger
    int i;
    fprintf(logger->fd, "# Call %" PRIu64 "\n", n_call);
    for (i = 0; i < comm_size; i++)
    {
        fprintf(logger->fd, "%f\n", times[i]);
    }
    fprintf(logger->fd, "\n");
    // We experienced some unexpected IO problems when we do not close the file
    // after the each alltoallv operation.
    fclose(logger->fd);
    logger->fd = NULL;
    return 0;
}

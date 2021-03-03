/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <assert.h>
#include <stdio.h>
#include <ctype.h>
#include <string.h>

#include "comm.h"
#include "common_utils.h"
#include "collective_profiler_config.h"
#include "format.h"

comm_data_t *comm_data_head = NULL;
comm_data_t *comm_data_tail = NULL;
uint32_t next_id = 0;

int lookup_comm(MPI_Comm comm, uint32_t *id)
{
    comm_data_t *data = comm_data_head;
    while (data != NULL)
    {
        if (data->comm == comm)
        {
            *id = data->id;
            return 0;
        }
        data = data->next;
    }
    return 1;
}

int add_comm(MPI_Comm comm, int world_rank, int comm_rank, uint32_t *id)
{
    if (comm_data_head == NULL)
    {
        comm_data_head = malloc(sizeof(comm_data_t));
        assert(comm_data_head);
        comm_data_head->id = next_id;
        comm_data_head->next = NULL;
        comm_data_head->comm = comm;
        comm_data_head->world_rank = world_rank;
        comm_data_head->comm_rank = comm_rank;
        comm_data_tail = comm_data_head;
    }
    else
    {
        comm_data_t *new_data = malloc(sizeof(comm_data_t));
        assert(new_data);
        new_data->id = next_id;
        new_data->next = NULL;
        new_data->comm = comm;
        new_data->world_rank = world_rank;
        new_data->comm_rank = comm_rank;
        comm_data_tail->next = new_data;
        comm_data_tail = new_data;
    }
    *id = next_id;
    next_id++;
    return 0;
}

int save_logger_data(comm_data_t *comm, FILE *fd)
{
    if (fd == NULL)
    {
        fprintf(stderr, "file descriptor to save communicators' data is undefined\n");
        return 1;
    }

    if (comm->comm_rank == 0)
    {
        fprintf(fd, "ID: %" PRIu32 "; world rank: %d\n", comm->id, comm->world_rank);
    }
    return 0;
}

int release_comm_data(char *collective_name, int lead_rank)
{
    int rc;
    char *filename = NULL;
    char *name = NULL;
    FILE *fd = NULL;

    while (comm_data_head != NULL)
    {
        comm_data_t *ptr = comm_data_head->next;
        if (comm_data_head->comm_rank == 0)
        {
            if (fd == NULL)
            {

                // Convert the collective name to a all-lower case string for consistency
                name = strdup(collective_name);
                name[0] = tolower(name[0]);

                //int lead_rank = comm_data_head->lead_rank;
                if (getenv(OUTPUT_DIR_ENVVAR))
                {
                    _asprintf(filename, rc, "%s/%s_comm_data_rank%d.md", getenv(OUTPUT_DIR_ENVVAR), name, lead_rank);
                    assert(rc > 0);
                }
                else
                {
                    _asprintf(filename, rc, "%s_comm_data_rank%d.md", name, lead_rank);
                    assert(rc > 0);
                }

                assert(filename);
                fd = fopen(filename, "w");
                assert(fd);
                FORMAT_VERSION_WRITE(fd);
            }
            assert(lead_rank == comm_data_head->world_rank);
            rc = save_logger_data(comm_data_head, fd);
            if (rc)
            {
                fclose(fd);
                free(filename);
                filename = NULL;
                return rc;
            }
        }
        free(comm_data_head);
        comm_data_head = ptr;
    }

    if (fd)
    {
        fclose(fd);
        fd = NULL;
        free(filename);
        filename = NULL;
        free(name);
        name = NULL;
    }
    return 0;
}
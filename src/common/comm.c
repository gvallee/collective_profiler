/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include "comm.h"
#include <stdlib.h>
#include <assert.h>

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

int add_comm(MPI_Comm comm, uint32_t *id)
{
    if (comm_data_head == NULL)
    {
        comm_data_head = malloc(sizeof(comm_data_t));
        assert(comm_data_head);
        comm_data_head->id = next_id;
        comm_data_head->next = NULL;
        comm_data_head->comm = comm;
        comm_data_tail = comm_data_head;
    }
    else
    {
        comm_data_t *new_data = malloc(sizeof(comm_data_t));
        assert(new_data);
        new_data->id = next_id;
        new_data->next = NULL;
        new_data->comm = comm;
        comm_data_tail->next = new_data;
        comm_data_tail = new_data;
    }
    *id = next_id;
    next_id++;
    return 0;
}

int release_comm_data()
{
    while (comm_data_head != NULL)
    {
        comm_data_t *ptr = comm_data_head->next;
        free(comm_data_head);
        comm_data_head = ptr;
    }
    return 0;
}
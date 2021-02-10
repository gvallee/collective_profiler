/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef COLLECTIVE_PROFILER_COMM_H
#define COLLECTIVE_PROFILER_COMM_H

#include <inttypes.h>
#include "mpi.h"

typedef struct comm_data
{
    uint32_t id;
    MPI_Comm comm;
    struct comm_data *next;

} comm_data_t;

int lookup_comm(MPI_Comm comm, uint32_t *id);
int add_comm(MPI_Comm comm, uint32_t *id);
int release_comm_data();

#endif // COLLECTIVE_PROFILER_COMM_H
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

#define GET_COMM_LOGGER(_comm_id)                                                  \
    do                                                                             \
    {                                                                              \
        int i;                                                                     \
        int rc;                                                                    \
        rc = lookup_comm(comm, &_comm_id);                                         \
        if (rc)                                                                    \
        {                                                                          \
            rc = add_comm(comm, &_comm_id);                                        \
            if (rc)                                                                \
            {                                                                      \
                fprintf(stderr, "unable to add communictor to tracking system\n"); \
                return 1;                                                          \
            }                                                                      \
        }                                                                          \
    } while (0)

#endif // COLLECTIVE_PROFILER_COMM_H
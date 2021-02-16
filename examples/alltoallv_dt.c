/*************************************************************************
 * Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <stdio.h>
#include <assert.h>
#include "mpi.h"

#define MPICHECK(c)                                  \
    do                                               \
    {                                                \
        if (c != MPI_SUCCESS)                        \
        {                                            \
            fprintf(stderr, "MPI command failed\n"); \
            return 1;                                \
        }                                            \
    } while (0);

int main(int argc, char **argv)
{
    int i;
    int world_size;
    int world_rank;
    int *send_buffer_int;
    int *recv_buffer_int;
    double *send_buffer_double;
    double *recv_buffer_double;
    int *send_count;
    int *recv_count;
    int *recv_displ;
    int *send_displ;

    MPICHECK(MPI_Init(&argc, &argv));
    MPICHECK(MPI_Comm_size(MPI_COMM_WORLD, &world_size));
    MPICHECK(MPI_Comm_rank(MPI_COMM_WORLD, &world_rank));

    send_buffer_int = (int *)calloc(world_size * world_size, sizeof(int));
    assert(send_buffer_int);
    recv_buffer_int = (int *)calloc(world_size * world_size, sizeof(int));
    assert(recv_buffer_int);
    send_buffer_double = (double *)calloc(world_size * world_size, sizeof(double));
    assert(send_buffer_double);
    recv_buffer_double = (double *)calloc(world_size * world_size, sizeof(double));
    assert(recv_buffer_double);
    send_count = calloc(world_size, sizeof(int));
    assert(send_count);
    recv_count = calloc(world_size, sizeof(int));
    assert(recv_count);
    send_displ = calloc(world_size, sizeof(int));
    assert(send_displ);
    recv_displ = calloc(world_size, sizeof(int));
    assert(recv_displ);

    for (i = 0; i < world_size * world_size; i++)
    {
        send_buffer_int[i] = i + 10 * world_rank;
    }

    for (i = 0; i < world_size * world_size; i++)
    {
        send_buffer_double[i] = i + 10 * world_rank;
    }

    for (i = 0; i < world_size; i++)
    {
        send_count[i] = 1;
        recv_count[i] = 1;
        recv_displ[i] = 0;
        send_displ[i] = 0;
    }

    if (world_rank == 0)
    {
        int s;
        MPI_Type_size(MPI_INT, &s);
        fprintf(stdout, "Size of MPI_INT: %d\n", s);
        MPI_Type_size(MPI_DOUBLE, &s);
        fprintf(stdout, "Size of MPI_DOUBLE: %d\n", s);
    }

    MPICHECK(MPI_Alltoallv(send_buffer_int, send_count, send_displ, MPI_INT,
                           recv_buffer_int, recv_count, recv_displ, MPI_INT,
                           MPI_COMM_WORLD));

    MPICHECK(MPI_Alltoallv(send_buffer_double, send_count, send_displ, MPI_DOUBLE,
                           recv_buffer_double, recv_count, recv_displ, MPI_DOUBLE,
                           MPI_COMM_WORLD));

    free(send_buffer_int);
    free(recv_buffer_int);
    free(send_buffer_double);
    free(recv_buffer_double);
    free(send_count);
    free(recv_count);
    free(send_displ);
    free(recv_displ);
    MPI_Finalize();
    return EXIT_SUCCESS;

exit_on_failure:
    MPI_Finalize();
    return EXIT_FAILURE;
}

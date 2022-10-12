/*************************************************************************
 * Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <stdio.h>
#include <mpi.h>

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
    int send_buffer;
    int send_count = 1;
    int *recv_buffer;
    int *recv_count;
    int *recv_displ;

    MPICHECK(MPI_Init(&argc, &argv));
    MPICHECK(MPI_Comm_size(MPI_COMM_WORLD, &world_size));
    MPICHECK(MPI_Comm_rank(MPI_COMM_WORLD, &world_rank));
    send_buffer = world_rank;

    recv_buffer = (int *)calloc(world_size, sizeof(int));
    recv_count = calloc(world_size, sizeof(int));
    recv_displ = calloc(world_size, sizeof(int));
    if (!recv_buffer || !recv_count || !recv_displ)
    {
        fprintf(stderr, "Out of resources\n");
        goto exit_on_failure;
    }

    for (i = 0; i < world_size; i++)
    {
        recv_count[i] = 1;
        recv_displ[i] = i;
    }

    MPICHECK(MPI_Allgatherv(&send_buffer, send_count, MPI_INT,
                            recv_buffer, recv_count, recv_displ, MPI_INT,
                            MPI_COMM_WORLD));

    free(recv_buffer);
    free(recv_count);
    free(recv_displ);
    MPI_Finalize();
    return EXIT_SUCCESS;

exit_on_failure:
    MPI_Finalize();
    return EXIT_FAILURE;
}

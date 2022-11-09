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

static int run_allgatherv(int num_elts, int world_size)
{
    size_t idx = 0;
    int i, j;
    int *send_buffer;
    int send_count = num_elts;
    int *recv_buffer;
    int *recv_counts;
    int *recv_displs;

    send_buffer = (int *)calloc(num_elts, sizeof(int));
    recv_buffer = (int *)calloc(num_elts * world_size, sizeof(int));
    recv_counts = (int *)calloc(world_size, sizeof(int));
    recv_displs = (int *)calloc(world_size, sizeof(int));
    if (!send_buffer || !recv_buffer || !recv_counts || !recv_displs)
    {
        fprintf(stderr, "Out of resources\n");
        goto exit_on_failure;
    }

    for (i = 0; i < num_elts; i++)
        send_buffer[i] = i;

    for (i = 0; i < world_size; i++)
    {
        recv_counts[i] = num_elts;
        recv_displs[i] = i + num_elts;
    }

    MPICHECK(MPI_Allgatherv(send_buffer, send_count, MPI_INT,
                            recv_buffer, recv_counts, recv_displs, MPI_INT,
                            MPI_COMM_WORLD));
    free(send_buffer);
    free(recv_buffer);
    free(recv_counts);
    free(recv_displs);
    return 0;

exit_on_failure:
    free(send_buffer);
    free(recv_buffer);
    free(recv_counts);
    free(recv_displs);
    return -1;
}

int main(int argc, char **argv)
{
    int rc, num_elts;
    int world_size;
    int world_rank;

    MPICHECK(MPI_Init(&argc, &argv));
    MPICHECK(MPI_Comm_size(MPI_COMM_WORLD, &world_size));
    MPICHECK(MPI_Comm_rank(MPI_COMM_WORLD, &world_rank));

    for (num_elts = 1; num_elts < 3; num_elts++)
    {
        rc = run_allgatherv(num_elts, world_size);
        if (rc)
            goto exit_on_failure;
    }

    MPI_Finalize();
    return EXIT_SUCCESS;

exit_on_failure:
    MPI_Finalize();
    return EXIT_FAILURE;
}

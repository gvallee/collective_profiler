/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include "mpi.h"
#include "collective_profiler_example_utils.h"

int alltoallv1(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv2(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv3(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv4(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv5(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv6(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv7(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv8(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv9(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int alltoallv10(int *send_buffer, int *send_count, int *send_displ, int *recv_buffer, int *recv_count, int *recv_displ)
{
    return MPI_Alltoallv(send_buffer, send_count, send_displ, MPI_INT,
                         recv_buffer, recv_count, recv_displ, MPI_INT,
                         MPI_COMM_WORLD);
}

int main(int argc, char **argv)
{
    int i;
    int world_size;
    int world_rank;
    int *send_buffer;
    int *recv_buffer;
    int *send_count;
    int *recv_count;
    int *recv_displ;
    int *send_displ;
    uint64_t iter = 10000;

    MPICHECK(MPI_Init(&argc, &argv));
    MPICHECK(MPI_Comm_size(MPI_COMM_WORLD, &world_size));
    MPICHECK(MPI_Comm_rank(MPI_COMM_WORLD, &world_rank));

    send_buffer = (int *)calloc(world_size * world_size, sizeof(int));
    recv_buffer = (int *)calloc(world_size * world_size, sizeof(int));
    send_count = calloc(world_size, sizeof(int));
    recv_count = calloc(world_size, sizeof(int));
    send_displ = calloc(world_size, sizeof(int));
    recv_displ = calloc(world_size, sizeof(int));
    if (!send_buffer || !recv_buffer || !send_count || !recv_count || !send_displ || !recv_displ)
    {
        fprintf(stderr, "Out of resources\n");
        goto exit_on_failure;
    }

    for (i = 0; i < world_size * world_size; i++)
    {
        send_buffer[i] = i + 10 * world_rank;
    }

    for (i = 0; i < world_size; i++)
    {
        send_count[i] = i;
        recv_count[i] = world_rank;
        recv_displ[i] = i * world_rank;
        send_displ[i] = (i * (i + 1) / 2);
    }

    int n;
    for (n = 0; n < iter; n++)
    {
        MPICHECK(alltoallv1(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv2(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv3(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv4(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv5(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv6(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv7(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv8(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv9(send_buffer, send_count, send_displ,
                            recv_buffer, recv_count, recv_displ));
        MPICHECK(alltoallv10(send_buffer, send_count, send_displ,
                             recv_buffer, recv_count, recv_displ));
    }

    free(send_buffer);
    free(recv_buffer);
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

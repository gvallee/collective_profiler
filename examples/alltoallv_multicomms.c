/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
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


typedef struct alltoallv_info
{
    int *send_buffer;
    int *recv_buffer;
    int *send_counts;
    int *recv_counts;
    int *send_displs;
    int *recv_displs;
    MPI_Comm comm;
} alltoallv_info_t;

static int do_alltoallv(alltoallv_info_t *info)
{
    return MPI_Alltoallv(info->send_buffer, info->send_counts, info->send_displs, MPI_INT,
                         info->recv_buffer, info->recv_counts, info->recv_displs, MPI_INT,
                         info->comm);
}

static alltoallv_info_t *setup_alltoallv(MPI_Comm comm)
{
    int comm_size;
    int comm_rank;

    MPI_Comm_rank(comm, &comm_rank);
    MPI_Comm_size(comm, &comm_size);

    alltoallv_info_t *info = (alltoallv_info_t *)malloc(sizeof(alltoallv_info_t));
    assert(info);
    info->comm = comm;
    info->send_buffer = (int *)calloc(comm_size * comm_size, sizeof(int));
    assert(info->send_buffer);
    info->recv_buffer = (int *)calloc(comm_size * comm_size, sizeof(int));
    assert(info->recv_buffer);
    info->send_counts = (int *)calloc(comm_size, sizeof(int));
    assert(info->send_counts);
    info->recv_counts = (int *)calloc(comm_size, sizeof(int));
    assert(info->recv_counts);
    info->send_displs = calloc(comm_size, sizeof(int));
    assert(info->send_displs);
    info->recv_displs = calloc(comm_size, sizeof(int));
    assert(info->recv_displs);

    int i;
    for (i = 0; i < comm_size * comm_size; i++)
    {
        info->send_buffer[i] = i + 10 * comm_rank;
    }

    for (i = 0; i < comm_size; i++)
    {
        info->send_counts[i] = i;
        info->recv_counts[i] = comm_rank;
        info->recv_displs[i] = i * comm_rank;
        info->send_displs[i] = (i * (i + 1) / 2);
    }

    return info;
}

static int finalize_alltoallv(alltoallv_info_t **info)
{
    free((*info)->send_buffer);
    free((*info)->recv_buffer);
    free((*info)->send_counts);
    free((*info)->recv_counts);
    free((*info)->send_displs);
    free((*info)->recv_displs);
    free(*info);
    *info = NULL;

    return 0;
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

    MPICHECK(MPI_Init(&argc, &argv));
    MPICHECK(MPI_Comm_size(MPI_COMM_WORLD, &world_size));
    MPICHECK(MPI_Comm_rank(MPI_COMM_WORLD, &world_rank));

    if (world_size != 4)
    {
        fprintf(stderr, "This test is designed to run with 4 ranks\n");
        return EXIT_FAILURE;
    }

    // We create 2 subcommunicators
    int color = world_rank / 2;
    MPI_Comm sub_comm;
    MPICHECK(MPI_Comm_split(MPI_COMM_WORLD, color, world_rank, &sub_comm));

    alltoallv_info_t *world_alltoallv = setup_alltoallv(MPI_COMM_WORLD);
    alltoallv_info_t *subcomm_alltoallv = setup_alltoallv(sub_comm);

    MPICHECK(do_alltoallv(subcomm_alltoallv));
    MPICHECK(do_alltoallv(world_alltoallv));
    MPICHECK(do_alltoallv(subcomm_alltoallv));

    finalize_alltoallv(&subcomm_alltoallv);
    finalize_alltoallv(&world_alltoallv);

    MPI_Finalize();
    return EXIT_SUCCESS;

exit_on_failure:
    MPI_Finalize();
    return EXIT_FAILURE;
}

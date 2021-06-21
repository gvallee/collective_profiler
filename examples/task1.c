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

static void print_buffer_int(void *buf, int len, char *msg, int rank) {
    int tmp, *v;
    printf("**<%d> %s (#%d): ", rank, msg, len);
    for (tmp = 0; tmp < len; tmp++) {
        v = buf;
        printf("[%d]", v[tmp]);
    }
    printf("\n");
    free(msg);
}

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

static alltoallv_info_t *setup_alltoallv_unbalanced(MPI_Comm comm)
{
    int comm_size;
    printf("ucomm_size %d ",comm_size);
    int comm_rank;

    MPI_Comm_rank(comm, &comm_rank);
    MPI_Comm_size(comm, &comm_size);

    if (comm_rank < 40) {
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
        for (i = 0; i < comm_size * comm_size; i++) {
            info->send_buffer[i] = i + 10 * comm_rank;
        }

        for (i = 0; i < comm_size; i++) {
            info->send_counts[i] = i;
            info->recv_counts[i] = comm_rank;
            info->recv_displs[i] = i * comm_rank;
            info->send_displs[i] = (i * (i + 1) / 2);
        }

        return info;
    }
}

static alltoallv_info_t *setup_alltoallv_balanced(MPI_Comm comm)
{
    int comm_size;
    printf("comm_size %u ",comm_size);
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
    for (i = 0; i < comm_size * comm_size; i++) {
        info->send_buffer[i] = i + 100 * comm_rank;
        info->recv_buffer[i] = -i;
    }

    for (i = 0; i < comm_size; i++) {
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

    char* is_balance= getenv("BALANCE");

    if (is_balance!=NULL) { // We create 2 subcommunicators

        // if (world_size != 160)
        // {
        //     fprintf(stderr, "This test is designed to run with 160 ranks\n");
        //     return EXIT_FAILURE;
        // }
        int color = world_rank / 2;
        MPI_Comm sub_comm;
        MPICHECK(MPI_Comm_split(MPI_COMM_WORLD, color, world_rank, &sub_comm));

        alltoallv_info_t *world_alltoallv = setup_alltoallv_unbalanced(MPI_COMM_WORLD);
        alltoallv_info_t *subcomm_alltoallv = setup_alltoallv_unbalanced(sub_comm);

        MPICHECK(do_alltoallv(subcomm_alltoallv));
        MPICHECK(do_alltoallv(world_alltoallv));
        MPICHECK(do_alltoallv(subcomm_alltoallv));

        finalize_alltoallv(&subcomm_alltoallv);
        finalize_alltoallv(&world_alltoallv);

        MPI_Finalize();
        return EXIT_SUCCESS;
    } else {
        /* Unbalanced Call
         0  1  2  3
         4  5  6  7
         8  9  10 11
         12 13 14 15
        */

        alltoallv_info_t *world_alltoallv = setup_alltoallv_balanced(MPI_COMM_WORLD);

        // return MPI_Alltoallv(info->send_buffer, info->send_counts, info->send_displs, MPI_INT,
        //                      info->recv_buffer, info->recv_counts, info->recv_displs, MPI_INT,
        //                      info->comm);

        print_buffer_int(world_alltoallv->send_buffer, world_size * world_size, strdup("sbuf:"), world_rank);
        print_buffer_int(world_alltoallv->send_counts, world_size, strdup("scount:"), world_rank);
        print_buffer_int(world_alltoallv->recv_counts, world_size, strdup("rcount:"), world_rank);
        print_buffer_int(world_alltoallv->send_displs, world_size, strdup("sdisp:"), world_rank);
        print_buffer_int(world_alltoallv->recv_displs, world_size, strdup("rdisp:"), world_rank);

        MPICHECK(do_alltoallv(world_alltoallv)); // for every row comm to setup

        print_buffer_int(world_alltoallv->recv_buffer, world_size * world_size, strdup("rbuf:"), world_rank);

        for (i = 0; i < world_size; i++) {
            int *p = world_alltoallv->recv_buffer + world_alltoallv->recv_displs[i];
            for (int j = 0; j < world_rank; j++) {
                if (p[j] != i * 100 + (world_rank * (world_rank + 1)) / 2 + j) {
                    printf("** Error: <%d> got %d expected %d for %dth\n", world_rank, p[j], (i * (i + 1)) / 2 + j, j);
                }
            }
        }

        printf("WORLD RANK/SIZE: %d/%d \t ROW RANK/SIZE: %d/%d\n", world_rank, world_size, world_rank, world_size);
        finalize_alltoallv(&world_alltoallv);
        MPI_Barrier(MPI_COMM_WORLD);
        MPI_Finalize();
        return EXIT_SUCCESS;
    }

exit_on_failure:
    MPI_Finalize();
    return EXIT_FAILURE;
}

/*************************************************************************
 * Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <unistd.h>
#include <mpi.h>

/* We do not care about Fortran here */

int MPI_Alltoallv(const void *sendbuf, const int *sendcounts, const int *sdispls,
                  MPI_Datatype sendtype, void *recvbuf, const int *recvcounts,
                  const int *rdispls, MPI_Datatype recvtype, MPI_Comm comm)
{
    int my_rank;
    PMPI_Comm_rank(comm, &my_rank);
    PMPI_Barrier(comm);
    if (my_rank == 0)
        sleep(1);

    return PMPI_Alltoallv(sendbuf, sendcounts, sdispls, sendtype,
                          recvbuf, recvcounts, rdispls, recvtype,
                          comm);
}
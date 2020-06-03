/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <mpi.h>

/* FORTRAN BINDINGS */
extern int mpi_fortran_in_place_;
#define OMPI_IS_FORTRAN_IN_PLACE(addr) \
    (addr == (void *)&mpi_fortran_in_place_)
extern int mpi_fortran_bottom_;
#define OMPI_IS_FORTRAN_BOTTOM(addr) \
    (addr == (void *)&mpi_fortran_bottom_)
#define OMPI_INT_2_FINT(a) a
#define OMPI_FINT_2_INT(a) a
#define OMPI_F2C_IN_PLACE(addr) (OMPI_IS_FORTRAN_IN_PLACE(addr) ? MPI_IN_PLACE : (addr))
#define OMPI_F2C_BOTTOM(addr) (OMPI_IS_FORTRAN_BOTTOM(addr) ? MPI_BOTTOM : (addr))

int _mpi_init(int *argc, char ***argv)
{
    return PMPI_Init(argc, argv);
}

int MPI_Init(int *argc, char ***argv)
{
    return _mpi_init(argc, argv);
}

int mpi_init_(MPI_Fint *ierr)
{
    int c_ierr;
    int argc = 0;
    char **argv = NULL;

    c_ierr = _mpi_init(&argc, &argv);
    if (NULL != ierr)
        *ierr = OMPI_INT_2_FINT(c_ierr);
}

int _mpi_finalize()
{
    return PMPI_Finalize();
}

int MPI_Finalize()
{
    return _mpi_finalize();
}

void mpi_finalize_(MPI_Fint *ierr)
{
    int c_ierr = _mpi_finalize();
    if (NULL != ierr)
        *ierr = OMPI_INT_2_FINT(c_ierr);
}

int _mpi_alltoallv(const void *sendbuf, const int *sendcounts, const int *sdispls,
                   MPI_Datatype sendtype, void *recvbuf, const int *recvcounts,
                   const int *rdispls, MPI_Datatype recvtype, MPI_Comm comm)
{
    return PMPI_Alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
}

int MPI_Alltoallv(const void *sendbuf, const int *sendcounts, const int *sdispls,
                  MPI_Datatype sendtype, void *recvbuf, const int *recvcounts,
                  const int *rdispls, MPI_Datatype recvtype, MPI_Comm comm)
{
    return _mpi_alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
}

void mpi_alltoallv_(void *sendbuf, MPI_Fint *sendcount, MPI_Fint *sdispls, MPI_Fint *sendtype,
                    void *recvbuf, MPI_Fint *recvcount, MPI_Fint *rdispls, MPI_Fint *recvtype,
                    MPI_Fint *comm, MPI_Fint *ierr)
{
    int c_ierr;
    MPI_Comm c_comm;
    MPI_Datatype c_sendtype, c_recvtype;

    c_comm = PMPI_Comm_f2c(*comm);
    c_sendtype = PMPI_Type_f2c(*sendtype);
    c_recvtype = PMPI_Type_f2c(*recvtype);

    sendbuf = (char *)OMPI_F2C_IN_PLACE(sendbuf);
    sendbuf = (char *)OMPI_F2C_BOTTOM(sendbuf);
    recvbuf = (char *)OMPI_F2C_BOTTOM(recvbuf);

    c_ierr = MPI_Alltoallv(sendbuf,
                           (int *)OMPI_FINT_2_INT(sendcount),
                           (int *)OMPI_FINT_2_INT(sdispls),
                           c_sendtype,
                           recvbuf,
                           (int *)OMPI_FINT_2_INT(recvcount),
                           (int *)OMPI_FINT_2_INT(rdispls),
                           c_recvtype, c_comm);
    if (NULL != ierr)
        *ierr = OMPI_INT_2_FINT(c_ierr);
}

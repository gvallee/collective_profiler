# Conversion from alltoallv

This direcotry was copied fromdirectory alltoallv

## Target signatures
from opempi.org
- FROM  
int MPI_Alltoallv(const void *sendbuf, const int sendcounts[],  
    const int sdispls[], MPI_Datatype sendtype,  
    void *recvbuf, const int recvcounts[],  
    const int rdispls[], MPI_Datatype recvtype, MPI_Comm comm)
- TO  
int MPI_Alltoall(const void *sendbuf, int sendcount,  
    MPI_Datatype sendtype, void *recvbuf, int recvcount,  
    MPI_Datatype recvtype, MPI_Comm comm)    
so const int sendcounts[], const int sdispls[] becomes int sendcount, and similarly for recevcounts

related public(ish) interfaces are: int _mpi_alltoallv, MPI_Alltoallv and mpi_alltoallv_ (latter is wrapper  for Fortran?) - first is the workhorse, second is a wrapper for C

Fortran interfaces are:

- FROM  
MPI_ALLTOALLV(SENDBUF, SENDCOUNTS, SDISPLS, SENDTYPE,  
    RECVBUF, RECVCOUNTS, RDISPLS, RECVTYPE, COMM, IERROR)  
    \<type\>    SENDBUF(*), RECVBUF(*)  
    INTEGER    SENDCOUNTS(*), SDISPLS(*), SENDTYPE  
    INTEGER    RECVCOUNTS(*), RDISPLS(*), RECVTYPE  
    INTEGER    COMM, IERROR  
- TO  
MPI_ALLTOALL(SENDBUF, SENDCOUNT, SENDTYPE, RECVBUF, RECVCOUNT,  
    RECVTYPE, COMM, IERROR)  
    \<type\>    SENDBUF(*), RECVBUF(*)  
    INTEGER    SENDCOUNT, SENDTYPE, RECVCOUNT, RECVTYPE  
    INTEGER    COMM, IERROR  


# Meaning of sendcout/recvcount
Study openmpi manual:

Stick with assuming sencount is the same across ranks

# Functions to save counts
_mpi_alltoall  
  if COMAPACT_FORMAT  -- this var indicates to buffer in memory  
  insert_sendrecv_data  
    same_call_counters  
      lookupRankSendCounters  
        lookup_rank_counters (logger.c)
      lookupRankRecvCounters
        lookup_rank_counters (logger.c)
    compareAndSaveSendCounters
    compareAndSaveRecvCounters
  -- there will be a writing of this data at MPI_Finalize
  MPI_Finalize
    _commit_data
      log_profiling_data (logger.c)
       log_data (logger.c)
         _log_data (logger.c)
           compress_int_array
           lookupRankCounters
	_finalize_profiling
      logger_fini (logger.c)
      _release_profiling_resources


  if !COMAPACT_FORMAT
    save_counts - writes one file per call - maybe have not seen this file yet! (more combinations of macro values)

checkout data structures as well
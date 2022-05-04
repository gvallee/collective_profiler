# Overview

The idea is extremely simple: we intercept all calls to `MPI_Alltoallv` using PMPI and we
artifically add a barrier, followed by an actual call to `MPI_Alltoallv` except
for rank 0 that sleep for 1 second. The expected result is that all rank will
arrive in the operation at the same time, except rank 0 that is expected to
be roughly 1 second late.

We do not care about Fortran, it is only supporting C.

# Execution

Like for any PMPI code, use `LD_PRELOAD`. For example:
```
LD_PRELOAD=/home/gvallee/Projects/collective_profiler/validation/late_arrival/lib_pmpi_late_arrival.so mpirun -np 2 examples/alltoallv_c
```
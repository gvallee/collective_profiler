#!/bin/bash

# module for HPCAC
spack unload --all
module purge
module load gcc/4.8.5 hpcx/2.7.0

# profiling environment variables
export A2A_PROFILING_OUTPUT_DIR=/global/home/users/cyrusl/placement/expt0066/alltoall_profiling/examples/results

MPIFLAGS="--mca pml ucx -x UCX_NET_DEVICES=mlx5_0:1 "
MPIFLAGS+="-x A2A_PROFILING_OUTPUT_DIR "
MPIFLAGS+="-x LD_LIBRARY_PATH "

# the alltoall program 
mpirun -np 8 \
        $MPIFLAGS \
       -x LD_PRELOAD=/global/home/users/cyrusl/placement/expt0066/alltoall_profiling/src/alltoall/liballtoall_counts.so \
       /global/home/users/cyrusl/placement/expt0066/alltoall_profiling/examples/alltoall
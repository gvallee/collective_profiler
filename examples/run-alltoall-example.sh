#!/bin/bash

# module for HPCAC
spack unload --all
module purge
module load gcc/8.3.1 hpcx/2.7.0

# profiling environment variables
export JOB_NOW=$( date +%Y%m%d-%H%M%S )
export PROJECT_ROOT=/global/home/users/cyrusl/placement/expt0070/alltoall_profiling

MPIFLAGS="--mca pml ucx -x UCX_NET_DEVICES=mlx5_1:1 "
MPIFLAGS+="-x A2A_PROFILING_OUTPUT_DIR "
MPIFLAGS+="-x LD_LIBRARY_PATH "

EXAMPLE_PROG=alltoall_simple_c

# the alltoall test programs:

# simple 1 call TODO select libraries
export A2A_PROFILING_OUTPUT_DIR=$PROJECT_ROOT/examples/results/run-at-${JOB_NOW}
EXAMPLE_PROG=alltoall_simple_c
mpirun -np 4 \
        $MPIFLAGS \
       -x LD_PRELOAD=$PROJECT_ROOT/src/alltoall/liballtoall_counts.so \
       $PROJECT_ROOT/examples/$EXAMPLE_PROG

# may calls, all the same  TODO select libraries
export A2A_PROFILING_OUTPUT_DIR=$PROJECT_ROOT/examples/results/run-at-${JOB_NOW}
EXAMPLE_PROG=alltoall_bigcounts_c
mpirun -np 4 \
        $MPIFLAGS \
       -x LD_PRELOAD=$PROJECT_ROOT/src/alltoall/liballtoall_counts.so \
       $PROJECT_ROOT/examples/$EXAMPLE_PROG
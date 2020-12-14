#!/bin/bash

# a build script for use at HPCAC
module purge
module load gcc/4.8.5 hpcx/2.7.0

cd /global/home/users/cyrusl/placement/expt0063/alltoall_profiling/alltoall
make clean
make

cd -
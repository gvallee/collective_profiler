#!/bin/bash

# a build script for use at HPCAC
module purge
module load gcc/8.3.1 hpcx/2.7.0

PROJECT_ROOT=/global/home/users/cyrusl/placement/expt0066/alltoall_profiling
cd $PROJECT_ROOT
make clean
make

cd -
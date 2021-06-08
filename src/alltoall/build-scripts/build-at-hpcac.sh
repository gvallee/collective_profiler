#!/bin/bash

# a build script for use at HPCAC
module purge
module load gcc/8.3.1 hpcx/2.7.0

PROJECT_ROOT=/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler
cd $PROJECT_ROOT
make clean
make

cd -
#!/bin/bash

cd /global/home/users/cyrusl/placement/expt0070/alltoall_profiling

module purge
spack unload --all

HNAME=$(hostname)

#if [[ ${HNAME:0:4} == "thor" ]]; then
module load gcc/8.3.1 hpcx/2.7.0 gnuplot/5.2.8
#else
#    module load gcc/4.8.5 hpcx/2.7.0  # these were used for compiling on Login node for use on Jupiter before change to Centos 8
#fi

export PATH=$GOPATH/bin:$PATH   
export LD_LIBRARY_PATH=$GOPATH/lib:$LD_LIBRARY_PATH 

env

make clean
make


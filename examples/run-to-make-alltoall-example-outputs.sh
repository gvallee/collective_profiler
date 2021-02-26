#!/bin/bash

# module for HPCAC
spack unload --all
module purge
module load gcc/8.3.1 hpcx/2.7.0

# one of these tests makes 1E6 files, so using temporary files, which are hoped to be in RAM
TMPROOT=$(mktemp -td alltoalltest.XXX)

# profiling environment variables
export JOB_NOW=$( date +%Y%m%d-%H%M%S )
export PROJECT_ROOT=/global/home/users/cyrusl/placement/expt0070/alltoall_profiling
declare -a EQUAL_SAMPLING_LIBS
declare -a UNEQUAL_SAMPLING_LIBS
EQUAL_SAMPLING_LIBS=( liballtoall_backtrace.so \
                        liballtoall_counts_compact.so \
                        liballtoall_counts.so \
                        liballtoall_exec_timings.so \
                        liballtoall_late_arrival.so \
                        liballtoall_location.so \
                        liballtoall.so )
UNEQUAL_SAMPLING_LIBS=( liballtoall_backtrace_counts_unequal.so \
                        liballtoall_counts_unequal_compact.so \
                        liballtoall_counts_unequal.so \
                        liballtoall_exec_timings_counts_unequal.so \
                        liballtoall_late_arrival_counts_unequal.so \
                        liballtoall_location_counts_unequal.so )
declare -a SAMPLING_LIBS
SAMPLING_LIBS=( "${EQUAL_SAMPLING_LIBS[@]}" "${UNEQUAL_SAMPLING_LIBS[@]}" )
# the test programs and sample libraryies 
#SAMPLING_LIBS=(liballtoall_counts_compact.so)
EXAMPLE_PROGS=(alltoall_simple_c alltoall_bigcounts_c alltoall_multicomms_c alltoall_dt_c)


# mpi stuff
HNAME=$(hostname)
if [[ "$HNAME" == "login01" ]]; then
    MPIFLAGS="--mca pml ucx -x UCX_NET_DEVICES=mlx5_1:1 "
elif [[ "$HNAME" == "login02" ]]; then
    MPIFLAGS="--mca pml ucx -x UCX_NET_DEVICES=mlx5_2:1 "
else
    MPIFLAGS=""
fi
MPIFLAGS+="-x LD_LIBRARY_PATH "

AN_MPI_ERROR=no_error
# loop round the programs and libs
for EXAMPLE_PROG in ${EXAMPLE_PROGS[@]}
do
    for SAMPLING_LIB in ${SAMPLING_LIBS[@]}
    do
        # export JOB_NOW=$( date +%Y%m%d-%H%M%S )
        RESULTS_DIR=$TMPROOT/prog_$EXAMPLE_PROG/sampler_$SAMPLING_LIB  # /runat_$JOB_NOW        
        mkdir -p $RESULTS_DIR
        export A2A_PROFILING_OUTPUT_DIR=$RESULTS_DIR
        echo "Calling mpirun "
        echo "    - for $EXAMPLE_PROG"
        echo "    - using sampler $SAMPLING_LIB"
        echo "    - with results at $RESULTS_DIR"
        mpirun -np 4 \
            $MPIFLAGS \
            -x LD_PRELOAD=$PROJECT_ROOT/src/alltoall/$SAMPLING_LIB \
            -x A2A_PROFILING_OUTPUT_DIR \
            $PROJECT_ROOT/examples/$EXAMPLE_PROG
        if [ $? -ne 0 ]; then
            AN_MPI_ERROR=at_least_one_error
        fi
    done
done 

if [[ "$AN_MPI_ERROR" == "no-error" ]]; then
    # create folders for test answers
    for EXAMPLE_PROG in ${EXAMPLE_PROGS[@]}
    do
        mkdir -p $PROJECT_ROOT/tests/alltoall/equalcounts/$EXAMPLE_PROG
        mkdir -p $PROJECT_ROOT/tests/alltoall/unequalcounts/$EXAMPLE_PROG
    fi

    # populate those folders with the test answers

fi

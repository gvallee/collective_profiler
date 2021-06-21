#!/bin/bash

# module for HPCAC
spack unload --all
module purge
spack load gcc@11

module load intel/2019u4  openmpi/4.0.1


# one of these tests makes 1E6 files, so using temporary files, which are hoped to be in RAM
TMPROOT=$(mktemp -td alltoalltest.XXX)

# profiling environment variables
export JOB_NOW=$( date +%Y%m%d-%H%M%S )
export PROJECT_ROOT=/global/home/users/cyrusl/placement/expt0070/alltoall_profiling
declare -a EQUAL_SAMPLING_LIBS
declare -a UNEQUAL_SAMPLING_LIBS
EQUAL_SAMPLING_LIBS=(   liballtoall_counts_compact.so \
                        liballtoall_counts.so \
                        liballtoall_exec_timings.so \
                        liballtoall_late_arrival.so \
                        liballtoall_location.so \ 
                        liballtoall_backtrace.so)  
                        # liballtoall.so )  # TO DO - what is this library for - is it equal or unequal counts? 
UNEQUAL_SAMPLING_LIBS=( liballtoall_counts_unequal_compact.so \
                        liballtoall_counts_unequal.so \
                        liballtoall_exec_timings_counts_unequal.so \
                        liballtoall_late_arrival_counts_unequal.so \
                        liballtoall_location_counts_unequal.so \
                        liballtoall_backtrace_counts_unequal.so) 

# make selection of refernce files to generate
declare -a SAMPLING_LIBS
SAMPLING_LIBS=(  "${UNEQUAL_SAMPLING_LIBS[@]}" )  # "${EQUAL_SAMPLING_LIBS[@]}"
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
        # echo "DEBUG: AN_MPI_ERROR=$AN_MPI_ERROR"
    done
done 

# TODO reinstate commented if below when code fixed so that all mpirun above run without error
if [[ "$AN_MPI_ERROR" == "no_error" ]]; then
    echo "Copying results to unchecked directories at tests/alltoall*"
    for EXAMPLE_PROG in ${EXAMPLE_PROGS[@]}
    do
        # create folders for test answers
        mkdir -p $PROJECT_ROOT/tests/$EXAMPLE_PROG/unchecked
        mkdir -p $PROJECT_ROOT/tests/$EXAMPLE_PROG/expectedOutput
        # clean out any old content
        rm -f $PROJECT_ROOT/tests/$EXAMPLE_PROG/unchecked/*
        rm -f $PROJECT_ROOT/tests/$EXAMPLE_PROG/expectedOutput/*
        # populate those folders with the test answers
        for SAMPLING_LIB in ${UNEQUAL_SAMPLING_LIBS[@]}
        do
            if [[ "$SAMPLING_LIB" == "liballtoall_counts_unequal.so" ]]; then
                cp $TMPROOT/prog_$EXAMPLE_PROG/sampler_$SAMPLING_LIB/counts.rank0_call0.md \
                $PROJECT_ROOT/tests/$EXAMPLE_PROG/unchecked/
            else
                cp $TMPROOT/prog_$EXAMPLE_PROG/sampler_$SAMPLING_LIB/* \
                $PROJECT_ROOT/tests/$EXAMPLE_PROG/unchecked/
            fi        

        done
        for SAMPLING_LIB in ${EQUAL_SAMPLING_LIBS[@]}
        do
            if [[ "$SAMPLING_LIB" == "liballtoall.so" ]]; then 
            # TODO what do these sampling lib(s) do and what results does it make
            # for now do noop
                :
            else
                if [[ "$SAMPLING_LIB" == "liballtoall_counts.so" ]]; then
                    cp $TMPROOT/prog_$EXAMPLE_PROG/sampler_$SAMPLING_LIB/counts.rank0_call0.md \
                    $PROJECT_ROOT/tests/$EXAMPLE_PROG/unchecked/
                else
                    cp $TMPROOT/prog_$EXAMPLE_PROG/sampler_$SAMPLING_LIB/* \
                    $PROJECT_ROOT/tests/$EXAMPLE_PROG/unchecked/
                fi        
            fi
        done
    done
else
    echo "NOT Copying results to unchecked directories at tests/alltoall* - because there was an error in the mpiruns"
fi

# tidy up by deleting temp files created above.
rm -r $TMPROOT

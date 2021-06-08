#!/bin/sh -l
# sbatch parameters following an example from the Internet at https://help.rc.ufl.edu/doc/Sample_SLURM_Scripts 
#SBATCH --job-name=alltoall          # Job name
#SBATCH --mail-type=ALL                     # Mail events (NONE, BEGIN, END, FAIL, ALL)
#SBATCH --mail-user=yangyiwei2000@gmail.com     # Where to send mail	
#SBATCH --nodes=4
#SBATCH --ntasks=160                     
#SBATCH --ntasks-per-node=40
##SBATCH --mem=128                          # Job memory request
#SBATCH --time=00:20:00                     # Time limit hrs:min:sec
#SBATCH --output=/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/alltoall_%j.out     # Standard output and error log
#SBATCH --error=/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/alltoall_%j.err
#SBATCH -p compute                          # which section of the cluster 

# expecting that this variable will be copied to the compute nodes
# where .bashrc will test it and set no environment if it is set
export SUPPRESS_BASHRC=1 #this is pointless - bashrc will have been run already!!

# discover the name and path of this script
# maybe not useful for identifying the path of the project because batch job because SLURM takes copy of script so path is changed 
THIS_SCRIPT=$(readlink --canonicalize --no-newline "$0")
THIS_SCRIPT_FILENAME=$(basename "$THIS_SCRIPT")
THIS_SCRIPT_DIR=$(dirname "$THIS_SCRIPT")

# environment and modules and some paths etc. for the job 
# /global/home/users/cyrusl/placement/expt0060/OSU/osu-micro-benchmarks-5.6.3/install/libexec/osu-micro-benchmarks/mpi/collective
export PROJECT_ROOT=/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/
# TODO - set modulefiles!!?
module purge
HNAME=$(hostname)

# should not need this - no environment variable means no spack modules loaded
# which spack
# spack unload --all
spack load gcc@11

module load intel/2019u4  openmpi/4.0.1

export JOB_NOW=$( date +%Y%m%d-%H%M%S )
export RESULTS_ROOT=${PROJECT_ROOT}/examples/results_task1_balance/run-at-${JOB_NOW}  #-${THIS_SCRIPT_FILENAME}
# TODO THIS-SCRIPT_FILENAME gets changed by sbatch to "slurm-script" - detect that and replace somehow with original

# makes the results directory and somewhere to put results of post processing.
mkdir -p "${RESULTS_ROOT}/analysis"
mkdir -p "${RESULTS_ROOT}/ranks"

# TO DO put this in brackets to end and tee to file
# or accept current solution of copying the slurm log file to the results dir

echo "========================================================="
echo "          START: This is the batch script" 
echo "========================================================="

# report creating the results dir
echo
echo "results directory created at: $RESULTS_ROOT"

# slurm stats
echo "recording Slurm job stats at beginning of job ..."
sstat -j "$SLURM_JOB_ID" > "$RESULTS_ROOT/slurm_stats_at_start.log" 

echo
echo "recording env ..."
env > "$RESULTS_ROOT/env.log"

echo
echo "recording ompi_info ..."
ompi_info > "$RESULTS_ROOT/ompi_info.log"

echo
echo "recording SLURM variables ..."
env | grep SLURM > "$RESULTS_ROOT/slurm_variable.log"

# commented out becuase$ SLURM_CONF does not exist (genreally or on this cluster?)
# echo
# echo "recording SLURM configuration ..."
# eval $( grep SLURM_CONF "$RESULTS_ROOT/slurm_variable.log" )
# cp "$SLURM_CONF" "$RESULTS_ROOT/"

# copy this script to the results directory
echo
echo "recording a copy of this script ..."
cp "$THIS_SCRIPT" "${RESULTS_ROOT}/${THIS_SCRIPT_FILENAME}.copy"

# create post processing scripts
echo
echo "creating post processing scripts to use in results dir..."

# script to copy slurm output, as indicated by sbatch option --output=, to the results dir
# 'EOF' so variables are expanded at runtime of the script below
cat - > "$RESULTS_ROOT/copy_slurm_output_here.sh" << 'EOF' 
#!/bin/bash
RESULTS_ROOT=$(dirname "$0")
# source "$RESULTS_ROOT/slurm_variable.log" # does not work because file has illegal values for bash variables
eval $( grep SLURM_SUBMIT_DIR "$RESULTS_ROOT/slurm_variable.log" )
eval $( grep SLURM_JOB_NAME  "$RESULTS_ROOT/slurm_variable.log" )
eval $( grep SLURM_JOB_ID  "$RESULTS_ROOT/slurm_variable.log" )
SLURM_OUTPUT_FILE=$(ls "$SLURM_SUBMIT_DIR" | grep "$SLURM_JOB_NAME" | grep "$SLURM_JOB_ID")
echo "copying $SLURM_SUBMIT_DIR/$SLURM_OUTPUT_FILE here ..."
cp "$SLURM_SUBMIT_DIR/$SLURM_OUTPUT_FILE" "$RESULTS_ROOT"
chmod a=r "$RESULTS_ROOT/$SLURM_OUTPUT_FILE"
EOF

cat - > "$RESULTS_ROOT/analyze.sh" << 'EOF'
#!/bin/bash
# somewhere to keep the results of post processing the results of the cluster job
export RESULTS_ROOT=$( dirname $(readlink --canonicalize --no-newline "$0" ) )
export POST_ANALYSYS_ROOT="$RESULTS_DIR/post_processed"
echo "this script is as yet a dummy and has set only some paths - no analysis performed
# TODO call some post processing scripts
# TODO copy the post processing scripts to the post processing directory for a record copy
# TODO set results of postprocessing to read only
# TODO test all this including the exports above
EOF

# set variables for the mpirun executable - repeat this section if more than one
# full path? (which below help ldd find executable)
export EXECUTABLE1=/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/examples/task1_balance_c
export EXECUTABLE1_PARAMS=""

# following example at /global/home/users/cyrusl/placement/expt0060/geoffs-profiler/build-570ff3aff83fa208f3d1e2fcbdb31d9ec7e93b6c/README.md
# TODO put in the results dir
export A2A_PROFILING_OUTPUT_DIR=$RESULTS_ROOT
ALLTOALL_LIB_ROOT=/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/src/alltoallv
COUNTSFLAGS="$ALLTOALL_LIB_ROOT/liballtoallv_counts_notcompact.so"
MAPFLAGS="$ALLTOALL_LIB_ROOT/liballtoallv_location.so"
BACKTRACEFLAGS="$ALLTOALL_LIB_ROOT/liballtoallv_backtrace.so"
A2ATIMINGFLAGS="$ALLTOALL_LIB_ROOT/liballtoallv_exec_timings.so"
LATETIMINGFLAGS="$ALLTOALL_LIB_ROOT/liballtoallv_late_arrival.so"

MPIFLAGS="--mca pml ucx -x UCX_NET_DEVICES=mlx5_0:1 "
MPIFLAGS+="-x A2A_PROFILING_OUTPUT_DIR "
MPIFLAGS+="-x LD_LIBRARY_PATH "
MPIFLAGS+="-np 160 -bind-to core "
MPIFLAGS+="--mca pml_base_verbose 100 --mca btl_base_verbose 100 " 
# --output-file# with mulltiple mpiruns this causes subsequent ones to overwrite the output files!

# the mpirun commands
declare -a MPIRUN_COMMANDS 
MPIRUN_COMMANDS[0]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/counts     -x LD_PRELOAD=$COUNTSFLAGS     $EXECUTABLE1 $EXECUTABLE1_PARAMS"
MPIRUN_COMMANDS[1]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/map        -x LD_PRELOAD=$MAPFLAGS        $EXECUTABLE1 $EXECUTABLE1_PARAMS"
MPIRUN_COMMANDS[2]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/backtrace  -x LD_PRELOAD=$BACKTRACEFLAGS  $EXECUTABLE1 $EXECUTABLE1_PARAMS"
MPIRUN_COMMANDS[3]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/a2atiming  -x LD_PRELOAD=$A2ATIMINGFLAGS  $EXECUTABLE1 $EXECUTABLE1_PARAMS"
MPIRUN_COMMANDS[4]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/latetiming -x LD_PRELOAD=$LATETIMINGFLAGS $EXECUTABLE1 $EXECUTABLE1_PARAMS"

echo
# TODO - some more of vars set above
echo "recording basic job details ..."
{
    echo "alltoallv sampling test script"
    echo "SCRIPT NAME             : $THIS_SCRIPT_FILENAME"
    echo "SCRIPT DIR              : $THIS_SCRIPT_DIR"
    echo "(the scheduler may have made a copy at a location other than the source)"
    echo "PROJECT_ROOT            : $PROJECT_ROOT"
    echo "RESULTS_ROOT            : $RESULTS_ROOT"
    echo "HOSTNAME                : $(hostname)"
    echo "USER                    : $USER"
    echo "JOB_NOW                 : ${JOB_NOW}"
    echo "(note that this the local time on the cluster, so California time)"
    echo "which mpirun            : $(which mpirun)"
    echo "mpirun --version ..." 
    mpirun --version
    echo "module list ..."
    module list
    echo "spack env status ..."
    spack env status
    echo "EXECUTABLE1             : $EXECUTABLE1"
    echo "EXECUTABLE1_PARAMS      : $EXECUTABLE1_PARAMS"
    echo "MPIFLAGS                : $MPIFLAGS"
    echo "A2A_PROFILING_OUTPUT_DIR: $A2A_PROFILING_OUTPUT_DIR"
} |& tee "$RESULTS_ROOT/basic_job_details.log"
# |& because module prints its info to stderr

# record the ldd
# TODO in this example are using PRELOAD so this may not be giving the right information to compare to that
echo
echo "recording ldd for the executables ..."
ldd "$(which $EXECUTABLE1)" > "${RESULTS_ROOT}/$(basename $EXECUTABLE1).ldd" 
echo  "in this example are using PRELOAD so this may not be giving the right information to compare to that" >> "${RESULTS_ROOT}/$(basename $EXECUTABLE1).ldd" 
# TODO check ldd results are as expected

# Record the mpirun command
echo
echo "recording the mpirun command ..."
# EOF w/o quotes so variables evaluated now
cat - > "$RESULTS_ROOT/mpirun_command1.log" << EOF
${MPIRUN_COMMANDS[0]}
${MPIRUN_COMMANDS[1]}
${MPIRUN_COMMANDS[2]}
${MPIRUN_COMMANDS[3]}
${MPIRUN_COMMANDS[4]}
EOF

# mpirun section
echo "Now calling mpirun ..."
echo "- stdout and stderr of this mpirun will be in " 
echo "  the results directory but appear also below"
echo "- stdout and stderr of the respective MPI ranks will be in" 
echo "  subdirectories of that and are not shown here "
echo "  if mpirun uses --output-file"
echo "*********************************************************"
echo 
#{ {
    for MPIRUN_COMMAND in "${MPIRUN_COMMANDS[@]}"
        do
        echo "mpirun command will be: $MPIRUN_COMMAND"
        $MPIRUN_COMMAND
        echo "... end of that mpirun"
    done
#} > >( tee ${RESULTS_ROOT}/mpirun.stdout.log ); } \
# 2> >( tee ${RESULTS_ROOT}/mpirun.stderr.log 1>&2 )
# if more than one mpirun the name in previous line should be distinguished
# the tee arrangements follow https://stackoverflow.com/questions/21465297/tee-stdout-and-stderr-to-separate-files-while-retaining-them-on-their-respective
echo
echo "*********************************************************"
echo "... mpirun complete"
# slurm stats
echo 
echo "recording Slurm job stats at end of job ..."
sstat -j "$SLURM_JOB_ID" > "$RESULTS_ROOT/slurm_stats_at_end.log" 

echo
echo "setting the files of the results directory to read only ..."
find "$RESULTS_ROOT" -type d -exec chmod ug=rwx,o=rx {} \;
find "$RESULTS_ROOT" -type f -exec chmod ug=r,o=r {} \;

echo
echo "adding execute permission to post processing scripts ..."
chmod ug+x "$RESULTS_ROOT/copy_slurm_output_here.sh"

echo 
echo "you can see the results at $RESULTS_ROOT"

echo
echo "========================================================="
echo "            END: This is the batch script" 
echo "========================================================="

#!/bin/sh -l
# sbatch parameters following an example from the Internet at https://help.rc.ufl.edu/doc/Sample_SLURM_Scripts 
#SBATCH --job-name=alltoall          # Job name
#SBATCH --mail-type=ALL                     # Mail events (NONE, BEGIN, END, FAIL, ALL)
#SBATCH --mail-user=j.legg.17@ucl.ac.uk     # Where to send mail	
#SBATCH --nodes=8
#SBATCH --ntasks=8                     
#SBATCH --ntasks-per-node=1
##SBATCH --mem=128                          # Job memory request
#SBATCH --time=00:20:00                     # Time limit hrs:min:sec
#SBATCH --output=alltoall_%j.out     # Standard output and error log
#SBATCH --error=alltoall_%j.err
#SBATCH -p jupiter                          # which section of the cluster 

##SBATCH -w xxxx                            # particular nodes?

###SBATCH --export=NONE   # SLURM is opt out for passing through the environment - unlike SGE!!!

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
export PROJECT_ROOT=/global/home/users/cyrusl/placement/expt0066
# TODO - set modulefiles!!?
module purge
HNAME=$(hostname)
#if [[ ${HNAME:0:4} == "thor" ]]; then
    module load gcc/8.3.1 hpcx/2.7.0
#else
#    module load gcc/4.8.5 hpcx/2.7.0  # these were used for compiling on Login node for use on Jupiter
#fi

# should not need this - no environment variable means no spack modules loaded
# which spack
# spack unload --all

export JOB_NOW=$( date +%Y%m%d-%H%M%S )
export RESULTS_ROOT=${PROJECT_ROOT}/alltoall_profiling/examples/results/run-at-${JOB_NOW} #-${THIS_SCRIPT_FILENAME}
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
export EXECUTABLE1=/global/home/users/cyrusl/placement/expt0066/alltoall_profiling/examples/alltoall
export EXECUTABLE1_PARAMS=""

# following example at /global/home/users/cyrusl/placement/expt0060/geoffs-profiler/build-570ff3aff83fa208f3d1e2fcbdb31d9ec7e93b6c/README.md
# TODO put in the results dir

ALLTOALL_LIB_ROOT=/global/home/users/cyrusl/placement/expt0066/alltoall_profiling/src/alltoall
declare -a COUNTSFLAGS
COUNTSFLAGS[0]="$ALLTOALL_LIB_ROOT/liballtoall_counts.so"
COUNTSFLAGS[1]="$ALLTOALL_LIB_ROOT/liballtoall_counts_unequal.so"
COUNTSFLAGS[2]="$ALLTOALL_LIB_ROOT/liballtoall_counts_compact.so"
COUNTSFLAGS[3]="$ALLTOALL_LIB_ROOT/liballtoall_counts_unequal_compact.so"


declare -a RESULTS_SUB
RESULTS_SUB[0]="equal_counts"
RESULTS_SUB[1]="unequal_counts"
RESULTS_SUB[2]="equal_counts_compact"
RESULTS_SUB[3]="unequal_counts_compact"


MPIFLAGS="--mca pml ucx -x UCX_NET_DEVICES=mlx5_0:1 "
MPIFLAGS+="-x A2A_PROFILING_OUTPUT_DIR "
MPIFLAGS+="-x LD_LIBRARY_PATH "
MPIFLAGS+="-np 8 -map-by ppr:1:node -bind-to core "
MPIFLAGS+="--mca pml_base_verbose 100 --mca btl_base_verbose 100 " 
# --output-file# with mulltiple mpiruns this causes subsequent ones to overwrite the output files!

# the mpirun commands
declare -a MPIRUN_COMMANDS 
MPIRUN_COMMANDS[0]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/${RESULTS_SUB[0]} -x LD_PRELOAD=${COUNTSFLAGS[0]} $EXECUTABLE1 $EXECUTABLE1_PARAMS"
MPIRUN_COMMANDS[1]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/${RESULTS_SUB[1]} -x LD_PRELOAD=${COUNTSFLAGS[1]} $EXECUTABLE1 $EXECUTABLE1_PARAMS"
MPIRUN_COMMANDS[2]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/${RESULTS_SUB[2]} -x LD_PRELOAD=${COUNTSFLAGS[2]} $EXECUTABLE1 $EXECUTABLE1_PARAMS"
MPIRUN_COMMANDS[3]="mpirun $MPIFLAGS --output-filename $RESULTS_ROOT/${RESULTS_SUB[3]} -x LD_PRELOAD=${COUNTSFLAGS[3]} $EXECUTABLE1 $EXECUTABLE1_PARAMS"


echo
# TODO - some more of vars set above
echo "recording basic job details ..."
{
    echo "alltoall sampling test script"
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
idx=0
 for MPIRUN_COMMAND in "${MPIRUN_COMMANDS[@]}"
  do
    export A2A_PROFILING_OUTPUT_DIR="$RESULTS_ROOT/${RESULTS_SUB[${idx}]}"
    mkdir -p $A2A_PROFILING_OUTPUT_DIR
    echo "mpirun command will be: $MPIRUN_COMMAND"

    $MPIRUN_COMMAND \
    ${RESULTS_ROOT}/${RESULTS_SUB[${idx}]}/mpirun.stdout \
    ${RESULTS_ROOT}/${RESULTS_SUB[${idx}]}/mpirun.stderr

    echo "... results stored at ${RESULTS_ROOT}/${RESULTS_SUB[${idx}]}/mpirun.stdout and .../stderr"
    echo "... end of that mpirun"
# } > >( tee "${RESULTS_ROOT}/${RESULTS_SUB[${idx}]}/mpirun.stdout"  ); } \
# 2> >( tee "${RESULTS_ROOT}/${RESULTS_SUB[${idx}]}/mpirun.stderr" 1>&2 )
let "idx=idx+1"
done

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

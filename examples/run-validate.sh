
#!/bin/bash

# module for HPCAC
spack unload --all
module purge
spack load gcc@11

module load intel/2019u4  openmpi/4.0.1


export PROJECT_ROOT=/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/
export JOB_NOW=$( date +%Y%m%d-%H%M%S )
export RESULTS_ROOT=${PROJECT_ROOT}/examples/$1
mkdir -p $RESULTS_ROOT

cd /home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/tools/cmd/validate
 ./validate -webui | tee $RESULTS_ROOT/$1.log

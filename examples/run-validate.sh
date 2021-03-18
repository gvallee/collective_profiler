
#!/bin/bash

# module for HPCAC
spack unload --all
module purge
module load gcc/8.3.1 hpcx/2.7.0 gnuplot/5.2.8

export PROJECT_ROOT=/global/home/users/cyrusl/placement/expt0070/alltoall_profiling 
export JOB_NOW=$( date +%Y%m%d-%H%M%S )
export RESULTS_ROOT=${PROJECT_ROOT}/examples/results_validation/
mkdir -p $RESULTS_ROOT

cd /global/home/users/cyrusl/placement/expt0070/alltoall_profiling/tools/cmd/validate
 ./validate -webui | tee $RESULTS_ROOT/run-at-${JOB_NOW}.log

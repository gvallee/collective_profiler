# this script set the environment and so should be sourced.
# also useful for settingh the runtime environment to match

module purge
spack unload --all

module load gcc/8.3.1 hpcx/2.7.0 gnuplot/5.2.8
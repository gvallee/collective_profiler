# Introduction

This repository provides a set of tools and libraries to profile Alltoallv MPI calls
without having to modify applications. Profiling is performed in two phases:
- creation of traces on the execution platform,
- analysis of the traces.
Users can find a makefile at the top directory of the repository. This makefile  
compiles both the shared libraries for the creation of traces and the tools for post-mortem
analysis. Note that the shared library is implemented in C, while the tools are
implemented in Go. It is therefore not unusual to only compile the shared library on
the execution platform and compile the analysis tool on the system where post-mortem
analysis is performed. To only compile the shared libraries only, execute `make alltoallv`;
to compile the post-mortem analysis tools, execute `make tool`.

# Creation of the profiling trace

The alltoallv folder of the repository gathers all the code required to compile a
shared library. That shared library is a PMPI implementation of MPI_Alltoallv, i.e.,
all calls to MPI_Alltoallv will be intercepted, data gathered and finally the actual
MPI_Alltoallv operation executed.

## Installation

### Requirements

It is adviced to compile the profiling tool with the same compilers than the ones
used to compile the application, including MPI.

### Configuration

The profiling shared library can gather a lot of data and while it is possible to
gather all the data at once, it is not encouraged to do so. We advice to follow the
following there steps. For convenience, multiple shared libraries are being generated
with the adequate configuration for these steps.
- Gather the send/receive counts associated to the alltoallv calls: use the 
`liballtoallv_counts.so` library. This generates files based on the following naming
scheme: 
`send-counters.job<JOBID>.rank<RANK>.txt` and `recv-counters.job<JOBID>.rank<RANK>.txt`;
where:
    - `JOBID` is the job number when using a job scheduler (e.g., SLURM) or `0` when no
job scheduler is used or the value assigned to the `SLURM_JOB_ID` environment variable
users decide to set (it is strongly advised to set the `SLURM_JOB_ID` environment variable
to specify a unique identifier when not using a job scheduler).
    - `RANK` is the rank number on `MPI_COMM_WORLD` that is rank 0 on the communicator used 
for the alltoallv operations. Note that this means that if the
application is executing alltoallv operations on different communicators, the tool
generates multiple counter files, one per communicator. If the application is only using
`MPI_COMM_WORLD`, the two output files are named `send-counters.job<JOBID>.rank0.txt` and
`recv-counters.job<JOBID>.rank0.txt`.
Using these two identifiers makes it easier to handle multiple traces from multiple 
applications and/or platforms.
- Gather timings: use the `liballtoallv_timings.so` shared library. This generates
by default multiple files based on the following naming scheme:
 `late-arrivals-timings.job<JOBID>.rank<RANK>.md` and `a2a-timings.job<JOBID>.rank<RANK>.md`. 
- Gather backtraces: use the `liballtoallv_backtrace.so` shared library. This generates
files `backtrace_rank<RANK>_call<ID>.md`, *one per alltoallv call*, all of them stored in a `backtraces`
directory. In other words, this generates one file per alltoallv call, where `<ID>` is the
alltoallv call number on the communicator (starting at 0).
- Gather location: use the `liballtoallv_location.so` shared library. This generates files
`location_rank<RANK>_call<ID>.md`, *one per alltoallv call*. In other words, this generates 
one file per alltoallv call, where `<ID>` is the alltoallv call number on the communicator 
(starting at 0).

### Compilation

From the top directory of the repository source code, execute: `make libraries`.
This requires to have MPI available on the system.

This creates the following shared libraries:
- `liballtoallv_counts.so`,
- `liballtoallv_exec_timings.so`,
- `liballtoallv_late_arrival.so`,
- `liballtoallv_backtrace.so`,
- `liballtoallv_location.so`,
- and `alltoallv/liballtoallv.so`.
For most cases, the first 4 libraries are all users need.

### Execution

Before running the application to get traces, users have the option to customize the
tool behavior, mainly setting the place where the output files are stored (if not specified,
the current directory) by using the `A2A_PROFILING_OUTPUT_DIR` environment variable.

Like any PMPI option, users need to use `LD_PRELOAD` while executing their application.

On a platform where `mpirun` is directly used, the command to start the application
looks like:
```
LD_PRELOAD=$HOME<path_to_repo>/alltoallv/liballtoallv.so mpirun --oversubscribe -np 3 app.exe 
```

On a platform where a job manager is used, such as Slurm, users need to update the
batch script used to submit an application run. For instance, with Open MPI and Slurm,
it would look like:
```
mpirun -np $NPROC -x LD_PRELOAD=/global/home/users/geoffroy/projects/alltoall_profiling/alltoallv/liballtoallv_counts.so app.exe
```

When using a job scheduler, users are required to correctly set the LD_PRELOAD details
in their scripts or command line.

### Example

Assuming Slurm is used to execute jobs on the target platform, the following is an example of
a Slurm batch script that runs the OSU microbenchmakrs and gathers all the profiling traces
supported by our tool:

```
#!/bin/bash
#SBATCH -p cluster
#SBATCH -N 32
#SBATCH -t 05:00:00
#SBATCH -e alltoallv-32nodes-1024pe-%j.err
#SBATCH -o alltoallv-32nodes-1024pe-%j.out

set -x

module purge
module load gcc/4.8.5 ompi/4.0.1

export A2A_PROFILING_OUTPUT_DIR=/shared/data/profiling/osu/alltoallv/traces1

COUNTSFLAGS="/path/to/profiler/code/alltoall_profiling/alltoallv/liballtoallv_counts.so"
MAPFLAGS="/path/to/profiler/code/alltoall_profiling/alltoallv/liballtoallv_location.so"
BACKTRACEFLAGS="/path/to/profiler/code/alltoall_profiling/alltoallv/liballtoallv_backtrace.so"
A2ATIMINGFLAGS="/path/to/profiler/code/alltoall_profiling/alltoallv/liballtoallv_exec_timings.so"
LATETIMINGFLAGS="/path/to/profiler/code/alltoall_profiling/alltoallv/liballtoallv_late_arrival.so"

MPIFLAGS="--mca pml ucx -x UCX_NET_DEVICES=mlx5_0:1 "
MPIFLAGS+="-x A2A_PROFILING_OUTPUT_DIR "
MPIFLAGS+="-x LD_LIBRARY_PATH "

mpirun -np 1024 -map-by ppr:32:node -bind-to core $MPIFLAGS -x LD_PRELOAD="$COUNTSFLAGS" /path/to/osu/install/osu-5.6.3/libexec/osu-micro-benchmarks/mpi/collective/osu_alltoallv -f
mpirun -np 1024 -map-by ppr:32:node -bind-to core $MPIFLAGS -x LD_PRELOAD="$MAPFLAGS" /path/to/osu/install/osu-5.6.3/libexec/osu-micro-benchmarks/mpi/collective/osu_alltoallv -f
mpirun -np 1024 -map-by ppr:32:node -bind-to core $MPIFLAGS -x LD_PRELOAD="$BACKTRACEFLAGS" /path/to/osu/install/osu-5.6.3/libexec/osu-micro-benchmarks/mpi/collective/osu_alltoallv -f
mpirun -np 1024 -map-by ppr:32:node -bind-to core $MPIFLAGS -x LD_PRELOAD="$A2ATIMINGFLAGS" /path/to/osu/install/osu-5.6.3/libexec/osu-micro-benchmarks/mpi/collective/osu_alltoallv -f
mpirun -np 1024 -map-by ppr:32:node -bind-to core $MPIFLAGS -x LD_PRELOAD="$LATETIMINGFLAGS" /path/to/osu/install/osu-5.6.3/libexec/osu-micro-benchmarks/mpi/collective/osu_alltoallv -f
``` 

# Post-mortem analysis

We provide a set of tools that parses and analyses the data compiled when executing
the application with our shared library. These tools are implemented in Go and all of
them except one (`analyzebacktrace`) can be executed on a different platform.

## Installation

Most MPI applications are executed on a system where users cannot install system 
software, i.e., can be installed without privileges. Furthermore many systems do not
provide a Go installation. We therefore advice the following installation and 
configuration when users want to enable backtrace analysis on the computing
platform:
- Go to `https://golang.org/dl/` and download the appropriate package. For most Linux
users, it is the `go<version>.linux.amd64.tar.gz` package.
- Decompress the package in your home directory, for example: 
`cd ~ ; tar xzf go<version>.linux.amd64.tar.gz`.
- Edit your `~/.bashrc` file and add the following content:
```
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH
export LD_LIBRARY_PATH=$GOPATH/lib:$LD_LIBRARY_PATH 
```
Users can then either logout/log back in, or source their `.bashrc` file and Go will be
fully functional

Once Go installed, compiling the tool only requires one command that needs to be
executed from the top directory of the repository source code: `make tools`

## Dependencies between tools

The set of post-mortem analysis can use data generated by the profiler, intermediate data and/or
data generated by other tools, creating a chain of dependencies. [A graphic shows the internal
dependencies](doc/tool_dependencies.png) for the main tools provided by this project.

# Visualization using the WebUI

The project provides a WebUI to visualize the data. The interface assumes that postmortem
analysis (see [post-mortem analysis section](#post-mortem-analysis) for details).

## Installation

The WebUI is a part of the project's infrastructure, similarily to all the other tools and
is therefore compiled at the same time than the rest of the tools. To compile the tools,
simply run the `make tools` command from the top directory of the source code, or `make` from the `tools` sub-directory.

## Execution

From the top directory, execute the following command: 
`./tools/cmd/webui/webui -basedir <PATH/TO/THE/DATASET>`, where `<PATH/TO/THE/DATASET>` is 
the path to the directory where all the profiling data (both raw data or post-mortem 
analysis results) are.

For more information about all the parameters available when invoking the webui, execute
the following command: `./tools/cmd/webui/webui -h`.

Once started, the webui command starts a HTTP server on port 8080 and can be accessed using
any browser.

The webui is composed on two main panels:
- the top tabs to access either the list of calls or the detected patterns (result of the
post-mortem analysis),
- the main panel to display data based on the selected tab.
When selecting the tab to access calls' details, the main panel is composed of two 
sub-panels: the list of all the calls on the left hand side and the rest of the main panel
to display call-specific details when a call is selected from the list. These details 
consist of a graph providing:
- the amount of data sent/received per rank,
- the execution time per rank,
- the arrival time per rank,
- the bandwidth per rank.
Under the graph appears the raw send and receive counters.
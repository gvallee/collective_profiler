# Introduction

This repository provides a set of tools and a library to profile Alltoallv MPI calls
without having to modify applications. Profiling is performed in two phases:
- creation of a trace on the execution platform,
- analysis of the traces.
Users can find a makefile at the top directory of the repository. This makefile will 
compile both a shared library for the creation of traces and the tools for post-mortem
analysis. Note that the shared library is implemented in C, while the tools are
implemented in Go. It is therefore not unusual to only compile the shared library on
the execution platform and compile the analysis tool on the system where post-mortem
analysis is performed.

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
`liballtoallv_counts.so` library. This will generate two files: 
`send-counters.job<JOBID>.pid<PID>.txt` and `recv-counters.job<JOBID>.pid<PID>.txt`. 
Note that these files automatically include the jobid (if executed on a platform where
Slurm is not used, it is strongly advised to set the `SLURM_JOB_ID` environment variable
to specify a unique identifier). Using these two identifiers makes it easier to handle
multiple traces from multiple application and/or platforms.
- Gather timings: use the `liballtoallv_timings.so` shared library. This generates
a file: `timings.job<JOBID>.pid<PID>.md`. 
- Gather backtraces: use the `liballtoallv_backtrace.so` shared library. This generates
files `backtrace_call<ID>.md`, one per alltoallv call. *It is important to note that
the analysis tool assumes that these files are later manually moved to a `backtraces`
folder in your output directory.*

### Compilation

From the top directory of the repository source code, execute: `make library`.

This will create the `liballtoallv_counts.so`,  `liballtoallv_timings.so`,
`liballtoallv_backtrace.so` and`alltoallv/liballtoallv.so` shared libraries. For
most cases, the first 3 libraries are all users need.

### Execution

Before running the application to get a trace, users have the option to customize the
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

# Post-mortem analysis

We provide a set of tools that parses and analyses the data compiled when executing
the application with our shared library. These tools are implemented in Go and all of
them except one (`analyzebacktrace`) can be executed on a different platform.

## Installation

Most MPI applications are executed on a system where users cannot install system 
software, i.e., can be installed without privileges. Furthermore most systems do not
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
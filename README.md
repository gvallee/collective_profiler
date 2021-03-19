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
- Gather timings: use the `liballtoallv_exec_timings.so` and `liballtoallv_late_arrival.so` shared libraries. These generate
by default multiple files based on the following naming scheme:
 `<COLLECTIVE>_late_arrivals_timings.rank<RANK>_comm<COMMID>_job<JOBID>.md` and `<COLLECTIVE>_execution_times.rank<RANK>_comm<COMMID>_job<JOBID>.md`. 
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

#### Example with Slurm

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

### Generated data

When using the default shared libraries, the following files are generated which are details further below:
- files with a name starting with `send-counters` and `recv-counters`; files providing the alltoallv counts as defined by the MPI standard,
- files prefixed with `alltoallv_locations`, which stores data about the location of the ranks involved in alltoallv operations,
- files prefixed with `alltoallv_late_arrival`, which stores time data about ranks arrival into the alltoallv operations,
- files prefixed with `alltoallv_execution_times`, which stores the time each rank spent in the alltoallv operations,
- files prefixed with `alltoallv_backtrace`, which stores information about the context in which the application is invoking alltoallv.

In other to compress data and control the size of the generated dataset, the tool is able to use a compact notation to avoid duplication in lists. This notation is mainly applied to list of ranks. The format is a comma-separated list where consecutive numbers are saved as a range. For example, ranks `1, 3` means ranks 1 and 3; ranks `2-5` means ranks 2, 3, 4, and 5; and ranks `1, 3-5` means ranks 1, 3, 4, and 5. 

#### Send and receive count files

A `send-counters` and `recv-counters` files is generated per communicator used to perform an alltoallv operations. In other words, if alltoallv operations are executed on a single communicator, only two files are generated: `send-counters.job<JOBID>.rank<LEADRANK>.txt` and `recv-counters.job<JOBID>.rank<LEADRANK>.txt`, where `JOBID` is the job number when a job manager such as Slurm is used (equal to 0 when no job manager is used) and `LEADRANK` is the rank on `MPI_COMM_WORLD` that is rank 0 on the communicator used. `LEADRANK` is therefore used to differantiate data from different sub-communicators.

The content of the count files is predictable and organized as follow:
- `# Raw counters` indicates a new set of counts and is always followed by an empty line.
- `Number of ranks:` indicates how many ranks were involved in the alltoallv operations.
- `Datatype size:` indicates the size of the datatype used during the operation. Note that at the moment, the size is saved only in the context of the lead rank (as previously defined); alltoallv communications involving different datatype sizes is currently not supported.
- `Alltoallv calls:` indicates how many alltoallv calls *in total* (not specifically for the current set of counts) are captured in the file.
- `Count:` indicates how many alltoallv calls have the counts reported below. This line gives the total number of all calls as well as the list of all the calls using our compact notation.
- And finally the raw counts which are delimited by `BEGINNING DATA` and `END DATA`. Each line of the raw counts represents the count for ranks. Please refer to the MPI standard to fully understand the semantic of counts. `Rank(s) 0, 2: 1 2 3 4` means that ranks 0 and 2 have the following counts: 1 for rank 0, 2 for rank 1, 3 for rank 2 and 4 for rank 3.

#### Time file: alltoallv_late_arrival* and alltoallv_execution_times* files

The first line is the version of the data format. This is used for internal purposes to ensure that the post-mortem analysis tool supports that format. 

Then the file has a series of timing data per call. Each call data starts with `# Call` with the number of the call following by the ordered list of timing data per rank.

All timings are in seconds.

#### Location files

The first line is the version of the data format. This is used for internal purposes to ensure that the post-mortem analysis tool supports that format. 

Then the files has a series of entries, one per unique location where a location is the rank on the communicator and the host name. An example of such a location is:
```
Hostnames:
    Rank 0: node1
    Rank 1: node2
    Rank 2: node3
```
In order to control the size of the dataset, the metadata for each unique location includes: the communicator identifier (`Communicator ID:`), the list of calls having the unique location (`Calls:`), the ranks on MPI_COMM_WORLD (`COMM_WORLD_ rank:`) and PIDs (`PIDs`)

#### Trace files

The first line is the version of the data format. This is used for internal purposes to ensure that the post-mortem analysis tool supports that format. 

After the format version, a line prefixed with `stack trace for` indicates the binary associated to the trace. In most cases, only one binary will be reported.

Then the files has a series of entries, one per unique backtrace where a backtrace the data returned by the backtrace system call. An example of such a backtrace is:
```
/home/user1/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(_mpi_alltoallv+0xd4) [0x147c58511fa8]
/home/user1/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(MPI_Alltoallv+0x7d) [0x147c5851240c]
./wrf.exe_i202h270vx2() [0x32fec53]
./wrf.exe_i202h270vx2() [0x866604]
./wrf.exe_i202h270vx2() [0x1a5fd30]
./wrf.exe_i202h270vx2() [0x148ad35]
./wrf.exe_i202h270vx2() [0x5776ba]
./wrf.exe_i202h270vx2() [0x41b031]
./wrf.exe_i202h270vx2() [0x41afe6]
./wrf.exe_i202h270vx2() [0x41af62]
/lib64/libc.so.6(__libc_start_main+0xf3) [0x147c552d57b3]
./wrf.exe_i202h270vx2() [0x41ae69]
```

Finally each unique trace is accociated to 1 or more context(s) (`# Context` followed by the context number, i.e., the number in which it has been detected). A context is composed of a communicator (`Communicator`), the rank on the communicator (`Communicator rank`) which in most cases is `0` because the lead on the communicator, the rank on MPI_COMM_WORLD (`COMM_WORLD rank`), and finally the list of alltoallv calls having the backtrace using the compact notation previously presented.

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

### Generated data

The execution of the `profile` command to run the post-mortem analysis generates the following files:
- a single file prefixed with `profile_alltoallv` which gives an overview of some of the results from the post-mortem analysis,
- files refixed with `patterns-` that presents all the patterns that were detected (see the patterns sub-section for details), as well as a summary (file prefixed with `pattern-summary`),
- files prefixed with `stats` which provides statistics based on send and receive count files,
- files prefixed with `ranks_map` which provide a map of the ranks on each node,
- a file named `rankfile.txt` which gives the location of each rank,
- files prefixed with `alltoallv_heat_map` that have heat map for individual alltoallv calls, i.e., the amount of data that is send or received on a per rank basis. The send heat map file name is suffixed with `send.md`, while the receive heat map file name is suffixed with `recv.md`.
- files prefixed with `alltoallv_hosts_heat_map` that have a similar heat map but based on hosts rather than ranks.

#### Post-mortem overview files

All post-mortem analysis generates a single `profile_alltoallv*` file. The file is in markdown. The format of the file is as follow:
- A summary that gives the size of MPI_COMM_WORLD and the total number of alltoallv calls that the profiler tracked.
- A series of datasets where a dataset is a group of alltoallv calls having the same characteristics, including:
    - the communicator size used for the alltoallv operation
    - the number of alltoallv calls
    - how many send and receive counts were equal to zero (data used to know the sparcity of alltoallv calls).

#### Patterns

Two types of pattern files are generated

File with the `patterns-job<JOBID>-rank<LEADRANK>.md` naming scheme where `JOBID` and `LEADRANK` follow the definition previously presented. A pattern captures how many ranks are actually in communication with other ranks during a given alltoallv calls. This is valuable information when send and receive counts include counts equal to zero.
The file is organized as follow:
- first the pattern ID with the number of alltoallv calls that have the patterns. For instance `## Pattern #0 (61/484 alltoallv calls)` means that 61 alltoallv calls out of 484 have the pattern 0.
- the list of calls having that pattern using the compact notation previously presented,
- the pattern itself is a succession of entries that are either `X ranks send to Y other ranks` or `X ranks recv'ed from Y other ranks`.

File with the `patterns-job<JOBID>-rank<LEADRANK>.md` naming scheme `patterns-summary-job<JOBID>-rank<LEADRANK>.md` where `JOBID` and `LEADRANK` follow the definition previously presented. These files captures the patterns that have predefined characteristics, such as 1->n patterns (a few ranks send or receive to/from many ranks). These patterns are useful to detect alltoallv operations that do not involve all the ranks and therefore may create performance bottlenecks.

#### Statistics files

These files are based on the following format:
- the total number of alltoallv calls,
- the description of the datatypes that have been used during the alltoallv calls; for example, `484/484 calls use a datatype of size 4 while sending data` means that 484 out of a total of 484 alltoallv calls (so all the calls) used a send datatype of size 4,
- the communicator size,
- the message sizes that are calculated using the counts and the datatype size; the sizes are grouped based on a threshold (which can be customized). The distinction between messages is small messages (below the threshold), large messages and small and non zero messages.
- minimum and maximum count values.

#### Heat maps

A heat map is defined as the amount of data exchanged between ranks. Two different heat maps are generated: a rank-based heat map and a host-based heat map.

The rank based heat map (e.g., from file `alltoallv_heat-map.rank0-send.md`) is organized as follow:
- the first line is the version of the data format. This is used for internal purposes to ensure that the post-mortem analysis tool supports that format,
- then a list all numered calls with the amount of data each rank sends or receives.

The host heat map is very similar but the amount of data presented is on a host-based basis.

#### Rank maps

These files present the amount of data exchanged between ranks on a specific node. The files are named as followed: `ranks_map_<hostname>.txt`. Each line represents the amount of data a rank is sending, the line number being the rank on the communicator used to perform the alltoallv operation (line 0 for rank 0; line _n_ for rank _n_).

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
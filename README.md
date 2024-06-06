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
analysis is performed. To only compile the shared libraries only, execute `make libraries`;
to compile the post-mortem analysis tools, execute `make tool`.

## Documentation for the profiling of MPI applications

More documentation about the MPI collective profiler itself, the one being used while profiling your application is available in [src/README.md](src/README.md).

## Documentation about post-mortem analysis

We provide a set of tools that parses and analyses the data compiled when executing
the application with our shared library. These tools are implemented in Go and all of
them except one (`analyzebacktrace`) can be executed on a different platform.

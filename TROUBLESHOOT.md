Gathering traces and profiles is a very difficult task: it gathers a lot of data, 
potentially uses a lot of memory and potentially create huge files. As a result, it
is not unusual that things do not work right out of the box, depeneding on the scale,
complexity of the applications and compilers.
Fortunately, in case an application is not running properly for our shared libraries,
we provide a set of tests to help troubleshoot the problem.

# Compiler checks

The first check is to ensure that unit tests are passing with the compiler that needs
to be used with the application. For that, go to the top directory of the source code
and execute the following command: `make check`

If any errors are reported, it means the code is facing problem with the compiler(s)
that you are using, please open an issue on the repository.

# Run check

The second check is to make sure that the shared libraries are working properly while
using a simple benchmarks. We provide 2 simple examples, one in C and one in Fortran.
Users can select the version that matches the programming laguage used by their
application.

To prepare the tests, execute first the following command from the top directory of
the source code: `make examples`.

## Tests for C code

### Check the creation of count traces

From the top directory of the source code, execute the following command:
```
cd examples ; LD_PRELOAD=../alltoallv/liballtoallv_counts.so mpirun -np 3 ./alltoallv_c
```

If successful, 3 files should have been created: `recv-counters.rank<ID>.txt`, 
`send-counters.rank<ID>.txt`, and `profile_alltoallv.rank<ID>.md`.

If one of these files is missing, please open an issue on our Github project page.
If the files were successfully created, please check their content:
- `profile_alltoallv.rank<ID>.md` should have the following first 3 lines:
```
# Summary                                                                                                               Total number of alltoallv calls = 1 (limit is -1; -1 means no limit)                                                    Alltoallv call range: [0-0] 
```
- `send-counters.rank<ID>.txt` should have the following content:
```
# Raw counters

Number of ranks: 3
Datatype size: 8
Alltoallv calls 0-0
Count: 1 calls - 0 


BEGINNING DATA
Rank(s) 0-2: 0 1 2
END DATA
```
- `recv-counters.rank<ID>.txt` should have the following content:
```
# Raw counters

Number of ranks: 3
Datatype size: 8
Alltoallv calls 0-0
Count: 1 calls - 0 


BEGINNING DATA
Rank(s) 0: 0 0 0
Rank(s) 1: 1 1 1
Rank(s) 2: 2 2 2
END DATA
```

### Check the creation of timing traces

From the top directory of the source code, execute the following command:
```
cd examples ; LD_PRELOAD=../alltoallv/liballtoallv_timings.so mpirun -np 3 ./alltoallv_c
```

A single `timings.rank<ID>.md` file should have been created with a content similar to:
```
Alltoallv call #0
# Late arrival timings
Rank 0: 0.000003
Rank 1: 0.000014
Rank 2: 0.000011
# Execution times of Alltoallv function 
Rank 0: 0.000022
Rank 1: 0.000023
Rank 2: 0.000023
```

### Check the creation of backtraces

From the top directory of the source code, execute the following command:
```
cd examples ; LD_PRELOAD=../alltoallv/liballtoallv_backtrace.so mpirun -np 3 ./alltoallv_c
```

A `backtrace_rank0_call0.md` file should have been created with a content similar to:
```
stack trace for /home/toto/alltoall_profiling/examples/alltoallv_c pid=5880
../alltoallv/liballtoallv_backtrace.so(_mpi_alltoallv+0x95) [0x7f5e5eff5ec3]
../alltoallv/liballtoallv_backtrace.so(MPI_Alltoallv+0x53) [0x7f5e5eff6185]
./alltoallv_c(+0xd81) [0x7f5e5f600d81] 
/lib/x86_64-linux-gnu/libc.so.6(__libc_start_main+0xe7) [0x7f5e5e8d1b97]
./alltoallv_c(+0x9aa) [0x7f5e5f6009aa] 
```

## Tests for Fortran code

### Check the creation of count traces

From the top directory of the source code, execute the following command:
```
cd examples ; LD_PRELOAD=../alltoallv/liballtoallv_counts.so mpirun -np 3 ./alltoallv_f
```

If successful, 3 files should have been created: `recv-counters.rank<ID>.txt`, 
`send-counters.rank<ID>.txt`, and `profile_alltoallv.rank<ID>.md`.

If one of these files is missing, please open an issue on our Github project page.
If the files were successfully created, please check their content:
- `profile_alltoallv.rank<ID>.md` should have the following first 3 lines:
```
# Summary
Total number of alltoallv calls = 2 (limit is -1; -1 means no limit)
Alltoallv call range: [0-1] 
```
- `send-counters.rank<ID>.txt` should have the following content:
```
# Raw counters

Number of ranks: 3
Datatype size: 8
Alltoallv calls 0-1
Count: 1 calls - 0-1 


BEGINNING DATA
Rank(s) 0: 1 2 0
Rank(s) 1: 0 0 3
Rank(s) 2: 1 0 0 
END DATA
```
- `recv-counters.rank<ID>.txt` should have the following content:
```
# Raw counters

Number of ranks: 3
Datatype size: 8
Alltoallv calls 0-1
Count: 1 calls - 0-1 


BEGINNING DATA
Rank(s) 0: 1 0 1
Rank(s) 1: 2 0 0
Rank(s) 2: 0 3 0
END DATA
```

### Check the creation of timing traces

From the top directory of the source code, execute the following command:
```
cd examples ; LD_PRELOAD=../alltoallv/liballtoallv_timings.so mpirun -np 3 ./alltoallv_f
```

A single `timings.rank<ID>.md` file should have been created with a content similar to:
```
Alltoallv call #0
# Late arrival timings
Rank 0: 0.000013
Rank 1: 0.000017
Rank 2: 0.000013
# Execution times of Alltoallv function 
Rank 0: 0.000023
Rank 1: 0.000031
Rank 2: 0.000023
Alltoallv call #1
# Late arrival timings
Rank 0: 0.000002
Rank 1: 0.000111
Rank 2: 0.000110
# Execution times of Alltoallv function 
Rank 0: 0.000002
Rank 1: 0.000002
Rank 2: 0.000002
```

### Check the creation of backtraces

From the top directory of the source code, execute the following command:
```
cd examples ; LD_PRELOAD=../alltoallv/liballtoallv_backtrace.so mpirun -np 3 ./alltoallv_f
```

A `backtraces` directory and a `backtrace_rank0_call0.md` file within the directory should have been created with a content similar to:
```
stack trace for /home/toto/alltoall_profiling/examples/alltoallv_f pid=6874
../alltoallv/liballtoallv_backtrace.so(_mpi_alltoallv+0x95) [0x7ffd7d5f5f62] 
../alltoallv/liballtoallv_backtrace.so(MPI_Alltoallv+0x53) [0x7ffd7d5f6224]
../alltoallv/liballtoallv_backtrace.so(mpi_alltoallv_+0xda) [0x7ffd7d5f6304]
./alltoallv_f(+0x1d43) [0x7ffd7dc01d43]
./alltoallv_f(+0x2047) [0x7ffd7dc02047]
/lib/x86_64-linux-gnu/libc.so.6(__libc_start_main+0xe7) [0x7ffd7cbd1b97]
./alltoallv_f(+0xdea) [0x7ffd7dc00dea] 
```
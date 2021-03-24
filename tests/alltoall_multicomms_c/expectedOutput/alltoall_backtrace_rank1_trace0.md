FORMAT_VERSION: 9

stack trace for /global/scratch/users/cyrusl/placement/expt0070/alltoall_profiling/examples/alltoall_multicomms_c pid=1101219

# Trace

/global/home/users/cyrusl/placement/expt0070/alltoall_profiling/src/alltoall/liballtoall_backtrace_counts_unequal.so(_mpi_alltoall+0x9c) [0x14dcab9cd900]
/global/home/users/cyrusl/placement/expt0070/alltoall_profiling/src/alltoall/liballtoall_backtrace_counts_unequal.so(MPI_Alltoall+0x4a) [0x14dcab9cdbb1]
/global/home/users/cyrusl/placement/expt0070/alltoall_profiling/examples/alltoall_multicomms_c() [0x401694]
/global/home/users/cyrusl/placement/expt0070/alltoall_profiling/examples/alltoall_multicomms_c() [0x402087]
/lib64/libc.so.6(__libc_start_main+0xf3) [0x14dcab0fe7b3]
/global/home/users/cyrusl/placement/expt0070/alltoall_profiling/examples/alltoall_multicomms_c() [0x400c1e]

# Context 0

Communicator: 0
Communicator rank: 0
COMM_WORLD rank: 1
Calls: 1


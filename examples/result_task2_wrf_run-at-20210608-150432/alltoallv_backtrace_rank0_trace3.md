FORMAT_VERSION: 9

stack trace for /gpfs/fs1/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/WRF/WRF_MPI4.0.1/main/wrf.exe pid=29695

# Trace

/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(_mpi_alltoallv+0xd4) [0x2ae1d7064bc4]
/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(MPI_Alltoallv+0x7d) [0x2ae1d7065028]
./wrf.exe() [0x3314863]
./wrf.exe() [0x863704]
./wrf.exe() [0x1a59e40]
./wrf.exe() [0x14880e5]
./wrf.exe() [0x5747ba]
./wrf.exe() [0x574f36]
./wrf.exe() [0x418131]
./wrf.exe() [0x4180d9]
./wrf.exe() [0x418052]
/lib64/libc.so.6(__libc_start_main+0xf5) [0x2ae1d9ba9555]
./wrf.exe() [0x417f69]

# Context 0

Communicator: 1
Communicator rank: 0
COMM_WORLD rank: 0
Calls: 3


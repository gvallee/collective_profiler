FORMAT_VERSION: 9

stack trace for /gpfs/fs1/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/WRF/WRF_MPI4.0.1/main/wrf.exe pid=29695

# Trace

/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(_mpi_alltoallv+0xd4) [0x2ae1d7064bc4]
/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(MPI_Alltoallv+0x7d) [0x2ae1d7065028]
./wrf.exe() [0x3314863]
./wrf.exe() [0x863704]
./wrf.exe() [0x1a602c6]
./wrf.exe() [0x14832fd]
./wrf.exe() [0x574b7e]
./wrf.exe() [0x418131]
./wrf.exe() [0x4180d9]
./wrf.exe() [0x418052]
/lib64/libc.so.6(__libc_start_main+0xf5) [0x2ae1d9ba9555]
./wrf.exe() [0x417f69]

# Context 0

Communicator: 0
Communicator rank: 0
COMM_WORLD rank: 0
Calls: 2, 12, 20, 28, 36, 44, 52, 60, 68, 76, 84, 92, 100, 108, 116, 124, 132, 140, 148, 156, 164, 172, 180, 188, 196, 204, 212, 220, 228, 236, 244, 252, 260, 268, 276, 284, 292, 300, 308, 316, 324, 332, 340, 348, 356, 364, 372, 380, 388, 396, 404, 412, 420, 428, 436, 444, 452, 460, 468, 476, 484, 492, 500, 508, 516, 524, 532, 540, 548, 556, 564, 572, 580, 588, 596, 604, 612, 620, 628, 636, 644, 652, 660, 668, 676, 684, 692, 700, 708, 716, 724, 732, 740, 748, 756, 764, 772, 780, 788, 796, 804, 812, 820, 828, 836, 844, 852, 860, 868, 876, 884, 892, 900, 908, 916, 924, 932, 940, 948, 956


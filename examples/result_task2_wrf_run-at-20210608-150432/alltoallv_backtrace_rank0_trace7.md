FORMAT_VERSION: 9

stack trace for /gpfs/fs1/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/WRF/WRF_MPI4.0.1/main/wrf.exe pid=29695

# Trace

/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(_mpi_alltoallv+0xd4) [0x2ae1d7064bc4]
/home/l/lcl_uotiscscc/lcl_uotiscsccs1034/scratch/code-challenge/collective_profiler/src/alltoallv/liballtoallv_backtrace.so(MPI_Alltoallv+0x7d) [0x2ae1d7065028]
./wrf.exe() [0x3314b8b]
./wrf.exe() [0x99e106]
./wrf.exe() [0x1a63eab]
./wrf.exe() [0x575212]
./wrf.exe() [0x418131]
./wrf.exe() [0x4180d9]
./wrf.exe() [0x418052]
/lib64/libc.so.6(__libc_start_main+0xf5) [0x2ae1d9ba9555]
./wrf.exe() [0x417f69]

# Context 0

Communicator: 0
Communicator rank: 0
COMM_WORLD rank: 0
Calls: 11, 19, 27, 35, 43, 51, 59, 67, 75, 83, 91, 99, 107, 115, 123, 131, 139, 147, 155, 163, 171, 179, 187, 195, 203, 211, 219, 227, 235, 243, 251, 259, 267, 275, 283, 291, 299, 307, 315, 323, 331, 339, 347, 355, 363, 371, 379, 387, 395, 403, 411, 419, 427, 435, 443, 451, 459, 467, 475, 483, 491, 499, 507, 515, 523, 531, 539, 547, 555, 563, 571, 579, 587, 595, 603, 611, 619, 627, 635, 643, 651, 659, 667, 675, 683, 691, 699, 707, 715, 723, 731, 739, 747, 755, 763, 771, 779, 787, 795, 803, 811, 819, 827, 835, 843, 851, 859, 867, 875, 883, 891, 899, 907, 915, 923, 931, 939, 947, 955


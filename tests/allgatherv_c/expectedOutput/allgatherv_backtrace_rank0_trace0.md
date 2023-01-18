FORMAT_VERSION: 9

stack trace for /home/gvallee/Projects/collective_profiler/examples/allgatherv_c pid=13226

# Trace

../src/allgatherv/liballgatherv_backtrace.so(_mpi_allgatherv+0xea) [0x7fd54d23b855]
../src/allgatherv/liballgatherv_backtrace.so(MPI_Allgatherv+0x4e) [0x7fd54d23bf24]
./allgatherv_c(+0x13f4) [0x55a01fdf23f4]
./allgatherv_c(+0x1566) [0x55a01fdf2566]
/lib/x86_64-linux-gnu/libc.so.6(+0x29d90) [0x7fd54cf09d90]
/lib/x86_64-linux-gnu/libc.so.6(__libc_start_main+0x80) [0x7fd54cf09e40]
./allgatherv_c(+0x1185) [0x55a01fdf2185]

# Context 0

Communicator: 0
Communicator rank: 0
COMM_WORLD rank: 0
Calls: 0-1


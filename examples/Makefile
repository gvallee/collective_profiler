#
# Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

all: alltoallv_c alltoallv_dt_c alltoallv_multicomms_c alltoallv_f alltoallv_bigcounts_c alltoall_demo alltoall_simple_c alltoall_bigcounts_c alltoall_multicomms_c alltoall_dt_c task1_c

alltoall_demo: alltoall_demo.c
	mpicc -g alltoall_demo.c -o alltoall_demo

alltoall_simple_c: alltoall_simple_c.c
	mpicc -g alltoall_simple_c.c -o alltoall_simple_c
  
alltoall_bigcounts_c: alltoall_bigcounts_c.c
	mpicc -g alltoall_bigcounts_c.c -o alltoall_bigcounts_c

alltoall_multicomms_c: alltoall_multicomms_c.c
	mpicc -g alltoall_multicomms_c.c -o alltoall_multicomms_c

alltoall_dt_c: alltoall_dt_c.c
	mpicc -g alltoall_dt_c.c -o alltoall_dt_c

alltoallv_c: alltoallv.c
	mpicc -g alltoallv.c -o alltoallv_c

alltoallv_dt_c: alltoallv_dt.c
	mpicc -g alltoallv_dt.c -o alltoallv_dt_c

alltoallv_multicomms_c: alltoallv_multicomms.c
	mpicc -g alltoallv_multicomms.c -o alltoallv_multicomms_c

alltoallv_f: alltoallv.f90
	mpifort -g alltoallv.f90 -o alltoallv_f

alltoallv_bigcounts_c: alltoallv_bigcounts.c
	mpicc -g alltoallv_bigcounts.c -o alltoallv_bigcounts_c

task1_c: task1.c
	mpicc -g task1.c -o task1_c

test: task1_c
	mpirun -np 4 --oversubscribe task1_c

clean:
	@rm -f alltoallv_c alltoallv_dt_c alltoallv_f alltoallv_multicomms_c alltoallv_bigcounts_c alltoall_demo alltoall_simple_c task1_c alltoall_c

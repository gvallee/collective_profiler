#
# Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

all: alltoallv_c                 \
	alltoallv_dt_c               \
	alltoallv_multicomms_c       \
	alltoallv_f                  \
	alltoallv_bigcounts_c        \
	alltoallv_multi_backtraces_c \
	alltoall_demo                \
	alltoall_simple_c            \
	alltoall_bigcounts_c         \
	alltoall_multicomms_c        \
	alltoall_dt_c				 \
	allgatherv_c

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

alltoallv_multi_backtraces_c: alltoallv_multi_backtraces.c collective_profiler_example_utils.h
	mpicc -g alltoallv_multi_backtraces.c -o alltoallv_multi_backtraces_c

allgatherv_c: allgatherv.c
	mpicc -g allgatherv.c -o allgatherv_c

clean:
	@rm -f alltoall_demo
	@rm -f alltoall_simple_c
	@rm -f alltoall_bigcounts_c
	@rm -f alltoall_multicomms_c
	@rm -f alltoall_dt_c
	@rm -f alltoallv_c
	@rm -f alltoallv_dt_c
	@rm -f alltoallv_multicomms_c
	@rm -f alltoallv_f
	@rm -f alltoallv_bigcounts_c
	@rm -f alltoallv_multi_backtraces_c
	@rm -f allgatherv_c
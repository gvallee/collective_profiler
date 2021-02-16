#
# Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

all: alltoallv_c alltoallv_dt_c alltoallv_multicomms_c alltoallv_f alltoallv_bigcounts_c alltoall

alltoall: alltoall.c
	mpicc -g alltoall.c -o alltoall
  
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

clean:
	@rm -f alltoallv_c alltoallv_dt_c alltoallv_f alltoallv_multicomms_c alltoallv_bigcounts_c alltoall

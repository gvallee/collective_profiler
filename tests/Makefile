#
# Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

all: liballtoallv_dummy.so wrapper_test/wrapper_test

liballtoallv_dummy.so: dummy_pmpi.c
	mpicc -g -shared -fPIC dummy_pmpi.c -o liballtoallv_dummy.so

wrapper_test/wrapper_test: wrapper_test/wrapper_test.c
	mpicc wrapper_test/wrapper_test.c -o wrapper_test/wrapper_test

clean:
	@rm -f *.so *.o
	@rm -f wrapper_test/wrapper_test
#
# Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

all: lib_pmpi_late_arrival.so

lib_pmpi_late_arrival.so: pmpi_late_arrival.c
	mpicc -g -shared -fPIC pmpi_late_arrival.c -o lib_pmpi_late_arrival.so

clean:
	@rm -f lib_pmpi_late_arrival.so
#
# Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

all: common alltoallv alltoall

.PHONY: alltoall alltoallv common

common:
	cd common && make

alltoallv: common
	cd alltoallv && make

alltoall: common
	cd alltoall && make

check:
	cd alltoall && make check
	cd alltoallv && make check

clean:
	cd alltoall && make clean
	cd alltoallv && make clean
	cd common && make clean
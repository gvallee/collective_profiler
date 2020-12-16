#
# Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

all: alltoallv libraries examples tools tests doc

.PHONY: libraries alltoallv examples tools check tests

alltoallv:
	cd src && make alltoallv

libraries: 
	cd src && make

examples: libraries
	cd examples && make

GOCMD := $(shell command -v go 2>/dev/null)
ifndef GOCMD
tools:
	@echo "Go not installed; skipping tools' compilation"
else
tools:
	cd tools && make;
endif

check: libraries
	cd src && make check
	cd tools && make check

tests:
	cd tests && make

doc:
	cd doc && make

clean:
	cd examples && make clean
	cd src && make clean
	cd tools && make clean
	cd tests && make clean
	cd doc && make clean

validate: tools tests
	cd tools/cmd/validate; ./validate -profiler

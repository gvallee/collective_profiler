#
# Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

all: libraries examples tools tests doc

.PHONY: libraries alltoallv alltoall examples tools check tests check_gnuplot

alltoallv:
	cd src && make alltoallv

alltoall:
	cd src && make alltoall

libraries: 
	cd src && make

examples: libraries
	cd examples && make

GNUPLOTCMD := $(shell command -v gnuplot 2>/dev/null)
ifndef GNUPLOTCMD
check_gnuplot:
	@echo "gnuplot is not installed; please install"
	@exit 1
else
check_gnuplot:
	@echo "gnuplot available: ${GNUPLOTCMD}"
endif

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

validate: clean check_gnuplot all
	# postmortem validates both the profiler's capabilities and postmortem analysis
	cd tools/cmd/validate; ./validate -postmortem

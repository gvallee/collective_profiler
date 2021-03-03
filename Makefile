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
	# We overwite CC because we discovered issues when CC=icc and in our context,
	# the tools can always be compiled with gcc.
	cd tools && make CC=gcc;
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

validate: clean check_gnuplot all check
	# webui validates the profiler's capabilities, postmortem analysis as well as the webui
	cd tools/cmd/validate; ./validate -webui

install-go:
ifndef GOCMD
	@echo "Installing Go 1.13 for Linux into your home directory..."
	`cd ${HOME}; wget https://golang.org/dl/go1.13.15.linux-amd64.tar.gz && tar xzf go1.13.15.linux-amd64.tar.gz`
	@echo 'Please add the following to your .bashrc:''
	@echo 'export GOPATH=$$HOME/go'
	@echo 'export PATH=$$GOPATH/bin:$$PATH'
	@echo 'export LD_LIBRARY_PATH=$$GOPATH/lib:$$LD_LIBRARY_PATH'
else
	@echo "Go already installed"
endif

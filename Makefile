all: alltoallv examples tools tests doc

.PHONY: alltoallv examples tools check tests

alltoallv:
	cd alltoallv && make

examples: alltoallv
	cd examples && make

GOCMD := $(shell command -v go 2>/dev/null)
ifndef GOCMD
tools:
	@echo "Go not installed; skipping tools' compilation"
else
tools:
	cd tools && make;
endif

check: alltoallv
	cd alltoallv && make check
	cd tools && make check

tests:
	cd tests && make

doc:
	cd doc && make

clean:
	cd examples && make clean
	cd alltoallv && make clean
	cd tools && make clean
	cd tests && make clean
	cd doc && make clean

validate: tools tests
	cd tools/cmd/validate; ./validate -profiler

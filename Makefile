all: alltoallv examples tools tests

.PHONY: alltoallv examples tools check tests

alltoallv:
	cd alltoallv && make

examples: alltoallv
	cd examples && make
	
tools:
	cd tools && make


check: alltoallv
	cd alltoallv && make check

tests:
	cd tests && make

clean:
	cd examples && make clean
	cd alltoallv && make clean
	cd tools && make clean
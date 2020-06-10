all: alltoallv examples tools tests doc

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

doc:
	cd doc && make

clean:
	cd examples && make clean
	cd alltoallv && make clean
	cd tools && make clean
	cd tests && make clean
	cd doc && make clean
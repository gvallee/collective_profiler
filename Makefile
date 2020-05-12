all:
	cd examples && make
	cd alltoallv && make

check:
	cd alltoallv && make check

clean:
	cd examples && make clean
	cd examples && make clean
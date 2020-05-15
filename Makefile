all:
	cd examples && make
	cd alltoallv && make
	cd tools && make

check:
	cd alltoallv && make check

clean:
	cd examples && make clean
	cd alltoallv && make clean
	cd tools && make clean
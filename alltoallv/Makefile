all: mpi_alltoallv.c
	mpicc -g -shared -fPIC mpi_alltoallv.c -o liballtoallv.so

clean:
	@rm -f liballtoallv.so

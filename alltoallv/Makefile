all: liballtoallv.so grouping_test

grouping.o: grouping.c grouping.h
	$(CC) -c grouping.c 

grouping_test: grouping.o grouping_test.c
	$(CC) grouping.o grouping_test.c -o grouping_test

liballtoallv.so: grouping.o mpi_alltoallv.c
	mpicc -g -shared -fPIC mpi_alltoallv.c -o liballtoallv.so

clean:
	@rm -f liballtoallv.so grouping.o grouping_test

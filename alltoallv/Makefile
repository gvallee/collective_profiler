all: liballtoallv.so grouping_test

logger.o: logger.c logger.h
	$(CC) -fPIC -c logger.c

grouping.o: grouping.c grouping.h
	$(CC) -fPIC -c grouping.c 

grouping_test: grouping.o grouping_test.c
	$(CC) -fPIC grouping.o grouping_test.c -o grouping_test

liballtoallv.so: logger.o grouping.o mpi_alltoallv.c
	mpicc -g -shared -fPIC logger.o grouping.o mpi_alltoallv.c -o liballtoallv.so

check: check_grouping

check_grouping: grouping_test
	./grouping_test

clean:
	@rm -f *.so grouping_test *.o

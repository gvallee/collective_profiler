all: liballtoallv.so grouping_test compress_array_test

logger.o: logger.c logger.h alltoallv_profiler.h
	$(CC) -fPIC -c logger.c

grouping.o: grouping.c grouping.h alltoallv_profiler.h
	$(CC) -fPIC -c grouping.c 

grouping_test: grouping.o grouping_test.c alltoallv_profiler.h
	$(CC) -fPIC grouping.o grouping_test.c -o grouping_test

compress_array_test: logger.o compress_array_test.c
	$(CC) -fPIC logger.o compress_array_test.c -o compress_array_test

liballtoallv.so: logger.o grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -g -shared -fPIC logger.o grouping.o mpi_alltoallv.c -o liballtoallv.so

check_compress_array: compress_array_test
	./compress_array_test

check_grouping: grouping_test
	./grouping_test

check: all check_grouping check_compress_array

clean:
	@rm -f *.so grouping_test *.o

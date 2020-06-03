all: liballtoallv.so liballtoallv_counts.so liballtoallv_timings.so liballtoallv_backtrace.so grouping_test compress_array_test patterns_detection_test

logger.o: logger.c logger.h alltoallv_profiler.h
	$(CC) -fPIC -c logger.c

logger_counts.o: logger.c logger.h alltoallv_profiler.h
	$(CC) -fPIC -DENABLE_RAW_DATA=1 -c logger.c -o logger_counts.o

logger_timings.o: logger.c logger.h alltoallv_profiler.h
	$(CC) -fPIC -DENABLE_TIMING=1 -c logger.c -o logger_timings.o

logger_backtrace.o: logger.c logger.h alltoallv_profiler.h
	$(CC) -fPIC -DENABLE_BACKTRACE=1 -c logger.c -o logger_backtrace.o

pattern.o: pattern.c pattern.h
	$(CC) -fPIC -c pattern.c

grouping.o: grouping.c grouping.h alltoallv_profiler.h
	$(CC) -fPIC -c grouping.c 

grouping_test: grouping.o grouping_test.c alltoallv_profiler.h
	$(CC) -fPIC grouping.o grouping_test.c -o grouping_test

compress_array_test: logger.o compress_array_test.c
	$(CC) -fPIC logger.o compress_array_test.c -o compress_array_test

patterns_detection_test: pattern.o patterns_detection_test.c
	$(CC) -fPIC pattern.o patterns_detection_test.c -o patterns_detection_test

liballtoallv_counts.so: logger_counts.o grouping.o pattern.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -g -shared -fPIC -DENABLE_RAW_DATA=1 logger_counts.o grouping.o mpi_alltoallv.c -o liballtoallv_counts.so

liballtoallv_timings.so: logger_timings.o grouping.o pattern.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -g -shared -fPIC -DENABLE_TIMING=1 logger.o grouping.o mpi_alltoallv.c -o liballtoallv_timings.so

liballtoallv_backtrace.so: logger_backtrace.o grouping.o pattern.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -g -shared -fPIC -DENABLE_BACKTRACE=1 logger_backtrace.o grouping.o mpi_alltoallv.c -o liballtoallv_backtrace.so

liballtoallv.so: logger.o grouping.o pattern.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -g -shared -fPIC logger.o grouping.o mpi_alltoallv.c -o liballtoallv.so

check_patterns_detection: patterns_detection_test
	./patterns_detection_test

check_compress_array: compress_array_test
	./compress_array_test

check_grouping: grouping_test
	./grouping_test

check: all check_grouping check_compress_array check_patterns_detection

clean:
	@rm -f *.so *.o
	@rm -f grouping_test compress_array_test patterns_detection_test 
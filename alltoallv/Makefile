#
# Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

all: liballtoallv.so liballtoallv_location.so liballtoallv_counts.so liballtoallv_late_arrival.so liballtoallv_a2a_timings.so liballtoallv_backtrace.so

logger_counts.o: ../common/logger.c ../common/logger.h alltoallv_profiler.h
	$(CC) -I../ -I../common/ -I../ -fPIC -DENABLE_RAW_DATA=1 -c ../common/logger.c -o logger_counts.o

logger_a2a_timings.o: ../common/logger.c ../common/logger.h alltoallv_profiler.h
	$(CC) -I../ -I../common/ -fPIC -DENABLE_A2A_TIMING=1 -c ../common/logger.c -o logger_a2a_timings.o

logger_late_arrival_timings.o: ../common/logger.c ../common/logger.h alltoallv_profiler.h
	$(CC) -I../ -I../common/ -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 -c ../common/logger.c -o logger_late_arrival_timings.o

logger_backtrace.o: ../common/logger.c ../common/logger.h alltoallv_profiler.h
	$(CC) -I../ -I../common/ -fPIC -DENABLE_BACKTRACE=1 -c ../common/logger.c -o logger_backtrace.o

logger_location.o: ../common/logger.c ../common/logger.h alltoallv_profiler.h
	$(CC) -I../ -I../common/ -fPIC -DENABLE_LOCATION_TRACKING=1 -c ../common/logger.c -o logger_location.o

liballtoallv_counts.so: logger_counts.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_RAW_DATA=1 logger_counts.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_counts.so

liballtoallv_a2a_timings.so: logger_a2a_timings.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_A2A_TIMING=1 logger_a2a_timings.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_a2a_timings.so

liballtoallv_late_arrival.so: logger_late_arrival_timings.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 logger_late_arrival_timings.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_late_arrival.so

liballtoallv_backtrace.so: logger_backtrace.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_BACKTRACE=1 logger_backtrace.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_backtrace.so

liballtoallv_location.so: logger_location.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LOCATION_TRACKING=1 logger_location.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_location.so

liballtoallv.so: ../common/logger.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC ../common/logger.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv.so

check: all

clean:
	@rm -f *.so *.o
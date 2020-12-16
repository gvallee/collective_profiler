#
# Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#
# ******************************************************************************************************
# * Copyright (c) 2020, University College London and Mellanox Technolgies Limited. All rights reserved.
# * - for further contributions 
# ******************************************************************************************************/

all: liballtoall.so liballtoall_location.so liballtoall_counts.so liballtoall_late_arrival.so liballtoall_a2a_timings.so liballtoall_backtrace.so

logger.o: logger.c logger.h alltoall_profiler.h
	$(CC) -I../ -I../common -fPIC -c logger.c

logger_counts.o: logger.c logger.h alltoall_profiler.h
	$(CC) -I../ -I../common -fPIC -DENABLE_RAW_DATA=1 -c logger.c -o logger_counts.o

logger_a2a_timings.o: logger.c logger.h alltoall_profiler.h
	$(CC) -I../ -I../common -fPIC -DENABLE_A2A_TIMING=1 -c logger.c -o logger_a2a_timings.o

logger_late_arrival_timings.o: logger.c logger.h alltoall_profiler.h
	$(CC) -I../ -I../common -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 -c logger.c -o logger_late_arrival_timings.o

logger_backtrace.o: logger.c logger.h alltoall_profiler.h
	$(CC) -I../ -I../common -fPIC -DENABLE_BACKTRACE=1 -c logger.c -o logger_backtrace.o

logger_location.o: logger.c logger.h alltoall_profiler.h
	$(CC) -I../ -I../common -fPIC -DENABLE_LOCATION_TRACKING=1 -c logger.c -o logger_location.o

liballtoall_counts.so: logger_counts.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_A2A_TIMING=1 logger_counts.o ../common/grouping.o mpi_alltoall.c -o liballtoall_counts.so

liballtoall_a2a_timings.so: logger_a2a_timings.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_A2A_TIMING=1 logger_a2a_timings.o ../common/grouping.o mpi_alltoall.c -o liballtoall_a2a_timings.so

liballtoall_late_arrival.so: logger_late_arrival_timings.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 logger_late_arrival_timings.o ../common/grouping.o mpi_alltoall.c -o liballtoall_late_arrival.so

liballtoall_backtrace.so: logger_backtrace.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_BACKTRACE=1 logger_backtrace.o ../common/grouping.o mpi_alltoall.c -o liballtoall_backtrace.so

liballtoall_location.so: logger_location.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LOCATION_TRACKING=1 logger_location.o ../common/grouping.o mpi_alltoall.c -o liballtoall_location.so

liballtoall.so: ../common/logger.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC ../common/logger.o ../common/grouping.o mpi_alltoall.c -o liballtoall.so

check: all 

clean:
	@rm -f *.so *.o
	@rm -f grouping_test compress_array_test patterns_detection_test 
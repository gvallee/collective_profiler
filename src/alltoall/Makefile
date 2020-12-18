#
# Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#
# ******************************************************************************************************
# * Copyright (c) 2020, University College London and Mellanox Technolgies Limited. All rights reserved.
# * - for further contributions 
# ******************************************************************************************************/

all: liballtoall.so liballtoall_location.so liballtoall_counts.so liballtoall_late_arrival.so liballtoall_a2a_timings.so liballtoall_backtrace.so

liballtoall_counts.so: ../common/logger_counts.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_A2A_TIMING=1 ../common/logger_counts.o ../common/grouping.o mpi_alltoall.c -o liballtoall_counts.so

liballtoall_a2a_timings.so: ../common/logger_exec_timings.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_A2A_TIMING=1 ../common/logger_exec_timings.o ../common/grouping.o mpi_alltoall.c -o liballtoall_exec_timings.so

liballtoall_late_arrival.so: ../common/logger_late_arrival_timings.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 ../common/logger_late_arrival_timings.o ../common/grouping.o mpi_alltoall.c -o liballtoall_late_arrival.so

liballtoall_backtrace.so: ../common/logger_backtrace.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_BACKTRACE=1 ../common/logger_backtrace.o ../common/grouping.o mpi_alltoall.c -o liballtoall_backtrace.so

liballtoall_location.so: ../common/logger_location.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LOCATION_TRACKING=1 ../common/logger_location.o ../common/grouping.o mpi_alltoall.c -o liballtoall_location.so

liballtoall.so: ../common/logger.o ../common/grouping.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC ../common/logger.o ../common/grouping.o mpi_alltoall.c -o liballtoall.so

check: all 

clean:
	@rm -f *.so *.o
	@rm -f grouping_test compress_array_test patterns_detection_test 
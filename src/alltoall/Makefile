#
# Copyright (c) 2020-2022, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#
# ******************************************************************************************************
# * Copyright (c) 2020, University College London and Mellanox Technolgies Limited. All rights reserved.
# * - for further contributions 
# ******************************************************************************************************/

include ../makefile_common.mk

all: liballtoall.so liballtoall_location.so liballtoall_counts.so liballtoall_late_arrival.so liballtoall_exec_timings.so liballtoall_backtrace.so

liballtoall_counts.so: ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_unequal.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_COMPACT_FORMAT=1 -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_compact.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_COMPACT_FORMAT=1 -DENABLE_RAW_DATA=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_unequal_compact.so

liballtoall_exec_timings.so: ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_EXEC_TIMING=1 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_exec_timings.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_EXEC_TIMING=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_exec_timings_counts_unequal.so

liballtoall_late_arrival.so: ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_late_arrival.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_late_arrival_counts_unequal.so

liballtoall_backtrace.so: ${COMMON_OBJECTS} ../common/logger_backtrace.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_BACKTRACE=1 ${COMMON_OBJECTS} ../common/logger_backtrace.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_backtrace.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_BACKTRACE=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/logger_backtrace.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_backtrace_counts_unequal.so

liballtoall_location.so: ${COMMON_OBJECTS} ../common/logger_location.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LOCATION_TRACKING=1 ${COMMON_OBJECTS} ../common/logger_location.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_location.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LOCATION_TRACKING=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/logger_location.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_location_counts_unequal.so

liballtoall.so: ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoall.c -o liballtoall.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_unequal.so

check: all 

clean:
	@rm -f *.so *.o
	@rm -f grouping_test compress_array_test patterns_detection_test 
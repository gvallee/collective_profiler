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

liballtoall_counts.so: check-env ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_unequal.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_COMPACT_FORMAT=1 -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_compact.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_COMPACT_FORMAT=1 -DENABLE_RAW_DATA=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_unequal_compact.so $(LDFLAGS)

liballtoall_exec_timings.so: check-env ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_EXEC_TIMING=1 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_exec_timings.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_EXEC_TIMING=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_exec_timings_counts_unequal.so $(LDFLAGS)

liballtoall_late_arrival.so: check-env ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_LATE_ARRIVAL_TIMING=1 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_late_arrival.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_LATE_ARRIVAL_TIMING=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_late_arrival_counts_unequal.so $(LDFLAGS)

liballtoall_backtrace.so: check-env ${COMMON_OBJECTS} ../common/logger_backtrace.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_BACKTRACE=1 ${COMMON_OBJECTS} ../common/logger_backtrace.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_backtrace.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_BACKTRACE=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/logger_backtrace.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_backtrace_counts_unequal.so $(LDFLAGS)

liballtoall_location.so: check-env ${COMMON_OBJECTS} ../common/logger_location.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_LOCATION_TRACKING=1 ${COMMON_OBJECTS} ../common/logger_location.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_location.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_LOCATION_TRACKING=1 -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/logger_location.o ../common/timings.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_location_counts_unequal.so $(LDFLAGS)

liballtoall.so: check-env ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoall.c alltoall_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoall.c -o liballtoall.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DASSUME_COUNTS_EQUAL_ALL_RANKS=0 ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoall.c -o liballtoall_counts_unequal.so $(LDFLAGS)

check-env:
ifdef MPIX_HARMONIZE_PREFIX
    CFLAGS+=-DHAVE_MPIX_HARMONIZE=1 -I$(MPIX_HARMONIZE_PREFIX)/include
    LDFLAGS+=-L$(MPIX_HARMONIZE_PREFIX)/lib64 -Wl,-rpath $(MPIX_HARMONIZE_PREFIX)/lib64 -lmpix-harmonize -lmpits
endif

check: all 

clean:
	@rm -f *.so *.o
	@rm -f grouping_test compress_array_test patterns_detection_test 

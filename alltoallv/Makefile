#
# Copyright (c) 2020-2022, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

include ../makefile_common.mk

all: liballtoallv.so                   \
	liballtoallv_location.so           \
	liballtoallv_counts.so             \
	liballtoallv_exec_timings.so       \
	liballtoallv_backtrace.so          \
	liballtoallv_savebuffcontent.so    \
	liballtoallv_comparebuffcontent.so \
	liballtoallv_late_arrival.so 

liballtoallv_counts.so: ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o ../common/logger_for_counts.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o ../common/logger_counts.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_counts.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o ../common/logger_counts.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_counts_notcompact.so

liballtoallv_exec_timings.so: ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_EXEC_TIMING=1 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_exec_timings.so

liballtoallv_late_arrival.so: ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_late_arrival.so

liballtoallv_backtrace.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_BACKTRACE=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_backtrace.so

liballtoallv_location.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LOCATION_TRACKING=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_location.so

liballtoallv_savebuffcontent.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_SAVE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_savebuffcontent.so -lssl -lcrypto

liballtoallv_comparebuffcontent.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_COMPARE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_comparebuffcontent.so -lssl -lcrypto

liballtoallv.so: ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC  ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv.so -lssl -lcrypto

check: all

clean:
	@rm -f *.so *.o
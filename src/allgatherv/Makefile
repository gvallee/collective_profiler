#
# Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

include ../makefile_common.mk

all: liballgatherv.so                   \
	liballgatherv_location.so           \
	liballgatherv_counts.so             \
	liballgatherv_exec_timings.so       \
	liballgatherv_backtrace.so          \
	liballgatherv_savebuffcontent.so    \
	liballgatherv_comparebuffcontent.so \
	liballgatherv_late_arrival.so 

liballgatherv_counts.so: ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_counts.so
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_counts_notcompact.so

liballgatherv_exec_timings.so: ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_EXEC_TIMING=1 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_exec_timings.so

liballgatherv_late_arrival.so: ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_late_arrival.so

liballgatherv_backtrace.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_BACKTRACE=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_backtrace.so

liballgatherv_location.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_LOCATION_TRACKING=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_location.so

liballgatherv_savebuffcontent.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_SAVE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_savebuffcontent.so -lssl -lcrypto

liballgatherv_comparebuffcontent.so: ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC -DENABLE_COMPARE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_comparebuffcontent.so -lssl -lcrypto

liballgatherv.so: ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC  ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv.so -lssl -lcrypto

check: all

clean:
	@rm -f *.so *.o
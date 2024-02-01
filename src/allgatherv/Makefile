#
# Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

include ../makefile_common.mk

all: liballgatherv.so                   \
	liballgatherv_location.so           \
	liballgatherv_counts.so             \
	liballgatherv_displs.so				\
	liballgatherv_exec_timings.so       \
	liballgatherv_backtrace.so          \
	liballgatherv_savebuffcontent.so    \
	liballgatherv_comparebuffcontent.so \
	liballgatherv_late_arrival.so 

liballgatherv_displs.so: check-env ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_displs.o ../common/logger_displs.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ $(CFLAGS) -g -shared -Wall -fPIC -DENABLE_DISPLS=1 ../common/logger_for_displs.o ${COMMON_OBJECTS} ../common/timings.o ../common/logger_displs.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_displs.so $(LDFLAGS)

liballgatherv_counts.so: check-env ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o ../common/logger_for_counts.o  mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g $(CFLAGS) -shared -Wall -fPIC -DENABLE_RAW_DATA=1 -DENABLE_COUNTS=1 ../common/logger_for_counts.o ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_counts.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS) -DENABLE_COMPACT_FORMAT=0 -DENABLE_COUNTS=1 -DENABLE_RAW_DATA=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o  ../common/logger_counts.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_counts_notcompact.so $(LDFLAGS)

liballgatherv_exec_timings.so: check-env ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS) -DENABLE_EXEC_TIMING=1 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_exec_timings.so $(LDFLAGS)

liballgatherv_late_arrival.so: check-env ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS) -DENABLE_LATE_ARRIVAL_TIMING=1 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_late_arrival.so $(LDFLAGS)

liballgatherv_backtrace.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS) -DENABLE_BACKTRACE=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_backtrace.so $(LDFLAGS)

liballgatherv_location.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS) -DENABLE_LOCATION_TRACKING=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_location.so $(LDFLAGS)

liballgatherv_savebuffcontent.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS) -DENABLE_SAVE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_savebuffcontent.so -lssl -lcrypto $(LDFLAGS)

liballgatherv_comparebuffcontent.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS) -DENABLE_COMPARE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv_comparebuffcontent.so -lssl -lcrypto $(LDFLAGS)

liballgatherv.so: check-env ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_allgatherv.c allgatherv_profiler.h
	mpicc -I../ -I../common/ -g -shared -Wall -fPIC $(CFLAGS)  ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_allgatherv.c -o liballgatherv.so -lssl -lcrypto $(LDFLAGS)

check-env:
ifdef MPIX_HARMONIZE_PREFIX
    CFLAGS+=-DHAVE_MPIX_HARMONIZE=1 -I$(MPIX_HARMONIZE_PREFIX)/include
    LDFLAGS+=-L$(MPIX_HARMONIZE_PREFIX)/lib64 -Wl,-rpath $(MPIX_HARMONIZE_PREFIX)/lib64 -lmpix-harmonize -lmpits
endif

check: all

clean:
	@rm -f *.so *.o

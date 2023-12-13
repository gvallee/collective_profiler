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

liballtoallv_counts.so: check-env ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o ../common/logger_for_counts.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_RAW_DATA=1 -DENABLE_COUNTS=1 ../common/logger_for_counts.o ${COMMON_OBJECTS} ../common/timings.o ../common/logger_counts.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_counts.so $(LDFLAGS)
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 -DENABLE_COUNTS=1 ${COMMON_OBJECTS} ../common/timings.o ../common/logger_for_counts.o ../common/logger_counts.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_counts_notcompact.so $(LDFLAGS)

liballtoallv_exec_timings.so: check-env ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_EXEC_TIMING=1 ${COMMON_OBJECTS} ../common/exec_timings.o ../common/logger_exec_timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_exec_timings.so $(LDFLAGS)

liballtoallv_late_arrival.so: check-env ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_LATE_ARRIVAL_TIMING=1 ${COMMON_OBJECTS} ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_late_arrival.so $(LDFLAGS)

liballtoallv_backtrace.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_BACKTRACE=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_backtrace.so $(LDFLAGS)

liballtoallv_location.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_LOCATION_TRACKING=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_location.so $(LDFLAGS)

liballtoallv_savebuffcontent.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_SAVE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_savebuffcontent.so -lssl -lcrypto $(LDFLAGS)

liballtoallv_comparebuffcontent.so: check-env ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC $(CFLAGS) -DENABLE_COMPARE_DATA_VALIDATION=1 ${COMMON_OBJECTS} ../common/logger.o ../common/timings.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv_comparebuffcontent.so -lssl -lcrypto $(LDFLAGS)

liballtoallv.so: check-env ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC  $(CFLAGS) ${COMMON_OBJECTS} ../common/timings.o ../common/logger.o ../common/buff_content.o mpi_alltoallv.c -o liballtoallv.so -lssl -lcrypto $(LDFLAGS)

check-env:
ifdef MPIX_HARMONIZE_PREFIX
    CFLAGS+=-DHAVE_MPIX_HARMONIZE=1 -I$(MPIX_HARMONIZE_PREFIX)/include
    LDFLAGS+=-L$(MPIX_HARMONIZE_PREFIX)/lib64 -Wl,-rpath $(MPIX_HARMONIZE_PREFIX)/lib64 -lmpix-harmonize -lmpits
endif

check: all

clean:
	@rm -f *.so *.o

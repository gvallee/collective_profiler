#
# Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

all: liballtoallv.so liballtoallv_location.so liballtoallv_counts.so liballtoallv_late_arrival.so liballtoallv_exec_timings.so liballtoallv_backtrace.so

liballtoallv_counts.so: ../common/format.o ../common/comm.o ../common/backtrace.o ../common/timings.o ../common/logger_counts.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_RAW_DATA=1 ../common/format.o ../common/comm.o ../common/backtrace.o ../common/timings.o ../common/logger_counts.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_counts.so
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_COMPACT_FORMAT=0 -DENABLE_RAW_DATA=1 ../common/format.o ../common/comm.o ../common/backtrace.o ../common/logger_counts.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_counts_notcompact.so

liballtoallv_exec_timings.so: ../common/format.o ../common/comm.o ../common/backtrace.o ../common/exec_timings.o ../common/logger_exec_timings.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_EXEC_TIMING=1 ../common/format.o ../common/comm.o ../common/backtrace.o ../common/exec_timings.o ../common/logger_exec_timings.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_exec_timings.so

liballtoallv_late_arrival.so: ../common/format.o ../common/comm.o ../common/backtrace.o ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 ../common/format.o ../common/comm.o ../common/backtrace.o ../common/late_arrival_timings.o ../common/logger_late_arrival_timings.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_late_arrival.so

liballtoallv_backtrace.so: ../common/format.o ../common/comm.o ../common/backtrace.o ../common/logger_backtrace.o ../common/timings.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_BACKTRACE=1 ../common/format.o ../common/comm.o ../common/backtrace.o ../common/logger_backtrace.o ../common/timings.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_backtrace.so

liballtoallv_location.so: ../common/format.o ../common/comm.o ../common/backtrace.o ../common/logger_location.o ../common/timings.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC -DENABLE_LOCATION_TRACKING=1 ../common/format.o ../common/comm.o ../common/backtrace.o ../common/logger_location.o ../common/timings.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv_location.so

liballtoallv.so: ../common/format.o ../common/comm.o ../common/backtrace.o ../common/logger.o ../common/grouping.o mpi_alltoallv.c alltoallv_profiler.h
	mpicc -I../ -I../common/ -g -shared -fPIC ../common/format.o ../common/comm.o ../common/backtrace.o ../common/timings.o ../common/logger.o ../common/grouping.o mpi_alltoallv.c -o liballtoallv.so

check: all

clean:
	@rm -f *.so *.o
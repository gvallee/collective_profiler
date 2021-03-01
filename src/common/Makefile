#
# Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

FORMATVERSION := `cat ../../FORMAT_VERSION`

all: \
	format.o                      \
	comm.o                        \
	location.o                    \
	timings.o                     \
	exec_timings.o                \
	late_arrival_timings.o        \
	backtrace.o                   \
	logger.o                      \
	logger_counts.o               \
	logger_exec_timings.o         \
	logger_late_arrival_timings.o \
	logger_backtrace.o            \
	logger_location.o             \
	pattern.o                     \
	grouping.o                    \
	grouping_test                 \
	compress_array_test           \
	patterns_detection_test

format.o: format.c format.h
	$(CC) -I../ -fPIC -c format.c

location.o: location.c location.h format.h
	mpicc -I../ -fPIC -DFORMAT_VERSION=${FORMATVERSION} -c location.c

backtrace.o: backtrace.c backtrace.h format.h
	mpicc -I../ -fPIC -DFORMAT_VERSION=${FORMATVERSION} -c backtrace.c

comm.o: comm.c comm.h
	mpicc -I../ -fPIC -DFORMAT_VERSION=${FORMATVERSION} -c comm.c

timings.o: timings.c timings.h comm.o 
	mpicc -I../ -fPIC -DFORMAT_VERSION=${FORMATVERSION} -c timings.c -o timings.o 

exec_timings.o: timings.c timings.h comm.o
	mpicc -I../ -fPIC -DENABLE_EXEC_TIMING=1 -DFORMAT_VERSION=${FORMATVERSION} -c timings.c -o exec_timings.o

late_arrival_timings.o: timings.c timings.h comm.o
	mpicc -I../ -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 -DFORMAT_VERSION=${FORMATVERSION} -c timings.c -o late_arrival_timings.o

logger.o: logger.c logger.h
	mpicc -I../ -fPIC -c logger.c -o logger.o

# logger object with only counts profiling enabled. This avoids having tons of condition statements in the data path when profiling
logger_counts.o: logger.c logger.h 
	mpicc -I../ -fPIC -DENABLE_RAW_DATA=1 -c logger.c -o logger_counts.o

# logger object with only execution timing profiling enabled. This avoids having tons of condition statements in the data path when profiling
logger_exec_timings.o: logger.c logger.h 
	mpicc -I../ -fPIC -DENABLE_EXEC_TIMING=1 -c logger.c -o logger_exec_timings.o

# logger object with only late arrivel timing profiling enabled. This avoids having tons of condition statements in the data path when profiling
logger_late_arrival_timings.o: logger.c logger.h 
	mpicc -I../ -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 -c logger.c -o logger_late_arrival_timings.o

# logger object with only backtrace profiling enabled. This avoids having tons of condition statements in the data path when profiling
logger_backtrace.o: logger.c logger.h 
	mpicc -I../ -fPIC -DENABLE_BACKTRACE=1 -c logger.c -o logger_backtrace.o

# logger object with only rank location profiling enabled. This avoids having tons of condition statements in the data path when profiling
logger_location.o: logger.c logger.h 
	mpicc -I../ -fPIC -DENABLE_LOCATION_TRACKING=1 -c logger.c -o logger_location.o

pattern.o: pattern.c pattern.h
	$(CC) -I../ -fPIC -c pattern.c

grouping.o: grouping.c grouping.h
	$(CC) -I../ -fPIC -c grouping.c

grouping_test: grouping.o grouping_test.c
	$(CC) -I../ -fPIC grouping.o grouping_test.c -o grouping_test

compress_array_test: logger.o compress_array_test.c
	$(CC) -I../ -fPIC format.o compress_array_test.c -o compress_array_test

patterns_detection_test: pattern.o patterns_detection_test.c
	$(CC) -I../ -fPIC pattern.o patterns_detection_test.c -o patterns_detection_test

check_patterns_detection: patterns_detection_test
	./patterns_detection_test

check_compress_array: compress_array_test
	./compress_array_test

check_grouping: grouping_test
	./grouping_test

check: all check_grouping check_compress_array check_patterns_detection

clean:
	@rm -f *.so *.o
	@rm -f grouping_test compress_array_test patterns_detection_test 
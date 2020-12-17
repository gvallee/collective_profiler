#
# Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
# See LICENSE.txt for license information
#

all: logger.o logger_counts.o logger_exec_timings.o logger_late_arrival_timings.o logger_backtrace.o logger_location.o pattern.o grouping.o grouping_test compress_array_test patterns_detection_test

logger.o: logger.c logger.h 
	$(CC) -I../ -fPIC -c logger.c

logger_counts.o: logger.c logger.h 
	$(CC) -I../ -fPIC -DENABLE_RAW_DATA=1 -c logger.c -o logger_counts.o

logger_exec_timings.o: logger.c logger.h 
	$(CC) -I../ -fPIC -DENABLE_EXEC_TIMING=1 -c logger.c -o logger_exec_timings.o

logger_late_arrival_timings.o: logger.c logger.h 
	$(CC) -I../ -fPIC -DENABLE_LATE_ARRIVAL_TIMING=1 -c logger.c -o logger_late_arrival_timings.o

logger_backtrace.o: logger.c logger.h 
	$(CC) -I../ -fPIC -DENABLE_BACKTRACE=1 -c logger.c -o logger_backtrace.o

logger_location.o: logger.c logger.h 
	$(CC) -I../ -fPIC -DENABLE_LOCATION_TRACKING=1 -c logger.c -o logger_location.o

pattern.o: pattern.c pattern.h
	$(CC) -I../ -fPIC -c pattern.c

grouping.o: grouping.c grouping.h
	$(CC) -I../ -fPIC -c grouping.c 

grouping_test: grouping.o grouping_test.c
	$(CC) -I../ -fPIC grouping.o grouping_test.c -o grouping_test

compress_array_test: logger.o compress_array_test.c
	$(CC) -I../ -fPIC logger.o compress_array_test.c -o compress_array_test

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
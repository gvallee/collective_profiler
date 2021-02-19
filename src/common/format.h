/*************************************************************************
 * Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef MPI_COLLECTIVE_PROFILER_FORMAT_H
#define MPI_COLLECTIVE_PROFILER_FORMAT_H

#include <inttypes.h>
#include <string.h>

#include "collective_profiler_config.h"

#define FORMAT_VERSION_WRITE(_fd) (fprintf(_fd, "FORMAT_VERSION: %d\n\n", FORMAT_VERSION))

char *compress_int_array(int *array, int xsize,  int ysize);
char *compress_uint64_array(uint64_t *array, size_t xsize,  size_t ysize);

#endif // MPI_COLLECTIVE_PROFILER_FORMAT_H
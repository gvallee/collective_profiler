/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef COLLECTIVE_PROFILER_EXAMPLE_UTILS_H
#define COLLECTIVE_PROFILER_EXAMPLE_UTILS_H

#define MPICHECK(_c)                                 \
    do                                               \
    {                                                \
        if ((_c) != MPI_SUCCESS)                     \
        {                                            \
            fprintf(stderr, "MPI command failed\n"); \
            return 1;                                \
        }                                            \
    } while (0);

#endif // COLLECTIVE_PROFILER_EXAMPLE_UTILS_H
/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>

#ifndef COLLECTIVE_PROFILER_DATATYPE_H
#define COLLECTIVE_PROFILER_DATATYPE_H

typedef enum type_id
{
    UNKNOWN_ID = 0,
    MPI_CHAR_ID,
    MPI_UNSIGNED_CHAR_ID,
    MPI_SIGNED_CHAR_ID,
    MPI_SHORT_ID,
    MPI_UNSIGNED_SHORT_ID,
    MPI_INT_ID,
    MPI_UNSIGNED_ID,
    MPI_LONG_ID,
    MPI_UNSIGNED_LONG_ID,
    MPI_LONG_LONG_INT_ID,
    MPI_FLOAT_ID,
    MPI_DOUBLE_ID,
    MPI_LONG_DOUBLE_ID,
    MPI_BYTE_ID,
    MPI_CHARACTER_ID,
    MPI_INTEGER_ID,
    MPI_INTEGER1_ID,
    MPI_INTEGER2_ID,
    MPI_INTEGER4_ID,
    MPI_INTEGER8_ID,
    MPI_INTEGER16_ID,
    MPI_REAL_ID,
    MPI_DOUBLE_PRECISION_ID,
    MPI_REAL2_ID,
    MPI_REAL4_ID,
    MPI_REAL8_ID,
    MPI_COMPLEX_ID,
    MPI_DOUBLE_COMPLEX_ID
} type_id_t;

typedef struct datatype_info
{
    bool analyzed;
    bool is_contiguous;
    bool is_predefined;
    int size;
    type_id_t id;

    MPI_Datatype type;
} datatype_info_t;

#endif // COLLECTIVE_PROFILER_DATATYPE_H
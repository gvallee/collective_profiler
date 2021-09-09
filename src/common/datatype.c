/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdbool.h>
#include <stdlib.h>
#include <assert.h>

#include "mpi.h"

#include "datatype.h"

void get_predefined_type(datatype_info_t *info)
{
    if (info->type == MPI_CHAR)
    {
        info->id = MPI_CHAR_ID;
        return;
    }
    if (info->type == MPI_UNSIGNED_CHAR)
    {
        info->id = MPI_UNSIGNED_CHAR_ID;
        return;
    }
    if (info->type == MPI_SIGNED_CHAR)
    {
        info->id = MPI_SIGNED_CHAR_ID;
        return;
    }
    if (info->type == MPI_SHORT)
    {
        info->id = MPI_SHORT_ID;
        return;
    }
    if (info->type == MPI_UNSIGNED_SHORT)
    {
        info->id = MPI_UNSIGNED_SHORT_ID;
        return;
    }
    if (info->type == MPI_INT)
    {
        info->id = MPI_INT_ID;
        return;
    }
    if (info->type == MPI_UNSIGNED)
    {
        info->id = MPI_UNSIGNED_SHORT_ID;
        return;
    }
    if (info->type == MPI_LONG)
    {
        info->id = MPI_LONG_ID;
        return;
    }
    if (info->type == MPI_UNSIGNED_LONG)
    {
        info->id = MPI_UNSIGNED_LONG_ID;
        return;
    }
    if (info->type == MPI_LONG_LONG_INT)
    {
        info->id = MPI_LONG_LONG_INT_ID;
        return;
    }
    if (info->type == MPI_FLOAT)
    {
        info->id = MPI_FLOAT_ID;
        return;
    }
    if (info->type == MPI_DOUBLE)
    {
        info->id = MPI_DOUBLE_ID;
        return;
    }
    if (info->type == MPI_LONG_DOUBLE)
    {
        info->id = MPI_LONG_DOUBLE_ID;
        return;
    }
    if (info->type == MPI_BYTE)
    {
        info->id = MPI_BYTE_ID;
        return;
    }
    if (info->type == MPI_CHARACTER)
    {
        info->id = MPI_CHARACTER_ID;
        return;
    }
    if (info->type == MPI_INTEGER)
    {
        info->id = MPI_INTEGER_ID;
        return;
    }
    if (info->type == MPI_INTEGER1)
    {
        info->id = MPI_INTEGER1_ID;
        return;
    }
    if (info->type == MPI_INTEGER2)
    {
        info->id = MPI_INTEGER2_ID;
        return;
    }
    if (info->type == MPI_INTEGER4)
    {
        info->id = MPI_INTEGER4_ID;
        return;
    }
    if (info->type == MPI_INTEGER8)
    {
        info->id = MPI_INTEGER8_ID;
        return;
    }
#if 0
    if (info->type ==  MPI_INTEGER16) {
            info->id = MPI_INTEGER16_ID;
            return;
        }
#endif
    if (info->type == MPI_REAL)
    {
        info->id = MPI_REAL_ID;
        return;
    }
    if (info->type == MPI_DOUBLE_PRECISION)
    {
        info->id = MPI_DOUBLE_PRECISION_ID;
        return;
    }
#if 0
    if (info->type ==  MPI_REAL2) {
            info->id = MPI_REAL2_ID;
            return;
        }
#endif
    if (info->type == MPI_REAL4)
    {
        info->id = MPI_REAL4_ID;
        return;
    }
    if (info->type == MPI_REAL8)
    {
        info->id = MPI_REAL8_ID;
        return;
    }
    if (info->type == MPI_COMPLEX)
    {
        info->id = MPI_COMPLEX_ID;
        return;
    }
    if (info->type == MPI_DOUBLE_COMPLEX)
    {
        info->id = MPI_DOUBLE_COMPLEX_ID;
        return;
    }

    info->id = UNKNOWN_ID;
}

int analyze_datatype(MPI_Datatype type, datatype_info_t **info)
{
    int dt_num_intergers;
    int dt_num_addresses;
    int dt_num_datatypes;
    int dt_combiner;

    // A little bit of boiler plate code because we want to be able to support
    // the situation where the caller is allocating a datatype_info_t structure
    // or want the function to allocate it.
    assert(info);
    datatype_info_t *i;
    if (*info == NULL)
    {
        i = (datatype_info_t *)malloc(sizeof(datatype_info_t));
        if (i == NULL)
            return -1;
        *info = i;
        i->analyzed = false;
    }
    else
    {
        i = *info;
        if (i->analyzed)
            return 0;
    }

    i->is_contiguous = false;
    i->is_predefined = false;
    i->type = type;

    PMPI_Type_size(type, &(i->size));
    PMPI_Type_get_envelope(type, &dt_num_intergers, &dt_num_addresses, &dt_num_datatypes, &dt_combiner);

    if (dt_combiner == MPI_COMBINER_NAMED)
    {
        i->is_contiguous = true;
        i->is_predefined = true;
        get_predefined_type(i);
    }

    if (dt_combiner == MPI_COMBINER_CONTIGUOUS)
    {
        i->is_contiguous = true;
    }

    i->analyzed = true;
    return 0;
}
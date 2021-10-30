/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdlib.h>
#include <stdio.h>

#include "collective_profiler_config.h"
#include "common_utils.h"
#include "comm.h"

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

static inline char *
type_id_to_str(int id)
{
    switch (id)
    {
    case UNKNOWN_ID:
        return "UKNOWN";
    case MPI_CHAR_ID:
        return "MPI_CHAR";
    case MPI_UNSIGNED_CHAR_ID:
        return "MPI_UNSIGNED_CHAR";
    case MPI_SIGNED_CHAR_ID:
        return "MPI_SIGNED_CHAR";
    case MPI_SHORT_ID:
        return "MPI_SHORT";
    case MPI_UNSIGNED_SHORT_ID:
        return "MPI_UNSIGNED_SHORT";
    case MPI_INT_ID:
        return "MPI_INT";
    case MPI_UNSIGNED_ID:
        return "MPI_UNSIGNED";
    case MPI_LONG_ID:
        return "MPI_LONG";
    case MPI_UNSIGNED_LONG_ID:
        return "MPI_UNSIGNED_LONG";
    case MPI_LONG_LONG_INT_ID:
        return "MPI_LONG_LONG_INT";
    case MPI_FLOAT_ID:
        return "MPI_FLOAT";
    case MPI_DOUBLE_ID:
        return "MPI_DOUBLE";
    case MPI_LONG_DOUBLE_ID:
        return "MPI_LONG_DOUBLE";
    case MPI_BYTE_ID:
        return "MPI_BYTE";
    case MPI_CHARACTER_ID:
        return "MPI_CHARACTER";
    case MPI_INTEGER_ID:
        return "MPI_INTEGER";
    case MPI_INTEGER1_ID:
        return "MPI_INTEGER1";
    case MPI_INTEGER2_ID:
        return "MPI_INTEGER2";
    case MPI_INTEGER4_ID:
        return "MPI_INTEGER4";
    case MPI_INTEGER8_ID:
        return "MPI_INTEGER8";
    case MPI_INTEGER16_ID:
        return "MPI_INTEGER16";
    case MPI_REAL_ID:
        return "MPI_REAL";
    case MPI_DOUBLE_PRECISION_ID:
        return "MPI_DOUBLE_PRECISION";
    case MPI_REAL2_ID:
        return "MPI_REAL2";
    case MPI_REAL4_ID:
        return "MPI_REAL4";
    case MPI_REAL8_ID:
        return "MPI_REAL8";
    case MPI_COMPLEX_ID:
        return "MPI_COMPLEX";
    case MPI_DOUBLE_COMPLEX_ID:
        return "MPI_DOUBLE_COMPLEX";
    }

    return "error";
}

static inline int
open_datatype_info_file(char *collective_name, uint32_t comm_id, int world_rank, uint64_t call_id, char *ctxt, char **file_name, FILE **file)
{
    char *filename = NULL;
    int rc;

    if (ctxt == NULL)
    {
        if (getenv(OUTPUT_DIR_ENVVAR))
        {
            _asprintf(filename, rc, "%s/%s_datatype-info_comm%" PRIu32 "_rank%d_call%" PRIu64 ".md", getenv(OUTPUT_DIR_ENVVAR), collective_name, comm_id, world_rank, call_id);
            assert(rc > 0);
        }
        else
        {
            _asprintf(filename, rc, "%s_datatype-info_comm%" PRIu32 "_rank%d_call%" PRIu64 ".md", collective_name, comm_id, world_rank, call_id);
            assert(rc > 0);
        }
    }
    else
    {
        if (getenv(OUTPUT_DIR_ENVVAR))
        {
            _asprintf(filename, rc, "%s/%s_datatype-info_%s_comm%" PRIu32 "_rank%d_call%" PRIu64 ".md", getenv(OUTPUT_DIR_ENVVAR), collective_name, ctxt, comm_id, world_rank, call_id);
            assert(rc > 0);
        }
        else
        {
            _asprintf(filename, rc, "%s_datatype-info_%s_comm%" PRIu32 "_rank%d_call%" PRIu64 ".md", collective_name, ctxt, comm_id, world_rank, call_id);
            assert(rc > 0);
        }
    }

    FILE *f = fopen(filename, "w");
    assert(f);

    *file = f;
    *file_name = filename;
    return 0;
}

static inline void
get_predefined_type(datatype_info_t *info)
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

static inline int
analyze_datatype(MPI_Datatype type, datatype_info_t *i)
{
    int dt_num_intergers;
    int dt_num_addresses;
    int dt_num_datatypes;
    int dt_combiner;

    assert(i);
    if (i->analyzed)
        return 0;

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

static inline int
save_datatype_info(char *collective_name, MPI_Comm comm, int comm_rank, int world_rank, uint64_t call_id, char *ctxt, datatype_info_t *dt_info)
{
    uint32_t comm_id;
    GET_COMM_LOGGER(comm, world_rank, comm_rank, comm_id);

    char *filename = NULL;
    FILE *file = NULL;
    int rc = open_datatype_info_file(collective_name, comm_id, world_rank, call_id, ctxt, &filename, &file);
    if (rc)
        return rc;

    if (dt_info->is_predefined)
    {
        char *type_id = type_id_to_str(dt_info->id);
        fprintf(file, "Predefined type: %s\n", type_id);
    }
    fprintf(file, "Size: %d\n", dt_info->size);
    fprintf(file, "Datatype is contiguous: %d\n", dt_info->is_contiguous);
    fprintf(file, "Datatype is pre-defined: %d\n", dt_info->is_predefined);

    fclose(file);
    free(filename);
    filename = NULL;
    return 0;
}

#endif // COLLECTIVE_PROFILER_DATATYPE_H
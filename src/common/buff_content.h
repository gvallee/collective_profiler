/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#ifndef MPI_COLLECTIVE_PROFILER_BUFFCONTENT_H
#define MPI_COLLECTIVE_PROFILER_BUFFCONTENT_H

#include <stdlib.h>
#include <inttypes.h>
#include <stdbool.h>

#include "mpi.h"

#include "collective_profiler_config.h"
#include "common_utils.h"
#include "format.h"
#include "comm.h"

#define COLLECTIVE_PROFILER_MAX_CALL_CHECK_BUFF_CONTENT_ENVVAR "COLLECTIVE_PROFILER_MAX_CALL_CHECK_BUFF_CONTENT"
#define COLLECTIVE_PROFILER_CHECK_SEND_BUFF_ENVVAR "COLLECTIVE_PROFILER_CHECK_SEND_BUFF"

#define SEND_CONTEXT_ID "send"
#define RECV_CONTEXT_ID "recv"
#define SEND_CONTEXT_IDX (0)
#define RECV_CONTEXT_IDX (1)
#define MAX_LOGGER_CONTEXTS (2) // Recv and send contexts

typedef struct logger_context
{
    char *name;
    FILE *fd;
    char *filename;
} logger_context_t;

// buffcontent_logger is the central structure to track and profile backtrace in
// the context of MPI collective. We track in a unique manner each trace but for each
// trace, multiple contexts can be tracked. A context is the tuple communictor id/rank/call.
typedef struct buffcontent_logger
{
    char *collective_name;
    uint64_t id;
    int world_rank;
    uint64_t comm_id;
    MPI_Comm comm;
    logger_context_t ctxt[MAX_LOGGER_CONTEXTS];
    struct buffcontent_logger *next;
    struct buffcontent_logger *prev;
} buffcontent_logger_t;

extern buffcontent_logger_t *buffcontent_loggers_head;
extern buffcontent_logger_t *buffcontent_loggers_tail;

static inline void
_display_config(int dt_num_intergers, int dt_num_addresses, int dt_num_datatypes, int dt_combiner)
{
    fprintf(stderr, "-> Num datatypes: %d\n", dt_num_datatypes);
    switch (dt_combiner)
    {
    case MPI_COMBINER_NAMED:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_NAMED\n");
        break;
    }
    case MPI_COMBINER_DUP:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_DUP\n");
        break;
    }
    case MPI_COMBINER_CONTIGUOUS:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_CONTIGUOUS\n");
        break;
    }
    case MPI_COMBINER_VECTOR:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_VECTOR\n");
        break;
    }
    case MPI_COMBINER_HVECTOR:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_HVECTOR\n");
        break;
    }
    case MPI_COMBINER_INDEXED:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_INDEXED\n");
        break;
    }
    case MPI_COMBINER_HINDEXED:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_HINDEXED\n");
        break;
    }
    case MPI_COMBINER_INDEXED_BLOCK:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_INDEXED_BLOCK\n");
        break;
    }
    case MPI_COMBINER_STRUCT:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_STRUCT\n");
        break;
    }
    case MPI_COMBINER_SUBARRAY:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_SUBARRAY\n");
        break;
    }
    case MPI_COMBINER_DARRAY:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_DARRAY\n");
        break;
    }
    case MPI_COMBINER_F90_REAL:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_F90_REAL\n");
        break;
    }
    case MPI_COMBINER_F90_COMPLEX:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_F90_COMPLEX\n");
        break;
    }
    case MPI_COMBINER_F90_INTEGER:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_F90_INTEGER\n");
        break;
    }
    case MPI_COMBINER_RESIZED:
    {
        fprintf(stderr, "-> Combiner: MPI_COMBINER_RESIZED\n");
        break;
    }
    default:

        fprintf(stderr, "-> Combiner: unknown\n");
    }
}

#define DT_CHECK(dt)                                                                                                \
    do                                                                                                              \
    {                                                                                                               \
        int dt_num_intergers;                                                                                       \
        int dt_num_addresses;                                                                                       \
        int dt_num_datatypes;                                                                                       \
        int dt_combiner;                                                                                            \
        PMPI_Type_get_envelope(dt, &dt_num_intergers, &dt_num_addresses, &dt_num_datatypes, &dt_combiner);          \
        if (dt_num_datatypes > 1 || !(dt_combiner == MPI_COMBINER_CONTIGUOUS || dt_combiner == MPI_COMBINER_NAMED)) \
        {                                                                                                           \
            fprintf(stderr, "Unsupported datatype configuration\n");                                                \
            _display_config(dt_num_intergers, dt_num_addresses, dt_num_datatypes, dt_combiner);                     \
            MPI_Abort(MPI_COMM_WORLD, 1);                                                                           \
        }                                                                                                           \
    } while (0)

static inline int
lookup_buffcontent_logger(char *collective_name, MPI_Comm comm, buffcontent_logger_t **logger)
{
    buffcontent_logger_t *ptr = buffcontent_loggers_head;
    while (ptr != NULL)
    {
        if (strcmp(ptr->collective_name, collective_name) == 0 && ptr->comm == comm)
        {

            *logger = ptr;
            return 0;
        }
        ptr = ptr->next;
    }

    *logger = NULL;
    return 0;
}

static inline int
open_content_storage_file(char *collective_name, char **filename, FILE **file, uint64_t comm_id, int world_rank, int ctxt, char *mode)
{
    char *_filename = NULL;
    int rc;
    if (ctxt < 0 || ctxt > 1)
    {
        if (getenv(OUTPUT_DIR_ENVVAR))
        {
            _asprintf(_filename, rc, "%s/%s_buffcontent_comm%" PRIu64 "_rank%d.txt", getenv(OUTPUT_DIR_ENVVAR), collective_name, comm_id, world_rank);
            assert(rc > 0);
        }
        else
        {
            _asprintf(_filename, rc, "%s_buffcontent_comm%" PRIu64 "_rank%d.txt", collective_name, comm_id, world_rank);
            assert(rc > 0);
        }
    }
    else
    {
        char *_ctxt = "send";
        if (ctxt == RECV_CONTEXT_IDX)
            _ctxt = "recv";
        if (getenv(OUTPUT_DIR_ENVVAR))
        {
            _asprintf(_filename, rc, "%s/%s_buffcontent_comm%" PRIu64 "_rank%d_%s.txt", getenv(OUTPUT_DIR_ENVVAR), collective_name, comm_id, world_rank, _ctxt);
            assert(rc > 0);
        }
        else
        {
            _asprintf(_filename, rc, "%s_buffcontent_comm%" PRIu64 "_rank%d_%s.txt", collective_name, comm_id, world_rank, _ctxt);
            assert(rc > 0);
        }
    }

    FILE *f = fopen(_filename, mode);
    assert(f);

    *file = f;
    *filename = _filename;
    return 0;
}

static inline int
init_buffcontent_logger(char *collective_name, int world_rank, MPI_Comm comm, uint64_t comm_id, buffcontent_logger_t **buffcontent_logger)
{
    assert(collective_name);
    buffcontent_logger_t *new_logger = malloc(sizeof(buffcontent_logger_t));
    assert(new_logger);
    new_logger->collective_name = strdup(collective_name);
    new_logger->world_rank = world_rank;
    new_logger->comm_id = comm_id;
    new_logger->comm = comm;
    new_logger->prev = NULL;
    new_logger->next = NULL;
    new_logger->ctxt[0].fd = NULL;
    new_logger->ctxt[1].fd = NULL;
    new_logger->ctxt[0].filename = NULL;
    new_logger->ctxt[1].filename = NULL;

    if (buffcontent_loggers_head == NULL)
    {
        buffcontent_loggers_head = new_logger;
        buffcontent_loggers_tail = new_logger;
        new_logger->id = 0;
    }
    else
    {
        buffcontent_loggers_tail->next = new_logger;
        new_logger->prev = buffcontent_loggers_tail;
        new_logger->id = buffcontent_loggers_tail->id + 1;
        buffcontent_loggers_tail = new_logger;
    }

    *buffcontent_logger = new_logger;
    return 0;
}

static inline int
get_buffcontent_logger(char *collective_name, int ctxt, char *mode, MPI_Comm comm, int world_rank, int comm_rank, buffcontent_logger_t **buffcontent_logger)
{
    int rc;
    uint32_t comm_id;
    buffcontent_logger_t *logger = NULL;
    rc = lookup_comm(comm, &comm_id);
    if (rc)
    {
        rc = add_comm(comm, world_rank, comm_rank, &comm_id);
        if (rc)
        {
            fprintf(stderr, "add_comm() failed: %d\n", rc);
            return 1;
        }
    }
    rc = lookup_buffcontent_logger(collective_name, comm, &logger);
    if (rc)
    {
        fprintf(stderr, "lookup_buffcontent_logger() failed: %d\n", rc);
        return 1;
    }
    if (logger == NULL)
    {
        rc = init_buffcontent_logger(collective_name, world_rank, comm, comm_id, &logger);
        if (rc)
        {
            fprintf(stderr, "init_buffcontent_logger() failed: %d\n", rc);
            return 1;
        }
    }
    assert(logger);
    if (logger->ctxt[ctxt].fd == NULL)
    {
        rc = open_content_storage_file(logger->collective_name,
                                       &(logger->ctxt[ctxt].filename),
                                       &(logger->ctxt[ctxt].fd),
                                       comm_id,
                                       logger->world_rank,
                                       ctxt,
                                       mode);
        if (rc)
        {
            fprintf(stderr, "_open_content_storage_files() failed: %d\n", rc);
            return 1;
        }
        if (strcmp(mode, "w") == 0)
        {
            // Write the format version at the begining of the file
            FORMAT_VERSION_WRITE(logger->ctxt[ctxt].fd);
        }
        else
        {
            // Read the format version so we can continue to read the file once we return from the function
            int version;
            fscanf(logger->ctxt[ctxt].fd, "FORMAT_VERSION: %d\n\n", &version);
            if (version != FORMAT_VERSION)
            {
                fprintf(stderr, "incompatible version (%d vs. %d)\n", version, FORMAT_VERSION);
                return -1;
            }
        }
    }
    assert(logger->ctxt[ctxt].filename);
    assert(logger->ctxt[ctxt].fd);
    *buffcontent_logger = logger;
    return 0;
}

static inline void
save_buf_content(void *buf, const int *counts, const int *displs, MPI_Datatype type, MPI_Comm comm, int rank, char *ctxt)
{
    assert(buf);
    assert(counts);
    assert(displs);
    assert(ctxt);

    int size;
    PMPI_Comm_size(comm, &size);
    int dtsz;
    PMPI_Type_size(type, &dtsz);

    char *filename = NULL;
    int rc;
    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        _asprintf(filename, rc, "%s/data_%s_rank%d.txt", getenv(OUTPUT_DIR_ENVVAR), ctxt, rank);
        assert(rc > 0);
    }
    else
    {
        _asprintf(filename, rc, "data_%s_rank%d.txt", ctxt, rank);
        assert(rc > 0);
    }

    FILE *f = fopen(filename, "w");
    assert(f);

    int i;
    for (i = 0; i < size; i++)
    {
        int displ = displs[i];
        int count = counts[i];
        off_t offset = displ * dtsz;
        //size_t data_size = count *dtsz;

        // We assume the data is contiguous and that the type is of a type compatible with a C double
        double *ptr = (double *)((ptrdiff_t)buf + offset);
        int j;
        for (j = 0; j < count; j++)
        {
            fprintf(f, "%f ", ptr[j]);
        }
        fprintf(f, "\n");
    }

    fclose(f);
    free(filename);
}

int store_call_data(char *collective_name, int ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], MPI_Datatype dt);
int store_call_data_single_count(char *collective_name, int ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int count, MPI_Datatype dt);
int read_and_compare_call_data(char *collective_name, int ctxt, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, void *buf, int counts[], int displs[], MPI_Datatype dt, bool check);
int release_buffcontent_loggers();

#endif // MPI_COLLECTIVE_PROFILER_BUFFCONTENT_H

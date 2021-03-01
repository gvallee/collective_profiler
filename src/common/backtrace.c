/*************************************************************************
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <sys/stat.h>
#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <assert.h>
#include <sys/types.h>
#include <unistd.h>

#include "backtrace.h"
#include "collective_profiler_config.h"
#include "common_utils.h"
#include "comm.h"
#include "format.h"

#include "mpi.h"

backtrace_logger_t *trace_loggers_head = NULL;
backtrace_logger_t *trace_loggers_tail = NULL;
uint64_t trace_id = 0;

static inline void _write_backtrace_info(FILE *f)
{
    assert(f);
    char pid_buf[30];
    int size = sprintf(pid_buf, "%d", getpid()); // The system file's name used to get the backtrace is based on the PID
    assert(size < 30);
    char name_buf[512];
    name_buf[readlink("/proc/self/exe", name_buf, 511)] = 0;
    fprintf(f, "stack trace for %s pid=%s\n", name_buf, pid_buf);
}

static inline int _open_backtrace_file(char *collective_name, char **backtrace_filename, FILE **backtrace_file, int world_rank, uint64_t id)
{
    char *filename = NULL;
    int rc;
    // filename schema: bracktrace_rank<WORLDRANK>_trace<ID>.md
    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        _asprintf(filename, rc, "%s/%s_backtrace_rank%d_trace%" PRIu64 ".md", getenv(OUTPUT_DIR_ENVVAR), collective_name, world_rank, id);
        assert(rc > 0);
    }
    else
    {
        _asprintf(filename, rc, "%s_backtrace_rank%d_trace%" PRIu64 ".md", collective_name, world_rank, id);
        assert(rc > 0);
    }

    FILE *f = fopen(filename, "w");
    assert(f);

    *backtrace_file = f;
    *backtrace_filename = filename;
    return 0;

exit_on_error:
    *backtrace_file = NULL;
    *backtrace_filename = NULL;
    return -1;
}

int write_backtrace_to_file(backtrace_logger_t *logger)
{
    assert(logger);
    assert(logger->fd);
    _write_backtrace_info(logger->fd);

    uint64_t i;
    fprintf(logger->fd, "\n# Trace\n\n");
    for (i = 0; i < logger->trace_size; i++)
    {

        fprintf(logger->fd, "%s\n", logger->trace[i]);
    }
    fprintf(logger->fd, "\n");

    for (i = 0; i < logger->num_contexts; i++)
    {
        fprintf(logger->fd, "# Context %" PRIu64 "\n\n", i);
        char *str = compress_uint64_array(logger->contexts[i].calls, logger->contexts[i].calls_count, 1);
        assert(str);
        fprintf(logger->fd, "Communicator: %d\n", logger->contexts[i].comm_id);
        fprintf(logger->fd, "Communicator rank: %d\n", logger->contexts[i].comm_rank);
        fprintf(logger->fd, "COMM_WORLD rank: %d\n", logger->contexts[i].world_rank);
        fprintf(logger->fd, "Calls: %s\n", str);
        fprintf(logger->fd, "\n");
        free(str);
    }

    return 0;
}

int lookup_trace_context(backtrace_logger_t *trace_logger, int commID, int comm_rank, trace_context_t **trace_ctxt)
{
    trace_context_t *ptr = trace_logger->contexts;

    while (ptr != NULL)
    {
        if (ptr->comm_id == commID && ptr->comm_rank == comm_rank)
        {
            *trace_ctxt = ptr;
            return 0;
        }
        ptr = ptr->next;
    }

    *trace_ctxt = NULL;
    return 0;
}

int lookup_backtrace(char *collective_name, char **trace, size_t trace_size, backtrace_logger_t **logger)
{
    assert(trace);
    backtrace_logger_t *ptr = trace_loggers_head;
    int i;
    while (ptr != NULL)
    {
        if (ptr->trace_size == trace_size && strcmp(ptr->collective_name, collective_name) == 0)
        {
            bool found = true;
            for (i = 0; i < trace_size; i++)
            {
                if (found == true && strcmp(trace[i], ptr->trace[i]) != 0)
                {
                    found = false;
                }
            }

            if (found == true)
            {
                *logger = ptr;
                return 0;
            }
        }
        ptr = ptr->next;
    }

    *logger = NULL;
    return 0;
}

static inline int _close_backtrace_file(backtrace_logger_t *logger)
{
    if (logger->fd)
    {
        fclose(logger->fd);
        logger->fd = NULL;
    }

    if (logger->filename)
    {
        free(logger->filename);
        logger->filename = NULL;
    }
    
    return 0;
}

int init_backtrace_context(MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call, trace_context_t **trace_ctxt)
{
    uint32_t comm_id;
    GET_COMM_LOGGER(comm, world_rank, comm_rank, comm_id);

    trace_context_t *new_ctxt = malloc(sizeof(trace_context_t));
    assert(new_ctxt);
    // At the moment, we hardcode the initial size of the calls array as size of 2.
    new_ctxt->max_calls = 2;
    new_ctxt->calls = malloc(new_ctxt->max_calls * sizeof(uint64_t));
    assert(new_ctxt->calls);
    new_ctxt->calls[0] = n_call;
    new_ctxt->calls_count = 1;
    new_ctxt->comm_id = comm_id;
    new_ctxt->next = NULL;
    new_ctxt->prev = NULL;
    new_ctxt->comm_rank = comm_rank;
    new_ctxt->world_rank = world_rank;

    *trace_ctxt = new_ctxt;
    return 0;
}

static inline int init_backtrace_logger(char *collective_name, char **trace, size_t trace_size, int world_rank, trace_context_t *trace_ctxt, backtrace_logger_t **trace_logger)
{
    assert(collective_name);
    assert(trace);
    backtrace_logger_t *new_logger = malloc(sizeof(backtrace_logger_t));
    assert(new_logger);
    new_logger->collective_name = strdup(collective_name);
    new_logger->id = trace_id;
    trace_id++;
    new_logger->world_rank = world_rank;
    new_logger->contexts = trace_ctxt;
    new_logger->fd = NULL;
    new_logger->filename = NULL;
    new_logger->num_contexts = 1;
    new_logger->max_contexts = 1;
    new_logger->trace = trace;
    new_logger->trace_size = trace_size;
    new_logger->prev = NULL;
    new_logger->next = NULL;

    int rc = _open_backtrace_file(new_logger->collective_name, &new_logger->filename, &new_logger->fd, new_logger->world_rank, new_logger->id);
    if (rc)
    {
        fprintf(stderr, "_open_backtrace_file() failed: %d\n", rc);
        return -1;
    }
    assert(new_logger->filename);

    if (trace_loggers_head == NULL)
    {
        trace_loggers_head = new_logger;
        trace_loggers_tail = new_logger;
        new_logger->id = 0;
    }
    else
    {
        trace_loggers_tail->next = new_logger;
        new_logger->prev = trace_loggers_tail;
        new_logger->id = trace_loggers_tail->id + 1;
        trace_loggers_tail = new_logger;
    }

    assert(new_logger->fd);
    // Write the format version at the begining of the file
    FORMAT_VERSION_WRITE(new_logger->fd);
    *trace_logger = new_logger;
    return 0;
}

static inline int _fini_trace_contexts(trace_context_t *ctx)
{
    while (ctx != NULL)
    {
        trace_context_t *next = ctx->next;
        if (ctx->calls)
        {
            free(ctx->calls);
            ctx->calls = NULL;
        }
        free(ctx);
        ctx = next;
    }
    return 0;
}

int fini_backtrace_logger(backtrace_logger_t **logger)
{
    write_backtrace_to_file(*logger);
    _close_backtrace_file(*logger);
    _fini_trace_contexts((*logger)->contexts);
    (*logger)->contexts = NULL;
    (*logger)->num_contexts = 0;
    (*logger)->contexts = NULL;

    if ((*logger)->collective_name)
    {
        free((*logger)->collective_name);
        (*logger)->collective_name = NULL;
    }

    if ((*logger)->fd)
    {
        fclose((*logger)->fd);
        (*logger)->fd = NULL;
    }

    if ((*logger)->filename)
    {
        free((*logger)->filename);
        (*logger)->filename = NULL;
    }

    // Secification for the trace buffer says: "This array is malloc(3)ed by backtrace_symbols(), 
    // and must be freed by the caller. (The strings pointed to by the array of pointers need 
    // not and should not be freed.)"
    if ((*logger)->trace)
    {
        free((*logger)->trace);
        (*logger)->trace = NULL;
    }
    *logger = NULL;

    return 0;
}

int release_backtrace_loggers()
{
    int rc;
    backtrace_logger_t *ptr = trace_loggers_head;
    while (ptr)
    {
        backtrace_logger_t *next = ptr->next;
        rc = fini_backtrace_logger(&ptr);
        if (rc)
        {
            fprintf(stderr, "release_backtrace_loggers() failed: %d\n", rc);
            return rc;
        }
        assert(ptr == NULL);
        ptr = next;
    }
    trace_loggers_head = NULL;
    trace_loggers_tail = NULL;
    return 0;
}

int insert_caller_data(char *collective_name, char **trace, size_t trace_size, MPI_Comm comm, int comm_rank, int world_rank, uint64_t n_call)
{
    int i;
    int rc;
    backtrace_logger_t *trace_logger = NULL;

    uint32_t comm_id;
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

    rc = lookup_backtrace(collective_name, trace, trace_size, &trace_logger);
    if (rc)
    {
        fprintf(stderr, "lookup_backtrace() failed: %d\n", rc);
        return 1;
    }

    if (trace_logger == NULL)
    {
        trace_context_t *trace_ctxt = NULL;
        rc = init_backtrace_context(comm, comm_rank, world_rank, n_call, &trace_ctxt);
        if (rc)
        {
            fprintf(stderr, "init_backtrace_context() failed: %d\n", rc);
            return 1;
        }

        // we do not have that trace yet, add it
        rc = init_backtrace_logger(collective_name, trace, trace_size, world_rank, trace_ctxt, &trace_logger);
        if (rc)
        {
            fprintf(stderr, "init_backtrace_logger() failed: %d\n", rc);
            return 1;
        }
    }
    else
    {
        // we already know that trace, we now lookup the context
        trace_context_t *trace_ctxt = NULL;
        rc = lookup_trace_context(trace_logger, comm_id, comm_rank, &trace_ctxt);
        if (rc)
        {
            fprintf(stderr, "lookup_trace_context() failed: %d\n", rc);
            return 1;
        }

        if (trace_ctxt)
        {
            if (trace_ctxt->calls_count >= trace_ctxt->max_calls)
            {
                trace_ctxt->max_calls = trace_ctxt->max_calls * 2;
                trace_ctxt->calls = (uint64_t *)realloc(trace_ctxt->calls, trace_ctxt->max_calls * sizeof(uint64_t));
                assert(trace_ctxt->calls);
            }
            trace_ctxt->calls[trace_ctxt->calls_count] = n_call;
            trace_ctxt->calls_count++;
        }
        else
        {
            trace_context_t *new_trace_ctxt = NULL;
            rc = init_backtrace_context(comm, comm_rank, world_rank, n_call, &new_trace_ctxt);
            if (rc)
            {
                fprintf(stderr, "init_backtrace_context() failed: %d\n", rc);
                return 1;
            }

            assert(trace_logger->contexts);
            trace_context_t *ptr = trace_logger->contexts;
            while (ptr->next != NULL)
                ptr = ptr->next;
            ptr->next = new_trace_ctxt;
            trace_logger->num_contexts++;
        }
    }
}

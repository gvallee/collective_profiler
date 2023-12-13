/*************************************************************************
 * Copyright (c) 2019-2010, Mellanox Technologies, Inc. All rights reserved.
 * Copyright (c) 2020-2022, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <mpi.h>

#include "allgatherv_profiler.h"
#include "logger.h"
#include "grouping.h"
#include "pattern.h"
#include "execinfo.h"
#include "timings.h"
#include "backtrace.h"
#include "location.h"
#include "buff_content.h"
#include "datatype.h"

static SRCountNode_t *counts_head = NULL;
static SRDisplNode_t *displs_head = NULL;
static TimingsNode_t *op_timing_exec_head = NULL;
static TimingsNode_t *op_timing_exec_tail = NULL;
static Pattern_t *spatterns = NULL;
static Pattern_t *rpatterns = NULL;
static avCallPattern_t *call_patterns = NULL;
// static caller_info_t *callers_head = NULL;
// static caller_info_t *callers_tail = NULL;

static int world_size = -1;
static int world_rank = -1;
static uint64_t allgathervCalls = 0;       // Total number of allgatherv calls that we went through (indexed on 0, not 1)
static uint64_t allgathervCallsLogged = 0; // Total number of allgatherv calls for which we gathered data
static uint64_t allgathervCallStart = -1;  // Number of allgatherv call during which we started to gather data
static uint64_t dump_call_data = -1;

static uint64_t _num_call_start_profiling = ALLGATHERV_NUM_CALL_START_PROFILING;
static uint64_t _limit_av_calls = DEFAULT_LIMIT_ALLGATHERV_CALLS;

#if ENABLE_LATE_ARRIVAL_TIMING
static int _inject_delay = 0;
#endif // ENABLE_LATE_ARRIVAL_TIMING

static int do_send_buffs = 0; // Specify that the focus is on send buffers rather than recv buffers
static int max_call = -1;     // Specify when to stop when checking content of buffers

// Buffers used to store data through all allgatherv calls
int *sbuf = NULL;
int *rbuf = NULL;
double *op_exec_times = NULL;
double *late_arrival_timings = NULL;

static logger_t *logger = NULL;

#if ENABLE_EXEC_TIMING
double timestamps_start[500];
double timestamps_end[500];
size_t num_timestamps = 0;
#endif // ENABLE_EXEC_TIMING

/* FORTRAN BINDINGS */
extern int mpi_fortran_in_place_;
#define OMPI_IS_FORTRAN_IN_PLACE(addr) \
    (addr == (void *)&mpi_fortran_in_place_)
extern int mpi_fortran_bottom_;
#define OMPI_IS_FORTRAN_BOTTOM(addr) \
    (addr == (void *)&mpi_fortran_bottom_)
#define OMPI_INT_2_FINT(a) a
#define OMPI_FINT_2_INT(a) a
#define OMPI_F2C_IN_PLACE(addr) (OMPI_IS_FORTRAN_IN_PLACE(addr) ? MPI_IN_PLACE : (addr))
#define OMPI_F2C_BOTTOM(addr) (OMPI_IS_FORTRAN_BOTTOM(addr) ? MPI_BOTTOM : (addr))

static int _finalize_profiling();
static int _commit_data();

static int *lookupRankRecvDispls(SRDisplNode_t *call_data, int rank)
{
    return lookup_rank_displs(call_data->recv_data_size, call_data->recv_data, rank);
}

static int *lookupRankSendCounters(SRCountNode_t *call_data, int rank)
{
    return lookup_rank_counters(call_data->send_data_size, call_data->send_data, rank);
}

static int *lookupRankRecvCounters(SRCountNode_t *call_data, int rank)
{
    return lookup_rank_counters(call_data->recv_data_size, call_data->recv_data, rank);
}

// Compare if two arrays are identical.
static bool same_call_counters(SRCountNode_t *call_data, int *send_counts, int *recv_counts, int size)
{
    int num = 0;
    int rank, count_num;

    DEBUG_ALLGATHERV_PROFILING("Comparing data with existing data...\n");
    DEBUG_ALLGATHERV_PROFILING("-> Comparing send counts...\n");
    // First compare the send counts, each rank has a single count
    for (rank = 0; rank < size; rank++)
    {
        int *_counts = lookupRankSendCounters(call_data, rank);
        assert(_counts);
        if (_counts[rank] != send_counts[num])
        {
            DEBUG_ALLGATHERV_PROFILING("Data differs\n");
            return false;
        }
        num++;
    }
    DEBUG_ALLGATHERV_PROFILING("-> Send counts are the same\n");

    // Then the receive counts
    DEBUG_ALLGATHERV_PROFILING("-> Comparing recv counts...\n");
    num = 0;
    for (rank = 0; rank < size; rank++)
    {
        int *_counts = lookupRankRecvCounters(call_data, rank);
        for (count_num = 0; count_num < size; count_num++)
        {
            if (_counts[count_num] != recv_counts[num])
            {
                DEBUG_ALLGATHERV_PROFILING("Data differs\n");
                return false;
            }
            num++;
        }
    }

    DEBUG_ALLGATHERV_PROFILING("Data is the same\n");
    return true;
}

// Compare if two arrays are identical.
static bool same_call_displs(SRDisplNode_t *call_data, int *displs, int size)
{
    int num = 0;
    int rank, displ_num;

    DEBUG_ALLGATHERV_PROFILING("Comparing data with existing data...\n");

    // Then the receive counts
    DEBUG_ALLGATHERV_PROFILING("-> Comparing recv displacements...\n");
    num = 0;
    for (rank = 0; rank < size; rank++)
    {
        int *_displs = lookupRankRecvDispls(call_data, rank);
        assert(_displs);
        for (displ_num = 0; displ_num < size; displ_num++)
        {
            if (_displs[displ_num] != displs[num])
            {
                DEBUG_ALLGATHERV_PROFILING("Data differs\n");
                return false;
            }
            num++;
        }
    }

    DEBUG_ALLGATHERV_PROFILING("Data is the same\n");
    return true;
}

static counts_data_t *lookupCounters(int size, int num, counts_data_t **list, int *count)
{
    int i, j;
    for (i = 0; i < num; i++)
    {
        for (j = 0; j < size; j++)
        {
            if (count[j] != list[i]->counters[j])
            {
                break;
            }
        }

        if (j == size)
        {
            return list[i];
        }
    }

    return NULL;
}

static displs_data_t *lookupDispls(int size, int num, displs_data_t **list, int *displs)
{
    int i, j;
    for (i = 0; i < num; i++)
    {
        for (j = 0; j < size; j++)
        {
            if (displs[j] != list[i]->displs[j])
            {
                break;
            }
        }

        if (j == size)
        {
            return list[i];
        }
    }

    return NULL;
}

static int extract_patterns_from_counts(int *send_counts, int *recv_counts, int size)
{
    int i, j, num;
    int src_ranks = 0;
    int dst_ranks = 0;
    int send_patterns[size + 1];
    int recv_patterns[size + 1];

    DEBUG_ALLGATHERV_PROFILING("Extracting patterns\n");

    for (i = 0; i < size; i++)
    {
        send_patterns[i] = 0;
    }

    for (i = 0; i < size; i++)
    {
        recv_patterns[i] = 0;
    }

    num = 0;
    for (i = 0; i < size; i++)
    {
        dst_ranks = 0;
        src_ranks = 0;
        for (j = 0; j < size; j++)
        {
            if (send_counts[num] != 0)
            {
                dst_ranks++;
            }
            if (recv_counts[num] != 0)
            {
                src_ranks++;
            }
            num++;
        }
        // We know the current rank sends data to <dst_ranks> ranks
        if (dst_ranks > 0)
        {
            send_patterns[dst_ranks - 1]++;
        }

        // We know the current rank receives data from <src_ranks> ranks
        if (src_ranks > 0)
        {
            recv_patterns[src_ranks - 1]++;
        }
    }

    // From here we know who many ranks send to how many ranks and how many ranks receive from how many rank
    DEBUG_ALLGATHERV_PROFILING("Handling send patterns\n");
    for (i = 0; i < size; i++)
    {
        if (send_patterns[i] != 0)
        {
            DEBUG_ALLGATHERV_PROFILING("Add pattern where %d ranks sent data to %d other ranks\n", send_patterns[i], i + 1);
#if COMMSIZE_BASED_PATTERNS
            spatterns = add_pattern_for_size(spatterns, send_patterns[i], i + 1, size);
#else
            spatterns = add_pattern(spatterns, send_patterns[i], i + 1);
#endif // COMMSIZE_BASED_PATTERNS
        }
    }
    DEBUG_ALLGATHERV_PROFILING("Handling receive patterns\n");
    for (i = 0; i < size; i++)
    {
        if (recv_patterns[i] != 0)
        {
            DEBUG_ALLGATHERV_PROFILING("Add pattern where %d ranks received data from %d other ranks\n", recv_patterns[i], i + 1);
#if COMMSIZE_BASED_PATTERNS
            rpatterns = add_pattern_for_size(rpatterns, recv_patterns[i], i + 1, size);
#else
            rpatterns = add_pattern(rpatterns, recv_patterns[i], i + 1);
#endif // COMMSIZE_BASED_PATTERNS
        }
    }

    return 0;
}

char *allgatherv_get_full_filename(int ctxt, char *id, int jobid, int world_rank)
{
    char *filename = NULL;
    int size;
    char *dir = get_output_dir();

    if (ctxt == MAIN_CTX)
    {
        if (id == NULL)
        {
            _asprintf(filename, size, "profile_allgatherv_job%d.rank%d.md", jobid, world_rank);
            assert(size > 0);
        }
        else
        {
            _asprintf(filename, size, "%s.job%d.rank%d.md", id, jobid, world_rank);
            assert(size > 0);
        }
    }
    else
    {
        char *context = ctx_to_string(ctxt);
        _asprintf(filename, size, "%s-%s.job%d.rank%d.txt", context, id, jobid, world_rank);
        assert(size > 0);
    }
    assert(filename);

    if (dir != NULL)
    {
        char *path = NULL;
        _asprintf(path, size, "%s/%s", dir, filename);
        assert(size > 0);
        free(filename);
        return path;
    }

    return filename;
}

int extract_call_patterns_from_counts(int callID, int *send_counts, int *recv_counts, int size)
{
    avCallPattern_t *cp = extract_call_patterns(callID, send_counts, recv_counts, size);
    if (call_patterns == NULL)
    {
        call_patterns = cp;
    }
    else
    {
        avCallPattern_t *existing_cp = lookup_call_patterns(call_patterns);
        if (existing_cp == NULL)
        {
            avCallPattern_t *ptr = call_patterns;
            while (ptr->next != NULL)
            {
                ptr = ptr->next;
            }
            ptr->next = cp;
        }
        else
        {
            existing_cp->n_calls++;
            free_patterns(cp->spatterns);
            free_patterns(cp->rpatterns);
            free(cp);
        }
    }

    return 0;
}

#if ENABLE_PATTERN_DETECTION && TRACK_PATTERNS_ON_CALL_BASIS
static int commit_pattern_from_counts(int callID, int *send_counts, int *recv_counts, int size)
{
    return extract_call_patterns_from_counts(callID, send_counts, recv_counts, size);
}
#endif
#if ENABLE_PATTERN_DETECTION && !TRACK_PATTERNS_ON_CALL_BASIS
static int commit_pattern_from_counts(int callID, int *send_counts, int *recv_counts, int size)
{
    return extract_patterns_from_counts(send_counts, recv_counts, size);
}
#endif

static int commit_pattern_from_counts(int callID, int *send_counts, int *recv_counts, int size)
{
#if TRACK_PATTERNS_ON_CALL_BASIS
    return extract_call_patterns_from_counts(callID, send_counts, recv_counts, size);
#else
    return extract_patterns_from_counts(send_counts, recv_counts, size);
#endif
}

static displs_data_t *lookupRecvDispls(int *counts, SRDisplNode_t *call_data)
{
    int num = 1;
    return lookupDispls(num, call_data->recv_data_size, call_data->recv_data, counts);
}

static counts_data_t *lookupSendCounters(int *counts, SRCountNode_t *call_data)
{
    int num_counts = 1;
    return lookupCounters(num_counts, call_data->send_data_size, call_data->send_data, counts);
}

static counts_data_t *lookupRecvCounters(int *counts, SRCountNode_t *call_data)
{
    return lookupCounters(call_data->size, call_data->recv_data_size, call_data->recv_data, counts);
}

static int add_rank_to_counters_data(int rank, counts_data_t *counters_data)
{
    if (counters_data->num_ranks >= counters_data->max_ranks)
    {
        counters_data->max_ranks = counters_data->num_ranks + MAX_TRACKED_RANKS;
        counters_data->ranks = (int *)realloc(counters_data->ranks, counters_data->max_ranks * sizeof(int));
        assert(counters_data->ranks);
    }

    counters_data->ranks[counters_data->num_ranks] = rank;
    counters_data->num_ranks++;
    return 0;
}

static void delete_counter_data(counts_data_t **data)
{
    if (*data)
    {
        if ((*data)->ranks)
        {
            free((*data)->ranks);
        }
        if ((*data)->counters)
        {
            free((*data)->counters);
        }
        free(*data);
        *data = NULL;
    }
}

static counts_data_t *new_counter_data(int size, int rank, int *counts)
{
    int i;
    counts_data_t *new_data = (counts_data_t *)malloc(sizeof(counts_data_t));
    assert(new_data);
    new_data->counters = (int *)malloc(size * sizeof(int));
    assert(new_data->counters);
    new_data->num_ranks = 0;
    new_data->max_ranks = MAX_TRACKED_RANKS;
    new_data->ranks = (int *)malloc(new_data->max_ranks * sizeof(int));
    assert(new_data->ranks);

    for (i = 0; i < size; i++)
    {
        new_data->counters[i] = counts[i];
    }
    new_data->ranks[new_data->num_ranks] = rank;
    new_data->num_ranks++;

    return new_data;
}

static int add_new_send_counters_to_counters_data(SRCountNode_t *call_data, int rank, int *counts)
{
    counts_data_t *new_data = new_counter_data(1 /*call_data->size*/, rank, counts);
    call_data->send_data[call_data->send_data_size] = new_data;
    call_data->send_data_size++;

    return 0;
}

static int add_new_recv_counters_to_counters_data(SRCountNode_t *call_data, int rank, int *counts)
{
    counts_data_t *new_data = new_counter_data(call_data->size, rank, counts);
    call_data->recv_data[call_data->recv_data_size] = new_data;
    call_data->recv_data_size++;

    return 0;
}

static int compareAndSaveSendCounters(int rank, int *counts, SRCountNode_t *call_data)
{
    counts_data_t *ptr = lookupSendCounters(counts, call_data);
    if (ptr)
    {
        DEBUG_ALLGATHERV_PROFILING("Add send rank %d to existing count data\n", rank);
        if (add_rank_to_counters_data(rank, ptr))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to add rank counters (rank: %d)\n", __FILE__, __LINE__, rank);
            return -1;
        }
    }
    else
    {
        DEBUG_ALLGATHERV_PROFILING("Add send new count data for rank %d\n", rank);
        if (add_new_send_counters_to_counters_data(call_data, rank, counts))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to add new send counters\n", __FILE__, __LINE__);
            return -1;
        }
    }

    return 0;
}

static int compareAndSaveRecvCounters(int rank, int *counts, SRCountNode_t *call_data)
{
    counts_data_t *ptr = lookupRecvCounters(counts, call_data);
    if (ptr)
    {
        DEBUG_ALLGATHERV_PROFILING("Add recv rank %d to existing count data\n", rank);
        if (add_rank_to_counters_data(rank, ptr))
        {
            fprintf(stderr, "[ERROR] unable to add rank counters\n");
            return -1;
        }
    }
    else
    {
        DEBUG_ALLGATHERV_PROFILING("Add recv new count data for rank %d\n", rank);
        if (add_new_recv_counters_to_counters_data(call_data, rank, counts))
        {
            fprintf(stderr, "[ERROR] unable to add new recv counters\n");
            return -1;
        }
    }

    return 0;
}

static int add_rank_to_displs_data(int rank, displs_data_t *displs_data)
{
    if (displs_data->num_ranks >= displs_data->max_ranks)
    {
        displs_data->max_ranks = displs_data->num_ranks + MAX_TRACKED_RANKS;
        displs_data->ranks = (int *)realloc(displs_data->ranks, displs_data->max_ranks * sizeof(int));
        assert(displs_data->ranks);
    }

    displs_data->ranks[displs_data->num_ranks] = rank;
    displs_data->num_ranks++;
    return 0;
}

static displs_data_t *new_displ_data(int size, int rank, int *displs)
{
    int i;
    displs_data_t *new_data = NULL;
    
    assert(displs);
    new_data = (displs_data_t *)malloc(sizeof(displs_data_t));
    assert(new_data);
    new_data->displs = (int *)malloc(size * sizeof(int));
    assert(new_data->displs);
    new_data->num_ranks = 0;
    new_data->max_ranks = MAX_TRACKED_RANKS;
    new_data->ranks = (int *)malloc(new_data->max_ranks * sizeof(int));
    assert(new_data->ranks);

    for (i = 0; i < size; i++)
    {
        new_data->displs[i] = displs[i];
    }
    new_data->ranks[new_data->num_ranks] = rank;
    new_data->num_ranks++;

    return new_data;
}

static int add_new_recv_displs_to_displs_data(SRDisplNode_t *call_data, int rank, int *displs)
{
    displs_data_t *new_data = new_displ_data(call_data->size, rank, displs);
    call_data->recv_data[call_data->recv_data_size] = new_data;
    call_data->recv_data_size++;

    return 0;
}

static int compareAndSaveRecvDispls(int rank, int *displs, SRDisplNode_t *call_data)
{
    displs_data_t *ptr = lookupRecvDispls(displs, call_data);
    if (ptr)
    {
        DEBUG_ALLGATHERV_PROFILING("Add recv rank %d to existing displacements data\n", rank);
        if (add_rank_to_displs_data(rank, ptr))
        {
            fprintf(stderr, "[ERROR] unable to add rank displacements\n");
            return -1;
        }
    }
    else
    {
        DEBUG_ALLGATHERV_PROFILING("Add recv new count data for rank %d\n", rank);
        if (add_new_recv_displs_to_displs_data(call_data, rank, displs))
        {
            fprintf(stderr, "[ERROR] unable to add new recv displacements\n");
            return -1;
        }
    }

    return 0;
}

// Compare new recv displacement data with existing data.
// If there is a match, increase the counter. Add new data, otherwise.
static int insert_displ_data(int *rbuf, int size, int sendtype_size, int recvtype_size)
{
    int num = 0;
    struct SRDisplNode *newNode = NULL;
    struct SRDisplNode *temp;

    DEBUG_ALLGATHERV_PROFILING("Insert displacement data for a new allgatherv call...\n");

    assert(rbuf);
    assert(logger);

    temp = displs_head;
    while (temp != NULL)
    {
        if (temp->size != size || temp->recvtype_size != recvtype_size || temp->sendtype_size != sendtype_size || !same_call_displs(temp, rbuf, size))
        {
            // New data
#if DEBUG
            fprintf(logger->f, "new data: %d\n", size);
#endif
            if (temp->next != NULL)
                temp = temp->next;
            else
                break;
        }
        else
        {
            // Data exist, adding call info to it
            DEBUG_ALLGATHERV_PROFILING("Displacement data already exists, updating metadata...\n");
            assert(temp->list_calls);
            if (temp->count >= temp->max_calls)
            {
                temp->max_calls = temp->max_calls * 2;
                temp->list_calls = (uint64_t *)realloc(temp->list_calls, temp->max_calls * sizeof(uint64_t));
                assert(temp->list_calls);
            }
            temp->list_calls[temp->count] = allgathervCalls; // Note: count starts at 1, not 0
            temp->count++;
#if DEBUG
            fprintf(logger->f, "old data: %d --> %d --- %d\n", size, temp->size, temp->count);
#endif
            DEBUG_ALLGATHERV_PROFILING("Metadata successfully updated\n");
            return 0;
        }
    }

#if DEBUG
    fprintf(logger->f, "no data: %d \n", size);
#endif
    newNode = (struct SRDisplNode *)malloc(sizeof(SRDisplNode_t));
    assert(newNode);

    newNode->size = size;
    newNode->rank_send_vec_len = 1; // 1 send count per rank
    newNode->rank_recv_vec_len = size; // communicator size counts per rank
    newNode->count = 1;
    newNode->list_calls = (uint64_t *)malloc(DEFAULT_TRACKED_CALLS * sizeof(uint64_t));
    assert(newNode->list_calls);
    newNode->max_calls = DEFAULT_TRACKED_CALLS;
    // We have at most <size> different counts (one per rank) and we just allocate pointers of pointers here, not much space used
    newNode->send_data = (displs_data_t **)malloc(size * sizeof(displs_data_t));
    assert(newNode->send_data);
    newNode->send_data_size = 0;
    newNode->recv_data = (displs_data_t **)malloc(size * sizeof(displs_data_t));
    assert(newNode->recv_data);
    newNode->recv_data_size = 0;

    // We add rank's data one by one so we can compress the data when possible
    num = 0;
    int _rank;
    for (_rank = 0; _rank < size; _rank++)
    {
        if (compareAndSaveRecvDispls(_rank, &(rbuf[num * size]), newNode))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to add recv displacements\n", __FILE__, __LINE__);
            return -1;
        }
        num++;
    }

    newNode->sendtype_size = sendtype_size;
    newNode->recvtype_size = recvtype_size;
    newNode->list_calls[0] = allgathervCalls;
    newNode->next = NULL;
#if DEBUG
    fprintf(logger->f, "new entry: %d --> %d --- %d\n", size, newNode->size, newNode->count);
#endif

    DEBUG_ALLGATHERV_PROFILING("Data for the new allgatherv call has %d unique series for recv displs\n", newNode->recv_data_size, newNode->send_data_size);

    if (displs_head == NULL)
    {
        displs_head = newNode;
    }
    else
    {
        temp->next = newNode;
    }

    return 0;
}


// Compare new send count data with existing data.
// If there is a match, increase the counter. Add new data, otherwise.
static int insert_sendrecv_count_data(int *sbuf, int *rbuf, int size, int sendtype_size, int recvtype_size)
{
    int num = 0;
    struct SRCountNode *newNode = NULL;
    struct SRCountNode *temp;

    DEBUG_ALLGATHERV_PROFILING("Insert data for a new allgatherv call...\n");

    assert(sbuf);
    assert(rbuf);
    assert(logger);

    temp = counts_head;
    while (temp != NULL)
    {
        if (temp->size != size || temp->recvtype_size != recvtype_size || temp->sendtype_size != sendtype_size || !same_call_counters(temp, sbuf, rbuf, size))
        {
            // New data
#if DEBUG
            fprintf(logger->f, "new data: %d\n", size);
#endif
            if (temp->next != NULL)
                temp = temp->next;
            else
                break;
        }
        else
        {
            // Data exist, adding call info to it
            DEBUG_ALLGATHERV_PROFILING("Data already exists, updating metadata...\n");
            assert(temp->list_calls);
            if (temp->count >= temp->max_calls)
            {
                temp->max_calls = temp->max_calls * 2;
                temp->list_calls = (uint64_t *)realloc(temp->list_calls, temp->max_calls * sizeof(uint64_t));
                assert(temp->list_calls);
            }
            temp->list_calls[temp->count] = allgathervCalls; // Note: count starts at 1, not 0
            temp->count++;
#if DEBUG
            fprintf(logger->f, "old data: %d --> %d --- %d\n", size, temp->size, temp->count);
#endif
            DEBUG_ALLGATHERV_PROFILING("Metadata successfully updated\n");
            return 0;
        }
    }

#if DEBUG
    fprintf(logger->f, "no data: %d \n", size);
#endif
    newNode = (struct SRCountNode *)malloc(sizeof(SRCountNode_t));
    assert(newNode);

    newNode->size = size;
    newNode->rank_send_vec_len = 1; // 1 send count per rank
    newNode->rank_recv_vec_len = size; // communicator size counts per rank
    newNode->count = 1;
    newNode->list_calls = (uint64_t *)malloc(DEFAULT_TRACKED_CALLS * sizeof(uint64_t));
    assert(newNode->list_calls);
    newNode->max_calls = DEFAULT_TRACKED_CALLS;
    // We have at most <size> different counts (one per rank) and we just allocate pointers of pointers here, not much space used
    newNode->send_data = (counts_data_t **)malloc(size * sizeof(counts_data_t));
    assert(newNode->send_data);
    newNode->send_data_size = 0;
    newNode->recv_data = (counts_data_t **)malloc(size * sizeof(counts_data_t));
    assert(newNode->recv_data);
    newNode->recv_data_size = 0;

    // We add rank's data one by one so we can compress the data when possible
    num = 0;
    int _rank = 0;

    DEBUG_ALLGATHERV_PROFILING("handling send counts...\n");
    for (_rank = 0; _rank < size; _rank++)
    {
        if (compareAndSaveSendCounters(_rank, &(sbuf[_rank]), newNode))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to add send counters\n", __FILE__, __LINE__);
            return -1;
        }
        num++;
    }

    DEBUG_ALLGATHERV_PROFILING("handling recv counts...\n");
    num = 0;
    for (_rank = 0; _rank < size; _rank++)
    {
        if (compareAndSaveRecvCounters(_rank, &(rbuf[num * size]), newNode))
        {
            fprintf(stderr, "[%s:%d][ERROR] unable to add recv counters\n", __FILE__, __LINE__);
            return -1;
        }
        num++;
    }

    newNode->sendtype_size = sendtype_size;
    newNode->recvtype_size = recvtype_size;
    newNode->list_calls[0] = allgathervCalls;
    newNode->next = NULL;
#if DEBUG
    fprintf(logger->f, "new entry: %d --> %d --- %d\n", size, newNode->size, newNode->count);
#endif

    DEBUG_ALLGATHERV_PROFILING("Data for the new allgatherv call has %d unique series for send counts and %d for recv counts\n", newNode->recv_data_size, newNode->send_data_size);

    if (counts_head == NULL)
    {
        counts_head = newNode;
    }
    else
    {
        temp->next = newNode;
    }

    return 0;
}

int _mpi_init(int *argc, char ***argv)
{
    int ret;
    char *num_call_envvar = getenv(NUM_CALL_START_PROFILING_ENVVAR);
    if (num_call_envvar != NULL)
    {
        _num_call_start_profiling = atoi(num_call_envvar);
    }

    char *limit_a2a_calls = getenv(ALLGATHERV_LIMIT_CALLS_ENVVAR);
    if (limit_a2a_calls != NULL)
    {
        _limit_av_calls = atoi(limit_a2a_calls);
    }

    ret = PMPI_Init(argc, argv);

    PMPI_Comm_rank(MPI_COMM_WORLD, &world_rank);
    PMPI_Comm_size(MPI_COMM_WORLD, &world_size);

    // We do not know what rank will gather allgatherv data since allgatherv can
    // be called on any communicator
    int jobid = get_job_id();
    logger_config_t allgatherv_logger_cfg;
    allgatherv_logger_cfg.get_full_filename = &allgatherv_get_full_filename;
    allgatherv_logger_cfg.collective_name = "Allgatherv";
    allgatherv_logger_cfg.limit_number_calls = DEFAULT_LIMIT_ALLGATHERV_CALLS;
    logger = logger_init(jobid, world_rank, world_size, &allgatherv_logger_cfg);
    assert(logger);

    // Allocate buffers reused between allgatherv calls
    // Note the buffer may be used on a communicator that is not comm_world
    // but in any case, it will be smaller or of the same size than comm_world.
    // So we allocate the biggest buffers possible but reuse them during the
    // entire execution of the application.
    sbuf = (int *)malloc(world_size * (sizeof(int)));
    assert(sbuf);
    rbuf = (int *)malloc(world_size * world_size * (sizeof(int)));
    assert(rbuf);
#if ENABLE_EXEC_TIMING
    op_exec_times = (double *)malloc(world_size * sizeof(double));
    assert(op_exec_times);
#endif // ENABLE_EXEC_TIMING
#if ENABLE_LATE_ARRIVAL_TIMING
    late_arrival_timings = (double *)malloc(world_size * sizeof(double));
    assert(late_arrival_timings);
    char *inject_delay = getenv("COLLECTIVE_PROFILER_INJECT_DELAY");
    if (inject_delay != NULL)
    {
        _inject_delay = atoi(inject_delay);
    }
#endif // ENABLE_LATE_ARRIVAL_TIMING

#if ENABLE_VALIDATION
    srand((unsigned)getpid());
#endif

    char *buff_type = getenv(COLLECTIVE_PROFILER_CHECK_SEND_BUFF_ENVVAR);
    if (buff_type != NULL)
    {
        do_send_buffs = atoi(buff_type);
    }

    char *max_call_num_envvar = getenv(COLLECTIVE_PROFILER_MAX_CALL_CHECK_BUFF_CONTENT_ENVVAR);
    if (max_call_num_envvar != NULL)
    {
        max_call = atoi(max_call_num_envvar);
    }

    char *dump_call_data_envvar = getenv("DUMP_CALL_DATA");
    if (dump_call_data_envvar != NULL)
        dump_call_data = atoi(dump_call_data_envvar);

    // Make sure we do not create an articial imbalance between ranks.
    PMPI_Barrier(MPI_COMM_WORLD);

    return ret;
}

int _mpi_init_thread(int *argc, char ***argv, int required, int *provided)
{
    int ret;
    char *num_call_envvar = getenv(NUM_CALL_START_PROFILING_ENVVAR);
    if (num_call_envvar != NULL)
    {
        _num_call_start_profiling = atoi(num_call_envvar);
    }

    char *limit_a2a_calls = getenv(ALLGATHERV_LIMIT_CALLS_ENVVAR);
    if (limit_a2a_calls != NULL)
    {
        _limit_av_calls = atoi(limit_a2a_calls);
    }

    ret = PMPI_Init_thread(argc, argv, required, provided);

    PMPI_Comm_rank(MPI_COMM_WORLD, &world_rank);
    PMPI_Comm_size(MPI_COMM_WORLD, &world_size);

    // We do not know what rank will gather allgatherv data since allgatherv can
    // be called on any communicator
    int jobid = get_job_id();
    logger_config_t allgatherv_logger_cfg;
    allgatherv_logger_cfg.get_full_filename = &allgatherv_get_full_filename;
    allgatherv_logger_cfg.collective_name = "Alltoallv";
    allgatherv_logger_cfg.limit_number_calls = DEFAULT_LIMIT_ALLGATHERV_CALLS;
    logger = logger_init(jobid, world_rank, world_size, &allgatherv_logger_cfg);
    assert(logger);

    // Allocate buffers reused between allgatherv calls
    // Note the buffer may be used on a communicator that is not comm_world
    // but in any case, it will be smaller or of the same size than comm_world.
    // So we allocate the biggest buffers possible but reuse them during the
    // entire execution of the application.
    sbuf = (int *)malloc(world_size * (sizeof(int)));
    assert(sbuf);
    rbuf = (int *)malloc(world_size * world_size * (sizeof(int)));
    assert(rbuf);
#if ENABLE_EXEC_TIMING
    op_exec_times = (double *)malloc(world_size * sizeof(double));
    assert(op_exec_times);
#endif // ENABLE_EXEC_TIMING
#if ENABLE_LATE_ARRIVAL_TIMING
    late_arrival_timings = (double *)malloc(world_size * sizeof(double));
    assert(late_arrival_timings);
    char *inject_delay = getenv("COLLECTIVE_PROFILER_INJECT_DELAY");
    if (inject_delay != NULL)
    {
        _inject_delay = atoi(inject_delay);
    }
#endif // ENABLE_LATE_ARRIVAL_TIMING

#if ENABLE_VALIDATION
    srand((unsigned)getpid());
#endif

    char *buff_type = getenv(COLLECTIVE_PROFILER_CHECK_SEND_BUFF_ENVVAR);
    if (buff_type != NULL)
    {
        do_send_buffs = atoi(buff_type);
    }

    char *max_call_num_envvar = getenv(COLLECTIVE_PROFILER_MAX_CALL_CHECK_BUFF_CONTENT_ENVVAR);
    if (max_call_num_envvar != NULL)
    {
        max_call = atoi(max_call_num_envvar);
    }

    char *dump_call_data_envvar = getenv("DUMP_CALL_DATA");
    if (dump_call_data_envvar != NULL)
        dump_call_data = atoi(dump_call_data_envvar);

    // Make sure we do not create an articial imbalance between ranks.
    PMPI_Barrier(MPI_COMM_WORLD);

    return ret;
}

int MPI_Finalize()
{
    _commit_data();
    _finalize_profiling();
    return PMPI_Finalize();
}

int MPI_Init_thread(int *argc, char ***argv, int required, int *provided)
{
    return _mpi_init_thread(argc, argv, required, provided);
}

int MPI_Init(int *argc, char ***argv)
{
    return _mpi_init(argc, argv);
}

int mpi_init_thread_(MPI_Fint *required, MPI_Fint *provided, MPI_Fint *ierr)
{
    int c_ierr;
    int argc = 0;
    char **argv = NULL;
    int c_provided;

    c_ierr = _mpi_init_thread(&argc, &argv, OMPI_FINT_2_INT(*required), &c_provided);
    if (NULL != ierr)
        *ierr = OMPI_INT_2_FINT(c_ierr);
    if (MPI_SUCCESS == c_ierr)
    {
        *provided = OMPI_INT_2_FINT(c_provided);
    }
    return 0;
}

int mpi_init_(MPI_Fint *ierr)
{
    int c_ierr;
    int argc = 0;
    char **argv = NULL;

    c_ierr = _mpi_init(&argc, &argv);
    if (NULL != ierr)
        *ierr = OMPI_INT_2_FINT(c_ierr);
    return 0;
}

#if ENABLE_DISPLS
static int _release_displs_resources()
{
    // All data has been handled, now we can clean up
    int i;
    while (displs_head != NULL)
    {
        SRDisplNode_t *c_ptr = displs_head->next;

        for (i = 0; i < displs_head->send_data_size; i++)
        {
            delete_counter_data(&(displs_head->send_data[i]));
        }

        for (i = 0; i < displs_head->recv_data_size; i++)
        {
            delete_counter_data(&(displs_head->recv_data[i]));
        }

        free(displs_head->recv_data);
        free(displs_head->send_data);
        free(displs_head->list_calls);

        free(displs_head);
        displs_head = c_ptr;
    }
    return 0;
}
#endif // ENABLE_DISPLS

#if ENABLE_RAW_DATA || ENABLE_VALIDATION
static int _release_counts_resources()
{
    // All data has been handled, now we can clean up
    int i;
    while (counts_head != NULL)
    {
        SRCountNode_t *c_ptr = counts_head->next;

        for (i = 0; i < counts_head->send_data_size; i++)
        {
            delete_counter_data(&(counts_head->send_data[i]));
        }

        for (i = 0; i < counts_head->recv_data_size; i++)
        {
            delete_counter_data(&(counts_head->recv_data[i]));
        }

        free(counts_head->recv_data);
        free(counts_head->send_data);
        free(counts_head->list_calls);

        free(counts_head);
        counts_head = c_ptr;
    }
    return 0;
}
#endif // ENABLE_RAW_DATA || ENABLE_VALIDATION

static int _release_pattern_resources()
{
    while (rpatterns != NULL)
    {
        avPattern_t *rp = rpatterns->next;
        free(rpatterns);
        rpatterns = rp;
    }

    while (spatterns != NULL)
    {
        avPattern_t *sp = spatterns->next;
        free(spatterns);
        spatterns = sp;
    }

    return 0;
}

static int _release_profiling_resources()
{
#if ENABLE_RAW_DATA || ENABLE_VALIDATION
    _release_counts_resources();
#endif // ENABLE_RAW_DATA || ENABLE_VALIDATION

#if ENABLE_DISPLS
    _release_displs_resources();
#endif // ENABLE_DISPLS

    while (op_timing_exec_head != NULL)
    {
        avTimingsNode_t *t_ptr = op_timing_exec_head->next;
        free(op_timing_exec_head->timings);
        free(op_timing_exec_head);
        op_timing_exec_head = t_ptr;
    }
    op_timing_exec_tail = NULL;

    _release_pattern_resources();

    // Free all the memory allocated during MPI_Init() for profiling purposes
    if (rbuf != NULL)
    {
        free(rbuf);
        rbuf = NULL;
    }
    if (sbuf != NULL)
    {
        free(sbuf);
        sbuf = NULL;
    }
    if (op_exec_times != NULL)
    {
        free(op_exec_times);
        op_exec_times = NULL;
    }
    if (late_arrival_timings != NULL)
    {
        free(late_arrival_timings);
        late_arrival_timings = NULL;
    }
#if 0
		if (hostnames)
		{
			free(hostnames);
		}
#endif
    return 0;
}

static int _finalize_profiling()
{
    logger_fini(&logger);
    _release_profiling_resources();
    return 0;
}

static int _commit_data()
{
    log_profiling_data(logger, allgathervCalls, allgathervCallStart, allgathervCallsLogged, counts_head, displs_head, op_timing_exec_head);

/*
#if ENABLE_TIMING
    //log_timing_data(logger, op_timing_exec_head);
#endif // ENABLE_TIMING
*/

#if ENABLE_EXEC_TIMING
    /* Save start & end timestamps */
    {
        int ret, rc;
        size_t i;
        char *filename = NULL;
        if (getenv(OUTPUT_DIR_ENVVAR))
        {
            _asprintf(filename, rc, "%s/timestamps.rank%d.md", getenv(OUTPUT_DIR_ENVVAR), logger->rank);
        }
        else
        {
            _asprintf(filename, rc, "timestamps.rank%d.md", logger->rank);
        }
        assert(rc > 0);

        FILE *f = fopen(filename, "w");
        assert(f);
        
        for (i = 0; i < num_timestamps; i++)
        {
            fprintf(f, "%lf %lf\n", timestamps_start[i], timestamps_end[i]);
        }

        fclose(f);
    }
#endif // ENABLE_EXEC_TIMING

#if ENABLE_PATTERN_DETECTION && !TRACK_PATTERNS_ON_CALL_BASIS
    save_patterns(world_rank);
#endif // ENABLE_PATTERN_DETECTION && !TRACK_PATTERNS_ON_CALL_BASIS

#if ENABLE_PATTERN_DETECTION && TRACK_PATTERNS_ON_CALL_BASIS
    save_call_patterns(world_rank);
#endif // ENABLE_PATTERN_DETECTION && TRACK_PATTERNS_ON_CALL_BASIS

    return 0;
}

#if ((ENABLE_RAW_DATA || ENABLE_PER_RANK_STATS || ENABLE_VALIDATION) && !ENABLE_COMPACT_FORMAT)
static void save_counts(int *sendcount, int *recvcounts, int s_datatype_size, int r_datatype_size, int comm_size, uint64_t n_call)
{
    char *filename = NULL;
    int i;
    int rc;

    if (getenv(OUTPUT_DIR_ENVVAR))
    {
        _asprintf(filename, rc, "%s/counts.rank%d_call%" PRIu64 ".md", getenv(OUTPUT_DIR_ENVVAR), world_rank, n_call);
    }
    else
    {
        _asprintf(filename, rc, "counts.rank%d_call%" PRIu64 ".md", world_rank, n_call);
    }
    assert(rc > 0);

    FILE *f = fopen(filename, "w");
    assert(f);

    fprintf(f, "Send datatype size: %d\n", s_datatype_size);
    fprintf(f, "Recv datatype size: %d\n", r_datatype_size);
    fprintf(f, "Comm size: %d\n\n", comm_size);

    int idx = 0;
    fprintf(f, "Send counts\n");
    for (i = 0; i < comm_size; i++)
    {
        int j;
        for (j = 0; j < comm_size; j++)
        {
            fprintf(f, "%d ", sendcount[idx]);
            idx++;
        }
        fprintf(f, "\n");
    }

    fprintf(f, "\n\nRecv counts\n");
    idx = 0;
    for (i = 0; i < comm_size; i++)
    {
        int j;
        for (j = 0; j < comm_size; j++)
        {
            fprintf(f, "%d ", recvcounts[idx]);
            idx++;
        }
        fprintf(f, "\n");
    }

    fclose(f);
    free(filename);
}
#endif // ((ENABLE_RAW_DATA || ENABLE_PER_RANK_STATS || ENABLE_VALIDATION) && !ENABLE_COMPACT_FORMAT)

static inline void
allgatherv_save_buf_content(void *buf, const int count, MPI_Datatype type, MPI_Comm comm, int rank, char *ctxt)
{
    char *filename = NULL;
    int rc;

    assert(buf);
    assert(ctxt);

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

    // We assume the data is contiguous and that the type is of a type compatible with a C double
    double *ptr = (double *)buf;
    int j;
    for (j = 0; j < count; j++)
    {
        fprintf(f, "%f ", ptr[j]);
    }
    fprintf(f, "\n");

    fclose(f);
    free(filename);
}

int _mpi_allgatherv(const void *sendbuf, const int sendcount, MPI_Datatype sendtype,
                    void *recvbuf, const int *recvcounts, const int *rdispls, MPI_Datatype recvtype,
                    MPI_Comm comm)
{
    int comm_size;
    int ret;
    bool need_profile = true;
    int my_comm_rank;
    char *collective_name = "allgatherv";

    PMPI_Comm_size(comm, &comm_size);
    PMPI_Comm_rank(comm, &my_comm_rank);
    PMPI_Comm_rank(MPI_COMM_WORLD, &world_rank);

#if ENABLE_BACKTRACE
    if (my_comm_rank == 0)
    {
        void *array[16];
        size_t _s;
        char **strings;
        char *caller_trace = NULL;

        _s = backtrace(array, 16);
        strings = backtrace_symbols(array, _s);
        insert_caller_data(collective_name, strings, _s, comm, my_comm_rank, world_rank, allgathervCalls);
    }
#endif // ENABLE_BACKTRACE

    // Check if we need to profile that specific call
    if (allgathervCalls < _num_call_start_profiling)
    {
        need_profile = false;
    }
    else
    {
        if (-1 != _limit_av_calls && allgathervCallsLogged >= _limit_av_calls)
        {
            need_profile = false;
        }
    }

    if (need_profile)
    {
        if (allgathervCallStart == -1)
        {
            allgathervCallStart = allgathervCalls;
        }

        if (dump_call_data == allgathervCalls)
        {
            // Save datatypes information
            if (my_comm_rank == 0)
            {
                datatype_info_t sendtype_info;
                sendtype_info.analyzed = false;
                analyze_datatype(sendtype, &sendtype_info);
                int rc = save_datatype_info(collective_name, comm, my_comm_rank, world_rank, allgathervCalls, "send", &sendtype_info);
                if (rc)
                {
                    fprintf(stderr, "save_datatype_info() failed (rc: %d)\n", rc);
                    MPI_Abort(MPI_COMM_WORLD, 12);
                }

                datatype_info_t recvtype_info;
                recvtype_info.analyzed = false;
                analyze_datatype(recvtype, &recvtype_info);
                rc = save_datatype_info(collective_name, comm, my_comm_rank, world_rank, allgathervCalls, "recv", &recvtype_info);
                if (rc)
                {
                    fprintf(stderr, "save_datatype_info() failed (rc: %d)\n", rc);
                    MPI_Abort(MPI_COMM_WORLD, 13);
                }
            }

            int rc = store_call_data_single_count(collective_name, SEND_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)sendbuf, sendcount, sendtype);
            if (rc)
            {
                fprintf(stderr, "store_call_data_single_count() failed on l.%d: %d\n", __LINE__, rc);
                MPI_Abort(MPI_COMM_WORLD, 11);
            }
            allgatherv_save_buf_content((void *)sendbuf, sendcount, sendtype, comm, world_rank, "send");
        }

#if ENABLE_LATE_ARRIVAL_TIMING
        if (_inject_delay == 1 && my_comm_rank == 0)
        {
            sleep(1);
        }
        double t_barrier_start = MPI_Wtime();
        PMPI_Barrier(comm);
        double t_barrier_end = MPI_Wtime();
#endif // ENABLE_LATE_ARRIVAL_TIMING

#if ENABLE_EXEC_TIMING
        double t_start = MPI_Wtime();
        if (num_timestamps < 500)
        {
            timestamps_start[num_timestamps] = t_start;
        }
#endif // ENABLE_EXEC_TIMING

        ret = PMPI_Allgatherv(sendbuf, sendcount, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);

        if (dump_call_data == allgathervCalls)
        {
            int rc = store_call_data(collective_name, RECV_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)recvbuf, (int *)recvcounts, (int *)rdispls, recvtype);
            if (rc)
            {
                fprintf(stderr, "store_call_data() failed on l.%d: %d\n", __LINE__, rc);
                MPI_Abort(MPI_COMM_WORLD, 11);
            }
            save_buf_content(recvbuf, recvcounts, rdispls, recvtype, comm, world_rank, "recv");
            release_buffcontent_loggers();
            PMPI_Barrier(comm);
            if (my_comm_rank == 0)
                fprintf(stderr, "All data acquired, aborting...\n");
            MPI_Abort(MPI_COMM_WORLD, 22);
        }

#if ENABLE_EXEC_TIMING
        double t_end = MPI_Wtime();
        if (num_timestamps < 500)
        {
            timestamps_end[num_timestamps] = t_end;
            num_timestamps++;
        }
        double t_op = t_end - t_start;
#endif // ENABLE_EXEC_TIMING

#if ENABLE_LATE_ARRIVAL_TIMING
        double t_arrival = t_barrier_end - t_barrier_start;
#endif // ENABLE_LATE_ARRIVAL_TIMING

#if ENABLE_DISPLS
        // Gather receive displacements
        sbuf = NULL;
        PMPI_Gather(rdispls, comm_size, MPI_INT, rbuf, comm_size, MPI_INT, 0, comm);
#else
        // Gather a bunch of counters
        PMPI_Gather(&sendcount, 1, MPI_INT, sbuf, 1, MPI_INT, 0, comm);
        PMPI_Gather(recvcounts, comm_size, MPI_INT, rbuf, comm_size, MPI_INT, 0, comm);
#endif // ENABLE_DISPLS

#if ENABLE_EXEC_TIMING
        PMPI_Gather(&t_op, 1, MPI_DOUBLE, op_exec_times, 1, MPI_DOUBLE, 0, comm);
#endif // ENABLE_EXEC_TIMING

#if ENABLE_LATE_ARRIVAL_TIMING
        PMPI_Gather(&t_arrival, 1, MPI_DOUBLE, late_arrival_timings, 1, MPI_DOUBLE, 0, comm);
#endif // ENABLE_LATE_ARRIVAL_TIMING

#if ENABLE_SAVE_DATA_VALIDATION
        if (do_send_buffs > 0)
        {
            int dtsize;
            PMPI_Type_size(sendtype, &dtsize);
            store_call_data_single_count(collective_name, SEND_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)sendbuf, sendcount, sendtype);
        }
        else
        {
            int dtsize;
            PMPI_Type_size(recvtype, &dtsize);
            store_call_data(collective_name, RECV_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)recvbuf, (int *)recvcounts, (int *)rdispls, recvtype);
        }

        if (allgathervCalls == max_call)
        {
            fprintf(stderr, "Reaching the limit, check successful\n");
            PMPI_Abort(MPI_COMM_WORLD, 32);
        }
#endif // ENABLE_SAVE_DATA_VALIDATION

#if ENABLE_COMPARE_DATA_VALIDATION
        if (do_send_buffs > 0)
        {
            if (allgathervCalls == max_call)
            {
                fprintf(stderr, "Reaching the analysis limit, check successful\n");
                PMPI_Abort(MPI_COMM_WORLD, 1);
            }
            if (my_comm_rank == 0)
            {
                fprintf(stderr, "Checking call %" PRIu64 "\n", allgathervCalls);
            }
            if (max_call == -1 || (max_call > -1 && allgathervCalls < max_call))
            {
                read_and_compare_call_data(collective_name, SEND_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)sendbuf, (int *)sendcount, NULL, sendtype, true);
            }
            else
            {
                read_and_compare_call_data(collective_name, SEND_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)sendbuf, (int *)sendcount, NULL, sendtype, false);
            }
        }
        else
        {
            if (allgathervCalls == max_call)
            {
                fprintf(stderr, "Reaching the analysis limit, check successful\n");
                PMPI_Abort(MPI_COMM_WORLD, 1);
            }
            if (max_call == -1 || (max_call > -1 && allgathervCalls < max_call))
            {
                read_and_compare_call_data(collective_name, RECV_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)recvbuf, (int *)recvcounts, (int *)rdispls, recvtype, true);
            }
            else
            {
                read_and_compare_call_data(collective_name, RECV_CONTEXT_IDX, comm, my_comm_rank, world_rank, allgathervCalls, (void *)recvbuf, (int *)recvcounts, (int *)rdispls, recvtype, false);
            }
        }
#endif // ENABLE_COMPARE_DATA_VALIDATION

#if ENABLE_LOCATION_TRACKING
        int my_pid = getpid();
        // Note that the library will free all the allocated memory. We hand over the pointer
        // and the profiler frees the memory when finalizing
        int *pids = (int *)malloc(comm_size * sizeof(int));
        assert(pids);
        int *world_comm_ranks = (int *)malloc(comm_size * sizeof(int));
        assert(world_comm_ranks);
        char hostname[256];
        gethostname(hostname, 256);
        char *hostnames = (char *)malloc(256 * comm_size * sizeof(char));
        assert(hostnames);

        PMPI_Gather(&my_pid, 1, MPI_INT, pids, 1, MPI_INT, 0, comm);
        PMPI_Gather(&world_rank, 1, MPI_INT, world_comm_ranks, 1, MPI_INT, 0, comm);
        PMPI_Gather(&hostname, 256, MPI_CHAR, hostnames, 256, MPI_CHAR, 0, comm);
        if (my_comm_rank == 0)
        {
            int rc = commit_rank_locations(collective_name, comm, comm_size, world_rank, my_comm_rank, pids, world_comm_ranks, hostnames, allgathervCalls);
            if (rc)
            {
                fprintf(stderr, "save_rank_locations() failed: %d", rc);
                PMPI_Abort(MPI_COMM_WORLD, 1);
            }
        }
#endif // ENABLE_LOCATION_TRACKING

        if (my_comm_rank == 0)
        {
#if DEBUG
            fprintf(logger->f, "Root: global %d - %d   local %d - %d\n", world_size, myrank, size, localrank);
#endif

#if ENABLE_DISPLS
            DEBUG_ALLGATHERV_PROFILING("Saving displacement data of call #%" PRIu64 ".\n", allgathervCalls);
            int s_dt_size, r_dt_size;
            PMPI_Type_size(sendtype, &s_dt_size);
            PMPI_Type_size(recvtype, &r_dt_size);
            if (insert_displ_data(rbuf, comm_size, s_dt_size, r_dt_size))
            {
                fprintf(stderr, "[%s:%d][ERROR] unable to insert displacement data\n", __FILE__, __LINE__);
                PMPI_Abort(MPI_COMM_WORLD, 1);
            }
#endif // ENABLE_DISPLS

#if ((ENABLE_RAW_DATA || ENABLE_PER_RANK_STATS || ENABLE_VALIDATION) && ENABLE_COMPACT_FORMAT)
            DEBUG_ALLGATHERV_PROFILING("Saving data of call #%" PRIu64 ".\n", allgathervCalls);
            int s_dt_size, r_dt_size;
            PMPI_Type_size(sendtype, &s_dt_size);
            PMPI_Type_size(recvtype, &r_dt_size);
            if (insert_sendrecv_count_data(sbuf, rbuf, comm_size, s_dt_size, r_dt_size))
            {
                fprintf(stderr, "[%s:%d][ERROR] unable to insert send/recv counts\n", __FILE__, __LINE__);
                PMPI_Abort(MPI_COMM_WORLD, 1);
            }
#endif // ((ENABLE_RAW_DATA || ENABLE_PER_RANK_STATS || ENABLE_VALIDATION) && ENABLE_COMPACT_FORMAT)

#if ((ENABLE_RAW_DATA || ENABLE_PER_RANK_STATS || ENABLE_VALIDATION) && !ENABLE_COMPACT_FORMAT)
            DEBUG_ALLGATHERV_PROFILING("Saving data of call #%" PRIu64 ".\n", allgathervCalls);
            int s_dt_size, r_dt_size;
            PMPI_Type_size(sendtype, &s_dt_size);
            PMPI_Type_size(recvtype, &r_dt_size);
            save_counts(sbuf, rbuf, s_dt_size, r_dt_size, comm_size, allgathervCalls);
#endif // ((ENABLE_RAW_DATA || ENABLE_PER_RANK_STATS || ENABLE_VALIDATION) && !ENABLE_COMPACT_FORMAT)

#if ENABLE_PATTERN_DETECTION
            commit_pattern_from_counts(allgathervCalls, sbuf, rbuf, size);
#endif

#if ENABLE_EXEC_TIMING
            int jobid = get_job_id();
            int rc = commit_timings(comm, collective_name, world_rank, my_comm_rank, jobid, op_exec_times, comm_size, allgathervCalls);
            if (rc)
            {
                fprintf(stderr, "commit_timings() failed: %d\n", rc);
                PMPI_Abort(MPI_COMM_WORLD, 1);
            }
#endif // ENABLE_EXEC_TIMING

#if ENABLE_LATE_ARRIVAL_TIMING
            int jobid = get_job_id();
            int rc = commit_timings(comm, collective_name, world_rank, my_comm_rank, jobid, late_arrival_timings, comm_size, allgathervCalls);
            if (rc)
            {
                fprintf(stderr, "commit_timings() failed: %d\n", rc);
                PMPI_Abort(MPI_COMM_WORLD, 1);
            }
#endif // ENABLE_LATE_ARRIVAL_TIMING
            allgathervCallsLogged++;
        }

#if ENABLE_LATE_ARRIVAL_TIMING
        // All ranks sync so that if we have I/O happening for some ranks during the data commit, it would not skew the next timings
        PMPI_Barrier(comm);
#endif // ENABLE_LATE_ARRIVAL_TIMING
    }
    else
    {
        // No need to profile that call but we still count the number of allgatherv calls
        ret = PMPI_Allgatherv(sendbuf, sendcount, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
    }

#if SYNC
    // We sync all the ranks again to make sure that rank 0, who does some calculations,
    // does not artificially fall behind.
    PMPI_Barrier(comm);
#endif // SYNC

    char *need_data_commit_str = getenv(A2A_COMMIT_PROFILER_DATA_AT_ENVVAR);
    char *need_to_free_data = getenv(A2A_RELEASE_RESOURCES_AFTER_DATA_COMMIT_ENVVAR);

    if (need_data_commit_str != NULL)
    {
        int targetCallID = atoi(need_data_commit_str);
        if (allgathervCalls == targetCallID)
        {
            _commit_data();
        }
    }

    if (need_to_free_data != NULL && strncmp(need_to_free_data, "0", 1) != 0)
    {
        _release_profiling_resources();
    }

    // allgathervCalls is the absolute number of calls that the rank is dealing with
    allgathervCalls++;

    return ret;
}

int MPI_Allgatherv(const void *sendbuf, const int sendcount, MPI_Datatype sendtype,
                   void *recvbuf, const int *recvcounts, const int *rdispls, MPI_Datatype recvtype,
                   MPI_Comm comm)
{
    return _mpi_allgatherv(sendbuf, sendcount, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
}

void mpi_allgatherv_(void *sendbuf, MPI_Fint *sendcount, MPI_Fint *sendtype,
                     void *recvbuf, MPI_Fint *recvcount, MPI_Fint *rdispls, MPI_Fint *recvtype,
                     MPI_Fint *comm, MPI_Fint *ierr)
{
    int c_ierr;
    MPI_Comm c_comm;
    MPI_Datatype c_sendtype, c_recvtype;

    c_comm = PMPI_Comm_f2c(*comm);
    c_sendtype = PMPI_Type_f2c(*sendtype);
    c_recvtype = PMPI_Type_f2c(*recvtype);

    sendbuf = (char *)OMPI_F2C_IN_PLACE(sendbuf);
    sendbuf = (char *)OMPI_F2C_BOTTOM(sendbuf);
    recvbuf = (char *)OMPI_F2C_BOTTOM(recvbuf);

    c_ierr = MPI_Allgatherv(sendbuf,
                            OMPI_FINT_2_INT(*sendcount),
                            c_sendtype,
                            recvbuf,
                            (int *)OMPI_FINT_2_INT(recvcount),
                            (int *)OMPI_FINT_2_INT(rdispls),
                            c_recvtype, c_comm);
    if (NULL != ierr)
        *ierr = OMPI_INT_2_FINT(c_ierr);
}

// This is a duplicate of MPI_Finalize() just in case we face a failure or
// if the app never calls MPI_Finalize().
__attribute__((destructor)) void calledLast()
{
    _commit_data();
    _finalize_profiling();
}

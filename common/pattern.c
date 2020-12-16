/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include "pattern.h"

static avPattern_t *new_pattern(int num_ranks, int num_peers)
{
    avPattern_t *sp = malloc(sizeof(avPattern_t));
    sp->n_ranks = num_ranks;
    sp->n_peers = num_peers;
    sp->n_calls = 1;
    sp->comm_size = -1;
    sp->next = NULL;
    return sp;
}

static avPattern_t *new_pattern_with_size(int num_ranks, int num_peers, int size)
{
    avPattern_t *p = new_pattern(num_ranks, num_peers);
    p->comm_size = size;
    return p;
}

avPattern_t *add_pattern_for_size(avPattern_t *patterns, int num_ranks, int num_peers, int size)
{
    DEBUG_PATTERN("Adding pattern\n");
    if (patterns == NULL)
    {
        DEBUG_PATTERN("Adding pattern to empty list\n");
        return new_pattern_with_size(num_ranks, num_peers, size);
    }
    else
    {
        avPattern_t *ptr = patterns;
        DEBUG_PATTERN("We already have patterns, comparing...\n");

        while (ptr != NULL)
        {
            if (ptr->n_ranks == num_ranks && ptr->n_peers == num_peers && size == ptr->comm_size)
            {
                DEBUG_PATTERN("Pattern already exists\n");
                ptr->n_calls++;
                DEBUG_PATTERN("Pattern successfully added\n");
                return patterns;
            }
            ptr = ptr->next;
        }

        DEBUG_PATTERN("Adding new pattern to list\n");
        ptr = patterns;
        assert(ptr);
        while (ptr->next != NULL)
        {
            ptr = ptr->next;
        }

        // Pattern does not exist yet, adding it to the head
        // First find the tail of the list
        avPattern_t *np = new_pattern_with_size(num_ranks, num_peers, size);
        ptr->next = np;
        return patterns;
    }

    return NULL;
}

avPattern_t *add_pattern(avPattern_t *patterns, int num_ranks, int num_peers)
{
    DEBUG_PATTERN("Adding pattern\n");
    if (patterns == NULL)
    {
        DEBUG_PATTERN("Adding pattern to empty list\n");
        return new_pattern(num_ranks, num_peers);
    }
    else
    {
        avPattern_t *ptr = patterns;
        DEBUG_PATTERN("We already have patterns, comparing...\n");

        while (ptr != NULL)
        {
            if (ptr->n_ranks == num_ranks && ptr->n_peers == num_peers)
            {
                DEBUG_PATTERN("Pattern already exists\n");
                ptr->n_calls++;
                DEBUG_PATTERN("Pattern successfully added\n");
                return patterns;
            }
            ptr = ptr->next;
        }

        DEBUG_PATTERN("Adding new pattern to list\n");
        ptr = patterns;
        assert(ptr);
        while (ptr->next != NULL)
        {
            ptr = ptr->next;
        }

        // Pattern does not exist yet, adding it to the head
        // First find the tail of the list
        avPattern_t *np = new_pattern(num_ranks, num_peers);
        ptr->next = np;
        return patterns;
    }

    return NULL;
}

int get_size_patterns(avPattern_t *p)
{
    avPattern_t *ptr = p;
    int count = 0;

    while (ptr != NULL)
    {
        count++;
        ptr = ptr->next;
    }
    return count;
}

bool compare_patterns(avPattern_t *p1, avPattern_t *p2)
{
    avPattern_t *ptr1 = p2;
    int s1, s2;

    s1 = get_size_patterns(p1);
    s2 = get_size_patterns(p2);
    if (s1 != s2)
    {
        return false;
    }

    // For each elements of p1, we have to scan the entire p2 because we have no guarantee about ordering
    while (ptr1 != NULL)
    {
        avPattern_t *ptr2 = p1;
        while (ptr2 != NULL)
        {
            if (ptr2->comm_size != ptr1->comm_size || ptr2->n_peers != ptr1->n_peers || ptr2->n_ranks != ptr1->n_ranks)
            {
                break;
            }
            ptr2 = ptr2->next;
        }
        if (ptr2 == NULL)
        {
            return false;
        }
        ptr1 = ptr1->next;
    }

    return true;
}

avCallPattern_t *lookup_call_patterns(avCallPattern_t *call_patterns)
{
    avCallPattern_t *ptr = call_patterns;
    bool spatterns_are_identical = false;
    bool rpatterns_are_identical = true;

    while (ptr != NULL)
    {
        if (compare_patterns(ptr->spatterns, call_patterns->spatterns) == true)
        {
            spatterns_are_identical = true;
        }

        if (compare_patterns(ptr->rpatterns, call_patterns->rpatterns) == true)
        {
            rpatterns_are_identical = true;
        }

        if (spatterns_are_identical && rpatterns_are_identical)
        {
            return ptr;
        }

        ptr = ptr->next;
    }
    return NULL;
}

void free_patterns(avPattern_t *p)
{
    avPattern_t *ptr = p;

    if (p == NULL)
    {
        return;
    }

    while (ptr != NULL)
    {
        avPattern_t *ptr2 = ptr->next;
        free(ptr);
        ptr = ptr2;
    }
}

avCallPattern_t *extract_call_patterns(int callID, int *send_counts, int *recv_counts, int size)
{
    int i, j, num;
    int src_ranks = 0;
    int dst_ranks = 0;
    int send_patterns[size + 1];
    int recv_patterns[size + 1];

    avCallPattern_t *cp = (avCallPattern_t *)calloc(1, sizeof(avCallPattern_t));
    cp->n_calls = 1;

    DEBUG_PATTERN("Extracting call patterns\n");

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
    DEBUG_PATTERN("Handling call send patterns\n");
    for (i = 0; i < size; i++)
    {
        if (send_patterns[i] != 0)
        {
            cp->spatterns = add_pattern_for_size(cp->spatterns, send_patterns[i], i + 1, size);
        }
    }
    DEBUG_PATTERN("Handling call receive patterns\n");
    for (i = 0; i < size; i++)
    {
        if (recv_patterns[i] != 0)
        {
            cp->rpatterns = add_pattern_for_size(cp->rpatterns, recv_patterns[i], i + 1, size);
        }
    }

    return cp;
}

/*************************************************************************
 * Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <assert.h>

#include "format.h"
#include "common_utils.h"

static char *add_range_uint64(char *str, uint64_t start, uint64_t end)
{
    int size;
    if (str == NULL)
    {
        size = MAX_STRING_LEN;
    }
    else
    {
        size = strlen(str) + (MAX_STRING_LEN - get_remainder(strlen(str), MAX_STRING_LEN));
    }
    int ret = size;

    if (str == NULL)
    {
        str = (char *)malloc(size * sizeof(char));
        assert(str);
        while (ret >= size)
        {
            ret = snprintf(str, size, "%" PRIu64 "-%" PRIu64, start, end);
            if (ret < 0)
            {
                fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
                return NULL;
            }
            if (ret >= size)
            {
                // truncated result, increasing the size of the buffer and trying again
                size = size * 2;
                str = (char *)realloc(str, size);
                assert(str);
            }
        }
        return str;
    }
    else
    {
        // We make sure we do not get a truncated result
        char *s = NULL;
        while (ret >= size)
        {
            if (s == NULL)
            {
                s = (char *)malloc(size * sizeof(char));
                assert(s);
            }
            else
            {
                // truncated result, increasing the size of the buffer and trying again
                size = size * 2;
                s = (char *)realloc(s, size);
                assert(s);
            }
            ret = snprintf(s, size, "%s, %" PRIu64 "-%" PRIu64, str, start, end);
            if (ret < 0)
            {
                fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
                return NULL;
            }
        }

        if (s != NULL)
        {
            if (str != NULL)
            {
                free(str);
            }
            str = s;
        }

        return str;
    }
}

static char *add_range(char *str, int start, int end)
{
    int size;
    if (str == NULL)
    {
        size = MAX_STRING_LEN;
    }
    else
    {
        size = strlen(str) + (MAX_STRING_LEN - get_remainder(strlen(str), MAX_STRING_LEN));
    }
    int ret = size;

    if (str == NULL)
    {
        str = (char *)malloc(size * sizeof(char));
        assert(str);
        while (ret >= size)
        {
            ret = snprintf(str, size, "%d-%d", start, end);
            if (ret < 0)
            {
                fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
                return NULL;
            }
            if (ret >= size)
            {
                // truncated result, increasing the size of the buffer and trying again
                size = size * 2;
                str = (char *)realloc(str, size);
                assert(str);
            }
        }
        return str;
    }
    else
    {
        // We make sure we do not get a truncated result
        char *s = NULL;
        while (ret >= size)
        {
            if (s == NULL)
            {
                s = (char *)malloc(size * sizeof(char));
                assert(s);
            }
            else
            {
                // truncated result, increasing the size of the buffer and trying again
                size = size * 2;
                s = (char *)realloc(s, size);
                assert(s);
            }
            ret = snprintf(s, size, "%s, %d-%d", str, start, end);
            if (ret < 0)
            {
                fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
                return NULL;
            }
        }

        if (s != NULL)
        {
            if (str != NULL)
            {
                free(str);
            }
            str = s;
        }

        return str;
    }
}

static char *add_singleton_uint64(char *str, uint64_t n)
{

    int size;
    int rc;
    if (str == NULL)
    {
        size = MAX_STRING_LEN;
    }
    else
    {
        size = strlen(str) + (MAX_STRING_LEN - get_remainder(strlen(str), MAX_STRING_LEN));
    }
    int ret = size;
    if (str == NULL)
    {
        str = (char *)malloc(size * sizeof(char));
        assert(str);
        rc = sprintf(str, "%" PRIu64, n);
        assert(rc <= size);
        return str;
    }

    // We make sure we do not get a truncated result
    char *s = NULL;
    while (ret >= size)
    {
        if (s == NULL)
        {
            s = (char *)malloc(size * sizeof(char));
            assert(s);
        }
        else
        {
            // truncated result, increasing the size of the buffer and trying again
            size = size * 2;
            s = (char *)realloc(s, size);
            assert(s);
        }
        ret = snprintf(s, size, "%s, %" PRIu64, str, n);
        if (ret < 0)
        {
            fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
            return NULL;
        }
    }

    if (s != NULL)
    {
        if (str != NULL)
        {
            free(str);
        }
        str = s;
    }

    return str;
}

static char *add_singleton(char *str, int n)
{

    int size;
    int rc;
    if (str == NULL)
    {
        size = MAX_STRING_LEN;
    }
    else
    {
        size = strlen(str) + (MAX_STRING_LEN - get_remainder(strlen(str), MAX_STRING_LEN));
    }
    int ret = size;
    if (str == NULL)
    {
        str = (char *)malloc(size * sizeof(char));
        assert(str);
        rc = sprintf(str, "%d", n);
        assert(rc <= size);
        return str;
    }

    // We make sure we do not get a truncated result
    char *s = NULL;
    while (ret >= size)
    {
        if (s == NULL)
        {
            s = (char *)malloc(size * sizeof(char));
            assert(s);
        }
        else
        {
            // truncated result, increasing the size of the buffer and trying again
            size = size * 2;
            s = (char *)realloc(s, size);
            assert(s);
        }
        ret = snprintf(s, size, "%s, %d", str, n);
        if (ret < 0)
        {
            fprintf(stderr, "[%s:%d] snprintf failed\n", __FILE__, __LINE__);
            return NULL;
        }
    }

    if (s != NULL)
    {
        if (str != NULL)
        {
            free(str);
        }
        str = s;
    }

    return str;
}

char *compress_uint64_array(uint64_t *array, size_t size)
{
    size_t i, start;
    char *compressedRep = NULL;

#if DEBUG
    fprintf(stderr, "Compressing:");
    for (i = 0; i < size; i++)
    {
        fprintf(stderr, " %" PRIu64, array[i]);
    }
    fprintf(stderr, "\n");
#endif // DEBUG

    for (i = 0; i < size; i++)
    {
        start = i;
        while (i + 1 < size && array[i] + 1 == array[i + 1])
        {
            i++;
        }
        if (i != start)
        {
            // We found a range
            compressedRep = add_range_uint64(compressedRep, array[start], array[i]);
        }
        else
        {
            // We found a singleton
            compressedRep = add_singleton_uint64(compressedRep, array[i]);
        }
    }
#if DEBUG
    fprintf(stderr, "Compressed version is: %s\n", compressedRep);
#endif // DEBUG
    return compressedRep;
}

char *compress_int_array(int *array, int size)
{
    int i, start;
    char *compressedRep = NULL;

#if DEBUG
    fprintf(stderr, "Compressing:");
    for (i = 0; i < size; i++)
    {
        fprintf(stderr, " %d", array[i]);
    }
    fprintf(stderr, "\n");
#endif // DEBUG

    for (i = 0; i < size; i++)
    {
        start = i;
        while (i + 1 < size && array[i] + 1 == array[i + 1])
        {
            i++;
        }
        if (i != start)
        {
            // We found a range
            compressedRep = add_range(compressedRep, array[start], array[i]);
        }
        else
        {
            // We found a singleton
            compressedRep = add_singleton(compressedRep, array[i]);
        }
    }
#if DEBUG
    fprintf(stderr, "Compressed version is: %s\n", compressedRep);
#endif // DEBUG
    return compressedRep;
}

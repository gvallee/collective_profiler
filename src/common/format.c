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


static char *_compress_uint64_vec(uint64_t *array, size_t start_idx, size_t size)
{
    size_t i, start;
    char *compressedRep = NULL;

#if DEBUG
    fprintf(stderr, "Compressing:");
    for (i = 0; i < size; i++)
    {
        fprintf(stderr, " %d", array[i]);
    }
    fprintf(stderr, "\n");
#endif // DEBUG

    for (i = start_idx; i < start_idx + size; i++)
    {
        start = i;
        while (i + 1 < start_idx + size && array[i] + 1 == array[i + 1])
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

// compress_uint64_array compresses a matrix or a vector of uint64_t
// The distinction between a matrix and a vector must be specified through the xsize and ysize parameters
char *compress_uint64_array(uint64_t *array, size_t xsize,  size_t ysize)
{
    int rc;
    size_t idx;
    char *compressedRep = NULL;
    for (idx = 0; idx < xsize * ysize; idx += xsize) 
    {
        char *compressed_line = _compress_uint64_vec(array, idx, xsize);
        if (compressedRep == NULL) {
            compressedRep = strdup(compressed_line);
        }
        else
        {
            compressedRep = realloc (compressedRep, strlen (compressedRep) + strlen (compressed_line) + 2);
            size_t n;
            size_t copy_idx = strlen(compressedRep);
            compressedRep[copy_idx] = '\n';
            copy_idx++;
            for (n = 0; n < strlen(compressed_line); n++)
            {
                compressedRep[copy_idx] = compressed_line[n];
                copy_idx++;
            }
            compressedRep[copy_idx] = '\0';
        }
        free(compressed_line);
    }
    return compressedRep;
}

static char *_compress_int_vec(int *array, size_t start_idx, size_t size)
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

    for (i = start_idx; i < start_idx + size; i++)
    {
        start = i;
        while (i + 1 < start_idx + size && array[i] + 1 == array[i + 1])
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

// compress_int_array compresses a matrix or a vector of int.
// The distinction between a matrix and a vector must be specified through the xsize and ysize parameters
char *compress_int_array(int *array, int xsize,  int ysize)
{
    int rc;
    size_t idx;
    char *compressedRep = NULL;
    for (idx = 0; idx < xsize * ysize; idx += xsize) 
    {
        char *compressed_line = _compress_int_vec(array, idx, xsize);
        if (compressedRep == NULL) {
            compressedRep = strdup(compressed_line);
        }
        else
        {
            compressedRep = realloc (compressedRep, strlen (compressedRep) + strlen (compressed_line) + 2);
            size_t n;
            size_t copy_idx = strlen(compressedRep);
            compressedRep[copy_idx] = '\n';
            copy_idx++;
            for (n = 0; n < strlen(compressed_line); n++)
            {
                compressedRep[copy_idx] = compressed_line[n];
                copy_idx++;
            }
            compressedRep[copy_idx] = '\0';
        }
        free(compressed_line);
    }
    return compressedRep;
}

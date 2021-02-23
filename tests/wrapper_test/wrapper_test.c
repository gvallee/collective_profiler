/*************************************************************************
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <stdio.h>

#include "mpi.h"

int main(int argc, char **argv)
{
    int my_rank;
    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &my_rank);
    if (my_rank == 0)
    {
        char str[100];
        while (scanf("%s", str) != EOF)
        {
            fprintf(stdout, "%s\n", str);
        }

    }
    MPI_Finalize();
}

# ******************************************************************************************************
# * Copyright (c) 2021, University College London and Mellanox Technolgies Limited. All rights reserved.
# ******************************************************************************************************/

all: alltoall_examples

alltoall_examples: alltoall.c
	mpicc -g alltoall.c -o alltoall

check: all 

clean:
	@rm -f alltoall
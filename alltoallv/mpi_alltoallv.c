/*************************************************************************
 * Copyright (c) 2019-2010, Mellanox Technologies, Inc. All rights reserved.
 * Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
 *
 * See LICENSE.txt for license information
 ************************************************************************/

#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>

#define DEBUG 0

// Data type for storing comm size, alltoallv counts, send/recv count, etc
struct avNode
{
	int size;
	int count;
	int comm;
	int *send_data;
	int *recv_data;
	struct avNode *next;
};

static FILE *f=NULL;
static struct avNode *head=NULL;
static int myrank = 0;
static int avCalls = 0;

// Compare if two arrays are identical.
static int same_data(int *dest, int *src, int size)
{
	int i, j, num = 0;
	for(i=0; i<size; i++){
	for(j=0; j<size; j++){
		if(dest[num] != src[num]) return 0;
		num++;
	}
	}
	return 1;
}


// Compare new send count data with existing data.
// If there is a match, increas the counter. Add new data, otherwise.
// recv count was not compared.
static void insert_data(int *sbuf, int *rbuf, int size) {
	int i, j, num = 0;
	struct avNode *newNode = NULL;
	struct avNode *temp;

	temp = head;
	while(temp != NULL){
		if(same_data(temp->send_data, sbuf, size) == 0 || temp->size != size){
                        if(DEBUG) fprintf(f, "new data: %d\n", size);
			if(temp->next != NULL)
				temp = temp->next;
			else
				break;
		}
		else {
			temp->count++;
                        if(DEBUG) fprintf(f, "old data: %d --> %d --- %d\n", size, temp->size, temp->count);
			return;
		}
	}
	
	if(DEBUG) fprintf(f, "no data: %d \n", size);
	newNode = (struct avNode*)malloc(sizeof(struct avNode));
        assert(newNode != NULL);

	newNode->size = size;
	newNode->count = 1;
	newNode->send_data = (int*)malloc(size*size*(sizeof(int)));
	newNode->recv_data = (int*)malloc(size*size*(sizeof(int)));
	newNode->next = NULL;
	if(DEBUG) fprintf(f, "new entry: %d --> %d --- %d\n", size, newNode->size, newNode->count);

	for(i=0; i<size; i++){
	for(j=0; j<size; j++){
		newNode->send_data[num] = sbuf[num];
		newNode->recv_data[num] = rbuf[num];
		num++;
	}
	}

	if(head == NULL){
		head = newNode;
	}
	else {
		temp->next = newNode;
	}
}

static void print_data(int *buf, int size)
{
	int i, j, num = 0;
	for(i=0; i<size; i++) {
	for(j=0; j<size; j++) {
		fprintf(f, "%d ", buf[num]);
		// if(buf[num]>0) fprintf(f, "%d ", buf[num]);
		num++;
	}
	fprintf(f, "\n");
	}
}
	
static void display_data() {
	int i;
	struct avNode *temp;

	temp = head;
	while(temp != NULL){
		fprintf(f, "comm size = %d, alltoallv calls = %d\n", temp->size, temp->count);
		fprintf(f, "Send counts:\n");
		print_data(temp->send_data, temp->size);
		fprintf(f, "Recv counts:\n");
		print_data(temp->recv_data, temp->size);
		temp = temp->next;
	}
}

// During Finalize, it prints all stored data to a file.
int MPI_Finalize(){
	if(myrank == 0){
		if (f!=NULL) fprintf(f, "Totall alltoallv calls = %d\n", avCalls);
 	  	display_data();
		free(head);
	}
	PMPI_Finalize();
}



int MPI_Alltoallv(const void *sendbuf, const int *sendcounts, const int *sdispls,
                                 MPI_Datatype sendtype, void *recvbuf, const int *recvcounts,
                                 const int *rdispls, MPI_Datatype recvtype, MPI_Comm comm)
{
   int myrank;
   int size;
   int i, j;
   int *sbuf = NULL;
   int *rbuf = NULL;
   int localrank, globalsize;
   char buf[200];

   MPI_Comm_rank(MPI_COMM_WORLD, &myrank);
   MPI_Comm_size(MPI_COMM_WORLD, &globalsize);
   MPI_Comm_rank(comm, &localrank);
   MPI_Comm_size(comm, &size);
   avCalls++;

   if(f == NULL && myrank == 0){
     sprintf(buf, "profile_alltoallv.%d", myrank);
     f=fopen(buf, "w");
     assert(f != NULL);
   }

   sbuf = (int*)malloc(size*size*(sizeof(int)));
   rbuf = (int*)malloc(size*size*(sizeof(int)));

   MPI_Gather(sendcounts, size, MPI_INT, sbuf, size, MPI_INT, 0, comm);

   MPI_Gather(recvcounts, size, MPI_INT, rbuf, size, MPI_INT, 0, comm);

   if(myrank == 0)
   {
        if(DEBUG) fprintf(f, "Root: global %d - %d   local %d - %d\n", globalsize, myrank, size, localrank);
        insert_data(sbuf, rbuf, size);
        fflush(f);
   }

   free(rbuf);
   PMPI_Alltoallv(sendbuf, sendcounts, sdispls, sendtype, recvbuf, recvcounts, rdispls, recvtype, comm);
}

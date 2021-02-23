#ifndef ALLTOALL_TEST_HELPERS_H
#define ALLTOALL_TEST_HELPERS_H

/* helper functios for demos and tests of all to all */
#include <mpi.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdio.h>
#include <unistd.h>

#include "../src/alltoall/alltoall_profiler.h"
#define DEBUG_FLUSH 1


/* man page for MPI_Alltoall says
 * "The amount of data sent must be equal to the amount of data received, pairwise, between every pair of processes."
 * hence the first constant below is constant - can use different values of it in different runs */ 
#define RANK_TO_RANK_BLOCKSIZE 16 
// Constants for placement of info encoded in send and recv buffers
#define BYTE_1_MULTIPLIER 256
#define BYTE_2_MULTIPLIER 65536


/* MPI and C types used in this test are:  
    MPI_UINT8_T         uint8_t    index=0
    MPI_UINT16_T        uint16_t   index=1
    MPI_UINT32_T        uint32_t   index=2
    MPI_UINT64_T        uint64_t   index=3
*/
MPI_Datatype MPI_Datatypes_used[4] = {MPI_UINT8_T,MPI_UINT16_T,MPI_UINT32_T, MPI_UINT64_T};
const char type_strings[4][10] = {"uint8_t", "uint16_t", "uint32_t", "uint64_t"} ;  //for printing out the parameter sets 


typedef struct rank_set{
    int count;
    int ranks[10];
    MPI_Comm communicator;
} rank_set_t;


typedef struct alltoall_test_node_params {
    int send_type_idx;
    int recv_type_idx;
    int sendcount;
    int recvcount;
    rank_set_t* rank_set;
    int repetitions;
} alltoall_test_node_params_t;


bool is_rank_in_rankset(int rank, rank_set_t* rank_set){
    int i;
    for(i=0; i<rank_set->count; i++){
        if (rank_set->ranks[i] == rank) return true;
    }
    return false;
}


// creates a set of communicators having ranks defined by ranksets
void create_communicators(int world_size, rank_set_t* rank_sets, int rank_sets_count){
    DEBUG_ALLTOALL_PROFILING("params for create_communicators: worldsize = %i, ranks_sets_count = %i\n", world_size, rank_sets_count);
    int k;
    for (k=0; k<8; k++) DEBUG_ALLTOALL_PROFILING("%i ", rank_sets[0].ranks[k]); 
    DEBUG_ALLTOALL_PROFILING("\n)");

    //MPI_Comm** communicators = (MPI_Comm**) malloc(sizeof(MPI_Comm*) * world_size);
    MPI_Group world_group;
    DEBUG_ALLTOALL_PROFILING("calling MPI_Comm_group ...\n", NULL);
    MPI_Comm_group(MPI_COMM_WORLD, &world_group);

    int group_size;
    MPI_Group_size(world_group , &group_size);
    DEBUG_ALLTOALL_PROFILING("World group size = %i\n", group_size);

    int rank_set_idx;
    for (rank_set_idx=0; rank_set_idx< rank_sets_count; rank_set_idx++){
        DEBUG_ALLTOALL_PROFILING("IN LOOP\n");
        rank_set_t* rank_set = &rank_sets[rank_set_idx];
        for (k=0; k<8; k++) DEBUG_ALLTOALL_PROFILING("* %i ", rank_set->ranks[k]); 
        DEBUG_ALLTOALL_PROFILING("\n");
        // signature: MPI_Group_incl( MPI_Group group , int n , const int ranks[] , MPI_Group* newgroup);
        DEBUG_ALLTOALL_PROFILING("calling MPI_Group_incl rank_set_idx=%i ...\n", rank_set_idx);
        // signature: int MPI_Group_incl(MPI_Group group, int n, const int ranks[], MPI_Group *newgroup)
        MPI_Group group;
        DEBUG_ALLTOALL_PROFILING("rankSetcount = %i\n", rank_set->count);
        MPI_Group_incl(world_group, rank_set->count, rank_set->ranks, &group);
        DEBUG_ALLTOALL_PROFILING("calling MPI_Comm_create_group rank_set_idx=%i ...\n", rank_set_idx);
        // signature: MPI_Comm_create_group( MPI_Comm comm , MPI_Group group , int tag , MPI_Comm* newcomm);
        MPI_Comm_create_group(MPI_COMM_WORLD , group , 0, &rank_set->communicator);
        DEBUG_ALLTOALL_PROFILING("Group created ...\n", NULL);       
    }
    return;
}

void* create_sendbuf(alltoall_test_node_params_t* node_params){
    void* a;
    int i;
    int j;
    DEBUG_ALLTOALL_PROFILING("in create_Sendbuf\n", NULL);
    switch (node_params->send_type_idx){
        case 0: 
            DEBUG_ALLTOALL_PROFILING("in case 0: buffersize = %i\n", node_params->sendcount * node_params->rank_set->count);
            a = malloc(sizeof(uint8_t) * node_params->sendcount * node_params->rank_set->count);
            DEBUG_ALLTOALL_PROFILING("sendbuf initialised\n", NULL);
            uint8_t* b = (uint8_t*) a;
            DEBUG_ALLTOALL_PROFILING("some buffer items %i %i %i\n", b[0], b[1], b[2]);
            for (i=0; i < node_params->sendcount * node_params->rank_set->count; i++){
                DEBUG_ALLTOALL_PROFILING("i=%i ", i);
                b[i] = i / node_params->sendcount;
            }
            DEBUG_ALLTOALL_PROFILING("\n");
#if DEBUG == 1            
            for (j=0; j<64; j++) DEBUG_ALLTOALL_PROFILING("~~ %i ", b[j]);
#endif           
            return a;
            break;
        case 1: 
            a = malloc(sizeof(uint16_t) * node_params->sendcount * node_params->rank_set->count);
            for (i=0; i < node_params->sendcount * node_params->rank_set->count; i++) ((uint16_t*) a)[i] = i / node_params->sendcount;
            return a;
            break;
        case 2:
            a = malloc(sizeof(uint32_t) * node_params->sendcount * node_params->rank_set->count);
            for (i=0; i < node_params->sendcount * node_params->rank_set->count; i++) ((uint32_t*) a)[i] = i / node_params->sendcount;
            return a;
            break;
        case 3:
            a = malloc(sizeof(uint64_t) * node_params->sendcount * node_params->rank_set->count);
            for (i=0; i < node_params->sendcount * node_params->rank_set->count; i++) ((uint64_t*) a)[i] = i / node_params->sendcount;
            return a;
            break;
    }
    DEBUG_ALLTOALL_PROFILING("fell out of case!!!!\n", NULL);

}


void* create_recvbuf(alltoall_test_node_params_t* node_params){
    switch (node_params->recv_type_idx){
        void* a;
        int i;
        case 0:
            a = malloc(sizeof(uint8_t) * node_params->recvcount * node_params->rank_set->count);
            for (i=0; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint8_t*) a)[i] = 0;
            return a;
            break;
        case 1:
            a = malloc(sizeof(uint16_t) * node_params->recvcount * node_params->rank_set->count);
            for (i=0; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint16_t*) a)[i] = 0;
            return a;
            break;
        case 2:
            a = malloc(sizeof(uint32_t) * node_params->recvcount * node_params->rank_set->count);
            for (i=0; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint32_t*) a)[i] = 0;
            return a;
            break;
        case 3:
            a = malloc(sizeof(uint64_t) * node_params->recvcount * node_params->rank_set->count);
            for (i=0; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint64_t*) a)[i] = 0;
            return a;
            break;
    }
}



void print_buffers(int my_rank, int world_size, alltoall_test_node_params_t* param_set, void* sendbuf, void* recvbuf){
    // make sure only one rank prints at once, using barrier and sleep
    int rank;
    int block_idx;
    int idx;
    for (rank=0; rank<world_size; rank++){ 
        MPI_Barrier(param_set->rank_set->communicator);
        DEBUG_ALLTOALL_PROFILING("Done MPI_Barrier for print from rank = %i\n", rank);
        if (my_rank == rank){
            printf("Buffers for RANK #%i\n", my_rank);
            for (block_idx=0; block_idx<param_set->rank_set->count; block_idx++){
                printf("SENDBUF to rank #%i  : ", block_idx);
                for (idx=0; idx<param_set->sendcount; idx++){
                    switch (param_set->send_type_idx){
                        case 0:
                            printf(" %02x ", ((uint8_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
                            break;
                        case 1:
                            printf(" %04x ", ((uint16_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
                            break;
                        case 2:
                            printf(" %08x ", ((uint32_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
                            break;
                        case 3:
                            printf(" %016x ", ((uint64_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
                            break;
                    }
                }
                printf("\n");
                fflush(stdout);
            }
            for (block_idx=0; block_idx<param_set->rank_set->count; block_idx++){
                printf("RECVBUF from rank #%i: ", block_idx);
                for (idx=0; idx<param_set->recvcount; idx++){
                    switch (param_set->recv_type_idx){
                        case 0:
                            printf(" %02x ", ((uint8_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
                            break;
                        case 1:
                            printf(" %04x ", ((uint16_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
                            break;
                        case 2:
                            printf(" %08x ", ((uint32_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
                            break;
                        case 3:
                            printf(" %016x ", ((uint64_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
                            break;
                    }
                }
                printf("\n");
                fflush(stdout);
            }
            printf("\n");
            fflush(stdout);
        }
        sleep(1.0);            
    }
}

void do_test(alltoall_test_node_params_t* param_sets, int param_sets_set_count, int* param_sets_indices, int my_rank){
    int set_idx;
    int repetition;
    for (set_idx=0; set_idx<param_sets_set_count; set_idx++){
        alltoall_test_node_params_t* param_set = &param_sets[param_sets_indices[set_idx]];    

        /* test that my rank is one of the communicator used in this call - if not omit this call */
        if (is_rank_in_rankset(my_rank, param_set->rank_set)){
            void* sendbuf = create_sendbuf(param_set);
            void* recvbuf = create_recvbuf(param_set);
            for (repetition=0; repetition<param_set->repetitions; repetition++){
                MPI_Alltoall(sendbuf, param_set->sendcount, MPI_Datatypes_used[param_set->send_type_idx], recvbuf, param_set->recvcount, MPI_Datatypes_used[param_set->recv_type_idx], param_set->rank_set->communicator);
            }
#if DEBUG
            print_buffers(my_rank, world_size, param_set, sendbuf, recvbuf); /* note that this function has long calls to sleep() */
#endif
            free(recvbuf);
            free(sendbuf);
        }
    }
}



#endif /* ALLTOALL_TEST_HELPERS_H */
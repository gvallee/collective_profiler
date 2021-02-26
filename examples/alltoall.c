/* A test of MPI_Alltoall provding count patterns to be recorded by the samples 
 */ 

#include <mpi.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdio.h>
#include <unistd.h>

#include "../src/alltoall/alltoall_profiler.h"
#define DEBUG_FLUSH 1


// to be compiled with std=c99

/* man page for MPI_Alltoall says
 * "The amount of data sent must be equal to the amount of data received, pairwise, between every pair of processes."
 * hence the first constant below is constant - can use different values of it in different runs */ 
#define RANK_TO_RANK_BLOCKSIZE 16 
// Constants for placement of info encoded in send and recv buffers
#define BYTE_1_MULTIPLIER 256
#define BYTE_2_MULTIPLIER 65536

#define PARAM_SETS_COUNT 10
#define RANK_SETS_COUNT 3

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
} alltoall_test_node_params_t;


rank_set_t* create_rank_sets(){
    rank_set_t* rank_sets = malloc(sizeof(rank_set_t) * RANK_SETS_COUNT);  
    rank_set_t new_set0 = { .count= 8, .ranks = {0, 1, 2, 3, 4, 5, 6, 7} };
    rank_sets[0] = new_set0;
    rank_set_t new_set1 = { .count = 4, .ranks = {0, 1, 2, 3} };
    rank_sets[1] = new_set1;
    rank_set_t new_set2 = { .count = 4, .ranks = {4, 5, 6, 7} };
    rank_sets[2] = new_set2;
    return rank_sets;
}

bool is_rank_in_rankset(int rank, rank_set_t* rank_set){
    for(int i=0; i<rank_set->count; i++){
        if (rank_set->ranks[i] == rank) return true;
    }
    return false;
}

// creates a set of communicators having ranks defined by ranksets
void create_communicators(int world_size, rank_set_t* rank_sets, int rank_sets_count){
    DEBUG_ALLTOALL_PROFILING("params for create_communicators: worldsize = %i, ranks_sets_count = %i\n", world_size, rank_sets_count);
    for (int k; k<8; k++) DEBUG_ALLTOALL_PROFILING("%i ", rank_sets[0].ranks[k]); 
    DEBUG_ALLTOALL_PROFILING("\n)");

    //MPI_Comm** communicators = (MPI_Comm**) malloc(sizeof(MPI_Comm*) * world_size);
    MPI_Group world_group;
    DEBUG_ALLTOALL_PROFILING("calling MPI_Comm_group ...\n", NULL);
    MPI_Comm_group(MPI_COMM_WORLD, &world_group);

    int group_size;
    MPI_Group_size(world_group , &group_size);
    DEBUG_ALLTOALL_PROFILING("World group size = %i\n", group_size);

    for (int rank_set_idx=0; rank_set_idx< rank_sets_count; rank_set_idx++){
        DEBUG_ALLTOALL_PROFILING("IN LOOP\n");
        rank_set_t* rank_set = &rank_sets[rank_set_idx];
        for (int k; k<8; k++) DEBUG_ALLTOALL_PROFILING("* %i ", rank_set->ranks[k]); 
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


alltoall_test_node_params_t* alltoall_test_all_node_params_sets(rank_set_t* rank_sets){
    DEBUG_ALLTOALL_PROFILING("creating param sets ...\n", NULL);
    alltoall_test_node_params_t* params_sets = (alltoall_test_node_params_t*) malloc(sizeof(alltoall_test_node_params_t) * PARAM_SETS_COUNT);
    // template: paramset[] = (alltoall_test_node_params_t) {.send_type_idx =  , .recv_type_idx = , .sendcount =  , .recvcount = , .communicator = communicators[] };
    params_sets[0] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount =  8, .recvcount =  8, .rank_set = &rank_sets[0]};
    params_sets[1] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount = 16, .recvcount = 16, .rank_set = &rank_sets[0]};
    params_sets[2] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount = 32, .recvcount = 32, .rank_set = &rank_sets[1]};
    params_sets[3] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount = 64, .recvcount = 64, .rank_set = &rank_sets[0]};
    params_sets[4] = (alltoall_test_node_params_t) {.send_type_idx = 1, .recv_type_idx = 1, .sendcount =  8, .recvcount =  8, .rank_set = &rank_sets[0]};
    params_sets[5] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount = 16, .recvcount = 16, .rank_set = &rank_sets[0]};
    params_sets[6] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount =  8, .recvcount =  8, .rank_set = &rank_sets[1]};
    params_sets[7] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount = 16, .recvcount = 16, .rank_set = &rank_sets[1]};
    params_sets[8] = (alltoall_test_node_params_t) {.send_type_idx = 1, .recv_type_idx = 0, .sendcount =  8, .recvcount = 16, .rank_set = &rank_sets[0]};
    params_sets[9] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 1, .sendcount = 16, .recvcount =  8, .rank_set = &rank_sets[0]};
    DEBUG_ALLTOALL_PROFILING("param sets created\n", NULL);
    return params_sets;
} 

alltoall_test_node_params_t* alltoall_test_individual_node_params_sets(rank_set_t* rank_sets){
    DEBUG_ALLTOALL_PROFILING("creating param sets ...\n", NULL);
    alltoall_test_node_params_t* params_sets = (alltoall_test_node_params_t*) malloc(sizeof(alltoall_test_node_params_t) * PARAM_SETS_COUNT);
    // template: paramset[] = (alltoall_test_node_params_t) {.send_type_idx =  , .recv_type_idx = , .sendcount =  , .recvcount = , .communicator = communicators[] };
    params_sets[0] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount = 16, .recvcount = 16, .rank_set = &rank_sets[0]};
    params_sets[1] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 1, .sendcount = 16, .recvcount =  8, .rank_set = &rank_sets[0]};
    params_sets[2] = (alltoall_test_node_params_t) {.send_type_idx = 1, .recv_type_idx = 0, .sendcount =  8, .recvcount = 16, .rank_set = &rank_sets[0]};
    params_sets[3] = (alltoall_test_node_params_t) {.send_type_idx = 1, .recv_type_idx = 1, .sendcount =  8, .recvcount =  8, .rank_set = &rank_sets[0]};
    DEBUG_ALLTOALL_PROFILING("param sets created\n", NULL);
    return params_sets;
} 


void* create_sendbuf(alltoall_test_node_params_t* node_params){
    void* a;
    DEBUG_ALLTOALL_PROFILING("in create_Sendbuf\n", NULL);
    switch (node_params->send_type_idx){
        case 0: 
            DEBUG_ALLTOALL_PROFILING("in case 0: buffersize = %i\n", node_params->sendcount * node_params->rank_set->count);
            a = malloc(sizeof(uint8_t) * node_params->sendcount * node_params->rank_set->count);
            DEBUG_ALLTOALL_PROFILING("sendbuf initialised\n", NULL);
            uint8_t* b = (uint8_t*) a;
            DEBUG_ALLTOALL_PROFILING("some buffer items %i %i %i\n", b[0], b[1], b[2]);
            for (int i=0; i < node_params->sendcount * node_params->rank_set->count; i++){
                DEBUG_ALLTOALL_PROFILING("i=%i ", i);
                b[i] = i / node_params->sendcount;
            }
            DEBUG_ALLTOALL_PROFILING("\n");
#if DEBUG == 1            
            for (int j=0; j<64; j++) DEBUG_ALLTOALL_PROFILING("~~ %i ", b[j]);
#endif           
            return a;
            break;
        case 1: 
            a = malloc(sizeof(uint16_t) * node_params->sendcount * node_params->rank_set->count);
            for (int i=0; i < node_params->sendcount * node_params->rank_set->count; i++) ((uint16_t*) a)[i] = i / node_params->sendcount;
            return a;
            break;
        case 2:
            a = malloc(sizeof(uint32_t) * node_params->sendcount * node_params->rank_set->count);
            for (int i=0; i < node_params->sendcount * node_params->rank_set->count; i++) ((uint32_t*) a)[i] = i / node_params->sendcount;
            return a;
            break;
        case 3:
            a = malloc(sizeof(uint64_t) * node_params->sendcount * node_params->rank_set->count);
            for (int i=0; i < node_params->sendcount * node_params->rank_set->count; i++) ((uint64_t*) a)[i] = i / node_params->sendcount;
            return a;
            break;
    }
    DEBUG_ALLTOALL_PROFILING("fell out of case!!!!\n", NULL);

}


void* create_recvbuf(alltoall_test_node_params_t* node_params){
    switch (node_params->recv_type_idx){
        void* a;
        case 0:
            a = malloc(sizeof(uint8_t) * node_params->recvcount * node_params->rank_set->count);
            for (int i; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint8_t*) a)[i] = 0;
            return a;
            break;
        case 1:
            a = malloc(sizeof(uint16_t) * node_params->recvcount * node_params->rank_set->count);
            for (int i; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint16_t*) a)[i] = 0;
            return a;
            break;
        case 2:
            a = malloc(sizeof(uint32_t) * node_params->recvcount * node_params->rank_set->count);
            for (int i; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint32_t*) a)[i] = 0;
            return a;
            break;
        case 3:
            a = malloc(sizeof(uint64_t) * node_params->recvcount * node_params->rank_set->count);
            for (int i; i < node_params->recvcount * node_params->rank_set->count; i++) ((uint64_t*) a)[i] = 0;
            return a;
            break;
    }
}



void print_buffers(int my_rank, int world_size, alltoall_test_node_params_t* param_set, void* sendbuf, void* recvbuf){
    // make sure only one rank prints at once, using barrier and sleep
    for (int rank=0; rank<world_size; rank++){ 
        MPI_Barrier(param_set->rank_set->communicator);
        DEBUG_ALLTOALL_PROFILING("Done MPI_Barrier for print from rank = %i\n", rank);
        if (my_rank == rank){
            printf("Buffers for RANK #%i\n", my_rank);
            for (int block_idx=0; block_idx<param_set->rank_set->count; block_idx++){
                printf("SENDBUF to rank #%i  : ", block_idx);
                for (int idx=0; idx<param_set->sendcount; idx++){
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
                            printf(" %016lx ", ((uint64_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
                            break;
                    }
                }
                printf("\n");
                fflush(stdout);
            }
            for (int block_idx=0; block_idx<param_set->rank_set->count; block_idx++){
                printf("RECVBUF from rank #%i: ", block_idx);
                for (int idx=0; idx<param_set->recvcount; idx++){
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
                            printf(" %016lx ", ((uint64_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
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

int main(int argc, char *argv[]) {

    DEBUG_ALLTOALL_PROFILING("in main ...\n", NULL);
    // Intialise MPI
    int world_size, my_rank;
    MPI_Init(NULL, NULL);
    MPI_Comm_size(MPI_COMM_WORLD, &world_size);
    MPI_Comm_rank(MPI_COMM_WORLD, &my_rank);
    DEBUG_ALLTOALL_PROFILING("MPI initialsised: world_size=%i, my_rank=%i\n", world_size, my_rank);    
    // Test #1 all ranks have same send and recv types same as its receive type
    // and type for first half of ranks is uint32_t, and second half is uint8_t
    printf("MPI Datatypes used:\n");
    for (int i=0; i<4; i++){
        printf("name, value: %s, %li\n", type_strings[i], (uint64_t) MPI_Datatypes_used[i]);
    }


    // set up alltoall parameter sets and the communicators therefor
    rank_set_t* rank_sets = create_rank_sets();

    DEBUG_ALLTOALL_PROFILING("calling create_communicators ...\n", NULL);
    create_communicators(world_size, rank_sets, RANK_SETS_COUNT); // TODO macro for the number of ranks sets =3
    DEBUG_ALLTOALL_PROFILING("creating alltoall_test_all_node_params_sets ...\n", NULL);
    alltoall_test_node_params_t* param_sets = alltoall_test_all_node_params_sets(rank_sets);
    DEBUG_ALLTOALL_PROFILING("returned from alltoall_test_all_node_params_sets()\n", NULL);
    DEBUG_ALLTOALL_PROFILING("param set [0]:\n", NULL);
    DEBUG_ALLTOALL_PROFILING(".send_type_idx = %i\n", param_sets[0].send_type_idx);
    DEBUG_ALLTOALL_PROFILING(".sendcount = %i\n", param_sets[0].sendcount);

    // to test aggregation of patters this set should have duplicates
    // int param_sets_indices[] = {2, 1, 2, 1, 1, 2, 2, 1, 2};  // this is the highest level of the pattern of the MPI_alltoall calls - each int here specifies the parameter set to be used in one MPI_alltoall
    int param_sets_indices[] = {0, 1, 2, 3, 4, 5, 6, 7, 8, 9 };
    int param_sets_set_count = 10;
    DEBUG_ALLTOALL_PROFILING("created param set indices\n", NULL);
    
    // test 1 all ranks use same sendcount and recvcount so 
    if (my_rank == 0){
        printf("\n\nMPI_Alltoall test with all nodes having same send and receive type\n");
        fflush(stdout);
    }

    for (int set_idx=0; set_idx<param_sets_set_count; set_idx++){
        DEBUG_ALLTOALL_PROFILING("retrieving next parameter set ... *****************\n", NULL);
        alltoall_test_node_params_t* param_set = &param_sets[param_sets_indices[set_idx]];    // the same parameter set is used for all ranks in the alltoall call
        DEBUG_ALLTOALL_PROFILING("next parameter set retrieved\n", NULL);
        DEBUG_ALLTOALL_PROFILING("param set [this set]:\n", NULL);
        DEBUG_ALLTOALL_PROFILING(".send_type_idx = %i\n", param_set->send_type_idx);
        DEBUG_ALLTOALL_PROFILING(".sendcount = %i\n", param_set->sendcount);

        // is this bit seg faulting?????
        // if (my_rank == 0){
        //     printf("Calling MPI_Alltoall with send type = %s , recv type = %s , send count = %i ,  recv count = %i \n", type_strings[param_set->send_type_idx], param_set->sendcount, type_strings[param_set->recv_type_idx], param_set->recvcount);
        // }

        if (is_rank_in_rankset(my_rank, param_set->rank_set)){

            DEBUG_ALLTOALL_PROFILING("creating sendbuf...\n", NULL);
            void* sendbuf = create_sendbuf(param_set);
            DEBUG_ALLTOALL_PROFILING("created sendbuf\n", NULL);
            void* recvbuf = create_recvbuf(param_set);
            DEBUG_ALLTOALL_PROFILING("created recvbuf\n", NULL);
            // signature: MPI_Alltoall( const void* sendbuf , int sendcount , MPI_Datatype sendtype , void* recvbuf , int recvcount , MPI_Datatype recvtype , MPI_Comm comm);
            DEBUG_ALLTOALL_PROFILING("MPI_UINT8_T = %i\n", MPI_UINT8_T);
            //MPI_Alltoall(sendbuf, 8 , MPI_UINT8_T , recvbuf , 8 , MPI_UINT8_T , MPI_COMM_WORLD);
            DEBUG_ALLTOALL_PROFILING("Done basic MPI_Alltoall\n", NULL);

            // test that my rank is one of the communicator used in this call - if not omit this call
            DEBUG_ALLTOALL_PROFILING("DEBUG driver prog: send type index, value, %i, %i\n", param_set->send_type_idx, MPI_Datatypes_used[param_set->send_type_idx] );
            MPI_Alltoall(sendbuf, param_set->sendcount , MPI_Datatypes_used[param_set->send_type_idx] , recvbuf , param_set->recvcount , MPI_Datatypes_used[param_set->recv_type_idx] , param_set->rank_set->communicator);

            // make sure only one rank prints at once, using barrier and sleep
            print_buffers(my_rank, world_size, param_set, sendbuf, recvbuf);
            // for (int rank=0; rank<world_size; rank++){ 
            //     MPI_Barrier(param_set->rank_set->communicator);
            //     DEBUG_ALLTOALL_PROFILING("Done MPI_Barrier for print from rank = %i\n", rank);
            //     if (my_rank == rank){
            //         printf("Buffers for RANK #%i\n", my_rank);
            //         for (int block_idx=0; block_idx<param_set->rank_set->count; block_idx++){
            //             printf("SENDBUF to rank #%i  : ", block_idx);
            //             for (int idx=0; idx<param_set->sendcount; idx++){
            //                 switch (param_set->send_type_idx){
            //                     case 0:
            //                         printf(" %02x ", ((uint8_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
            //                         break;
            //                     case 1:
            //                         printf(" %04x ", ((uint16_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
            //                         break;
            //                     case 2:
            //                         printf(" %08x ", ((uint32_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
            //                         break;
            //                     case 3:
            //                         printf(" %016x ", ((uint64_t*)sendbuf)[block_idx * param_set->sendcount + idx]);
            //                         break;
            //                 }
            //             }
            //             printf("\n");
            //             fflush(stdout);
            //         }
            //         for (int block_idx=0; block_idx<param_set->rank_set->count; block_idx++){
            //             printf("RECVBUF from rank #%i: ", block_idx);
            //             for (int idx=0; idx<param_set->recvcount; idx++){
            //                 switch (param_set->recv_type_idx){
            //                     case 0:
            //                         printf(" %02x ", ((uint8_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
            //                         break;
            //                     case 1:
            //                         printf(" %04x ", ((uint16_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
            //                         break;
            //                     case 2:
            //                         printf(" %08x ", ((uint32_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
            //                         break;
            //                     case 3:
            //                         printf(" %016x ", ((uint64_t*)recvbuf)[block_idx * param_set->recvcount + idx]);
            //                         break;
            //                 }
            //             }
            //             printf("\n");
            //             fflush(stdout);
            //         }
            //         printf("\n");
            //         fflush(stdout);
            //     }
            //     sleep(1.0);            
            // }
            free(recvbuf);
            free(sendbuf);
        }
    }
    fflush(stdout);


    // test 2 - mix up send and recv counts while keeping block transferred byte size = const.
    if (my_rank == 0){
        printf("\n\nMPI_Alltoall test with nodes having differnt send and receive type\n");
        fflush(stdout);
    }

    param_sets = alltoall_test_individual_node_params_sets(rank_sets);
    // in this test the param set is dependent on the rank
    alltoall_test_node_params_t* param_set = &param_sets[my_rank % 4];  // as we have 4 paramsets to choose from 

    if (is_rank_in_rankset(my_rank, param_set->rank_set)){  // a precaution - using all 8 ranks for this test
        printf("Creating buffer in rank %i\n", my_rank);
        fflush(stdout);
        DEBUG_ALLTOALL_PROFILING("creating sendbuf...\n", NULL);
        void* sendbuf = create_sendbuf(param_set);
        DEBUG_ALLTOALL_PROFILING("created sendbuf\n", NULL);
        void* recvbuf = create_recvbuf(param_set);
        DEBUG_ALLTOALL_PROFILING("created recvbuf\n", NULL);
        // signature: MPI_Alltoall( const void* sendbuf , int sendcount , MPI_Datatype sendtype , void* recvbuf , int recvcount , MPI_Datatype recvtype , MPI_Comm comm);
        DEBUG_ALLTOALL_PROFILING("MPI_UINT8_T = %i\n", MPI_UINT8_T);
        //MPI_Alltoall(sendbuf, 8 , MPI_UINT8_T , recvbuf , 8 , MPI_UINT8_T , MPI_COMM_WORLD);
        DEBUG_ALLTOALL_PROFILING("Done basic MPI_Alltoall\n", NULL);

        // test that my rank is one of the communicator used in this call - if not omit this call
        DEBUG_ALLTOALL_PROFILING("DEBUG driver prog: send type index, value, %i, %i\n", param_set->send_type_idx, MPI_Datatypes_used[param_set->send_type_idx] );
        // note for next line - paramset has been prepared with all items using the same rankset of all 8 nodes
        // printf("Calling alltoall in rank %i\n", my_rank);
        // fflush(stdout);        
        MPI_Alltoall(sendbuf, param_set->sendcount , MPI_Datatypes_used[param_set->send_type_idx] , recvbuf , param_set->recvcount , MPI_Datatypes_used[param_set->recv_type_idx] , param_set->rank_set->communicator);
        printf("Returned from alltoall in rank %i\n", my_rank);
        // fflush(stdout);        
        // print_buffers(my_rank, world_size, param_set, sendbuf, recvbuf);
        free(recvbuf);
        free(sendbuf);
    }
    MPI_Finalize();
}
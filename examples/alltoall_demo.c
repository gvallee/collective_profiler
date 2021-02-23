/* 
A complex test of MPI_Alltoall provding count patterns to be recorded by the samples
which also provides visualisation of hte send and receive buffers
the latter uses sleep to synchronise the nodes contributions making it slow
and so should not be used with automated testing, 
*/ 

#include "alltoall_test_helpers.h"

#define PARAM_SETS_COUNT 10
#define RANK_SETS_COUNT 3

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
        printf("name, value: %s, %i\n", type_strings[i], (uint64_t) MPI_Datatypes_used[i]);
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
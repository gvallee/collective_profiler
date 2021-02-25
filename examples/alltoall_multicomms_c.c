/* 
A simple test of MPI_Alltoall, just one call to that
*/ 

#include "alltoall_test_helpers.h"

#define PARAM_SETS_COUNT 9
#define RANK_SETS_COUNT 9

rank_set_t* create_rank_sets(){
    rank_set_t* rank_sets = malloc(sizeof(rank_set_t) * RANK_SETS_COUNT);  
    rank_set_t new_set0 = { .count= 4, .ranks = {0, 1, 2, 3} };
    rank_sets[0] = new_set0;
    rank_set_t new_set1 = { .count= 3, .ranks = {1, 2, 3} };
    rank_sets[1] = new_set1;
    rank_set_t new_set2 = { .count= 3, .ranks = {0, 2, 3} };
    rank_sets[2] = new_set2;
    rank_set_t new_set3 = { .count= 3, .ranks = {0, 1, 3} };
    rank_sets[3] = new_set3;
    rank_set_t new_set4 = { .count= 3, .ranks = {0, 1, 2} };
    rank_sets[4] = new_set4;
    rank_set_t new_set5 = { .count= 2, .ranks = {1, 2} };
    rank_sets[5] = new_set5;
    rank_set_t new_set6 = { .count= 2, .ranks = {0, 3} };
    rank_sets[6] = new_set6;
    rank_set_t new_set7 = { .count= 2, .ranks = {0, 1} };
    rank_sets[7] = new_set7;
    rank_set_t new_set8 = { .count= 2, .ranks = {2, 3} };
    rank_sets[8] = new_set8;
    return rank_sets;
}

alltoall_test_node_params_t* create_params_sets(rank_set_t* rank_sets){
    alltoall_test_node_params_t* params_sets = (alltoall_test_node_params_t*) malloc(sizeof(alltoall_test_node_params_t) * PARAM_SETS_COUNT);
    params_sets[0] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[0], .repetitions=1};
    params_sets[1] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[1], .repetitions=1};
    params_sets[2] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[2], .repetitions=1};
    params_sets[3] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[3], .repetitions=1};
    params_sets[4] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[4], .repetitions=1};
    params_sets[5] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[5], .repetitions=1};
    params_sets[6] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[6], .repetitions=1};
    params_sets[7] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[7], .repetitions=1};
    params_sets[8] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[8], .repetitions=1};
    return params_sets;
} 


int main(int argc, char *argv[]) {

    // Intialise MPI
    int world_size, my_rank;
    MPI_Init(NULL, NULL);
    MPI_Comm_size(MPI_COMM_WORLD, &world_size);
    MPI_Comm_rank(MPI_COMM_WORLD, &my_rank);

    /* set up alltoall parameter sets and the communicators therefor */
    rank_set_t* rank_sets = create_rank_sets();
    create_communicators(world_size, rank_sets, RANK_SETS_COUNT); 
    alltoall_test_node_params_t* param_sets = create_params_sets(rank_sets);

    /* to test aggregation of patterns this set should have duplicates */
    int param_sets_set_count = 9;
    int param_sets_indices[] = {0, 1, 2, 3, 4, 5,6 ,7, 8};
    
    do_test(param_sets, param_sets_set_count, param_sets_indices, my_rank);

    MPI_Finalize();
}
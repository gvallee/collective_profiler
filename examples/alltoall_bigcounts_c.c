/* 
A simple test of MPI_Alltoall, just one call to that
*/ 

#include "alltoall_test_helpers.h"

#define PARAM_SETS_COUNT 1
#define RANK_SETS_COUNT 1

rank_set_t* create_rank_sets(){
    rank_set_t* rank_sets = malloc(sizeof(rank_set_t) * RANK_SETS_COUNT);  
    rank_set_t new_set0 = { .count= 4, .ranks = {0, 1, 2, 3} };
    rank_sets[0] = new_set0;
    return rank_sets;
}

alltoall_test_node_params_t* create_params_sets(rank_set_t* rank_sets){
    alltoall_test_node_params_t* params_sets = (alltoall_test_node_params_t*) malloc(sizeof(alltoall_test_node_params_t) * PARAM_SETS_COUNT);
    params_sets[0] = (alltoall_test_node_params_t) {.send_type_idx = 2, .recv_type_idx = 2, .sendcount =  16, .recvcount =  16, .rank_set = &rank_sets[0], .repetitions=1000};  /* 1000 for tesing this script - 1E6 desired for full test of sampling" */
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
    int param_sets_set_count = 1;
    int param_sets_indices[] = {0};
    
    do_test(param_sets, param_sets_set_count, param_sets_indices, my_rank);

    MPI_Finalize();
}
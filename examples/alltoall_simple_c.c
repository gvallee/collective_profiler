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
    params_sets[0] = (alltoall_test_node_params_t) {.send_type_idx = 0, .recv_type_idx = 0, .sendcount =  8, .recvcount =  8, .rank_set = &rank_sets[0], .repetitions=1};
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

    /* to test aggregation of patters this set should have duplicates */
    int param_sets_set_count = 1;
    int param_sets_indices[] = {0};
    
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

    MPI_Finalize();
}
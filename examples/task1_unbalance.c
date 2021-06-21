#include <stdio.h>
#include <stdlib.h>
#include <mpi.h>
 
/**
 * @brief Illustrates how to use a variable all to all.
 * @details This application is meant to be run with 3 MPI processes. Each
 * process has an arbitrary number of elements to send and receive, at different
 * positions. To demonstrate the great flexibility of the MPI_Alltoallv routine,
 * the data exchange designed is rather irregular, so it is extra detailed in
 * this description.
 * 
 * It can be described as follows:
 * - Process 0-39:
 *     - has 1MB integers to send, as follows, it sends:
 *         - to process 40-79: 40 * 40 MB
 *     - has 1MB integers to receive, as follows, it sends:
 *         - from process 40-79: 40 * 40 MB
 * - Process 40-79:
 *     - has 1MB integers to send, as follows, it sends:
 *         - to process 0-39: 40 * 40 MB
 *     - has 1MB integers to receive, as follows, it receives:
 *         - from process 0-39: 40 * 40 MB
 * - Process 80-159:
 *     - has nothing to send
 *     - has nothing to recevie
 *
 * In addition to the above, it can be visualised as follows:
 *
 * +---------------------------+ +---------------------------+ +----------------------------+
 * |       Process 0-39       | |       Process 40-79       | |       Process 80-159       |
 * +-------+-------+----------+ +-------+-------+----------+ +-------+-------+-------------+
 * |      Value   | | Value | Value | Value |         | Value |
 * |   0 40 * 40 MB  |  100  |  200  | |  300  |  400  |  500  |         |  600  |
 * +-------+-------+-------+ +-------+-------+-------+         +-------+
 *     |       |       |        |        |       |_________________|_______
 *     |       |       |        |        |_________________________|_      |
 *     |       |       |        |______________________________    | |     |
 *     |       |       |_____________________                  |   | |     |
 *     |       |_______________________      |                 |   | |     | 
 *     |   ____________________________|_____|_________________|___| |     |
 *     |__|_____                       |     |                 |     |     | 
 *        |     |                      |     |                 |     |     | 
 *     +-----+-----+                +-----+-----+           +-----+-----+-----+
 *     | 600 |  0  |                | 100 | 200 |           | 300 | 400 | 500 |
 *  +--+-----+-----+--+         +---+-----+-----+-+         +-----+-----+-----+
 *  |    Process 0    |         |    Process 1    |         |    Process 2    |
 *  +-----------------+         +-----------------+         +-----------------+
 **/
int main(int argc, char* argv[])
{
    MPI_Init(&argc, &argv);
 
    // Get number of processes and check that 3 processes are used
    int size;
    MPI_Comm_size(MPI_COMM_WORLD, &size);
    if(size != 160)
    {
        printf("This application is meant to be run with 3 MPI processes.\n");
        MPI_Abort(MPI_COMM_WORLD, EXIT_FAILURE);
    }
 
    // Get my rank
    int my_rank;
    MPI_Comm_rank(MPI_COMM_WORLD, &my_rank);
 
    // Define the buffer containing the values to send
    int* buffer_send;
    int buffer_send_length = 1024 * 1024 / 4;
    buffer_send = (int *)calloc( buffer_send_length,sizeof(int));
    switch(my_rank-my_rank %40)
    {
        case 0:
            for (int i = 0; i < buffer_send_length; ++i)
                buffer_send[i] = 0;
            printf("Process %d, my values = %d * 1MB.\n", my_rank, buffer_send[0]);
            break;
        case 40:
            for (int i = 0; i < buffer_send_length; ++i)
                buffer_send[i] = 1;
            printf("Process %d, my values = %d * 1MB.\n", my_rank, buffer_send[0]);
            break;
    }
 
    // Define my counts for sending (how many integers do I send to each process?)
    int *counts_send=(int *)calloc( 160,sizeof(int) );
    switch(my_rank-my_rank%40)
    {
        case 0:
            counts_send[my_rank + 40] = 1024 * 1024 / 4;
            break;
        case 40:
            counts_send[my_rank - 40] = 1024 * 1024 / 4;
    }
 
    // Define my displacements for sending (where is located in the buffer each message to send?)
    int displacements_send[1024 * 1024 / 4];
    for (int i = 0; i < 1024 * 1024 / 4; ++i)
        displacements_send[i]=0;
 
    // Define the buffer for reception
    int* buffer_recv;
    int buffer_recv_length;

    buffer_recv_length = 1024 * 1024 / 4;
    buffer_recv = (int *)malloc(sizeof(int) * buffer_recv_length);


    // Define my counts for receiving (how many integers do I receive from each process?)
    int *counts_recv=(int*) calloc(160,sizeof(int));
    switch (my_rank - my_rank % 40) {
    case 0:
        counts_recv[my_rank + 40] = 1024 * 1024 / 4;
        break;
    case 40:
        counts_recv[my_rank - 40] = 1024 * 1024 / 4;
    }

    // Define my displacements for reception (where to store in buffer each message received?)
    int displacements_recv[1024 * 1024 / 4];
    for (int i = 0; i < 1024 * 1024 / 4; ++i)
        displacements_recv[i]=i;
 
    MPI_Alltoallv(buffer_send, counts_send, displacements_send, MPI_INT, buffer_recv, counts_recv, displacements_recv, MPI_INT, MPI_COMM_WORLD);
    
    char* is_debug=getenv("DEBUG");
    if (is_debug!=NULL)
    {printf("Values received on process %d:", my_rank);
    for(int i = 0; i < buffer_recv_length; i++)
    {
        printf(" %d", buffer_recv[i]);
    }
    printf("\n");}
 
    free(buffer_send);
    free(buffer_recv);
 
    MPI_Finalize();
 
    return EXIT_SUCCESS;
}
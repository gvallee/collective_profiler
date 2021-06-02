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
 * - Process 0:
 *     - has 3 integers to send, as follows, it sends:
 *         - to process 0: the first integer
 *         - to process 1: the last 2 integers
 *         - to process 2: nothing
 *     - has 2 integers to receive, as follows, it receives:
 *         - from process 0: 1 integer, stores it at the end
 *         - from process 1: nothing
 *         - from process 2: 1 integer, stores it at the beginning
 * - Process 1:
 *     - has 3 integers to send, as follows, it sends:
 *         - nothing to process 0
 *         - nothing to itself
 *         - 3 integers to process 2
 *     - has 2 integers to receive, as follows, it receives:
 *         - 2 integers rom process 0
 *         - nothing from itself
 *         - nothing from process 2
 * - Process 2:
 *     - has 1 integer to send, as follows, it sends:
 *         - 1 integer to process 0
 *         - nothing to process 1
 *         - nothing to itself
 *     - has 3 integers to receive, as follows, it receives:
 *         - nothing from process 0
 *         - 3 integers from process 1
 *         - nothing from itself
 *
 * In addition to the above, it can be visualised as follows:
 *
 * +-----------------------+ +-----------------------+ +-----------------------+
 * |       Process 0       | |       Process 1       | |       Process 2       |
 * +-------+-------+-------+ +-------+-------+-------+ +-------+-------+-------+
 * | Value | Value | Value | | Value | Value | Value |         | Value |
 * |   0   |  100  |  200  | |  300  |  400  |  500  |         |  600  |
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
        case 80:
            for (int i = 0; i < buffer_send_length; ++i)
                buffer_send[i] = 2;
            printf("Process %d, my values = %d * 1MB.\n", my_rank, buffer_send[0]);
            break;
        case 120:
            for (int i = 0; i < buffer_send_length; ++i)
                buffer_send[i] = 3;
            printf("Process %d, my values = %d * 1MB.\n", my_rank, buffer_send[0]);
            break;
    }
 
    // Define my counts for sending (how many integers do I send to each process?)
    int *counts_send=(int *)calloc( 160,sizeof(int) );
    switch(my_rank-my_rank%20)
    {
        case 0:
            counts_send[my_rank + 40] = 1024 * 1024 / 4;
            counts_send[my_rank + 80] = 1024 * 1024 / 4;
            break;
        case 40:
            counts_send[my_rank + 60] = 1024 * 1024 / 4;
            counts_send[my_rank + 80] = 1024 * 1024 / 4;
            break;
        case 80:
            counts_send[my_rank + 60] = 1024 * 1024 / 4;
            counts_send[my_rank - 80] = 1024 * 1024 / 4;
            break;
        case 120:
            counts_send[my_rank - 100] = 1024 * 1024 / 4;
            counts_send[my_rank - 60] = 1024 * 1024 / 4;
            break;
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
    switch (my_rank - my_rank % 20) {
    case 0:
        counts_recv[my_rank + 80] = 1024 * 1024 / 4;
        break;
    case 20:
        counts_recv[my_rank + 100] = 1024 * 1024 / 4;
        break;
    case 40:
        counts_recv[my_rank - 40] = 1024 * 1024 / 4;
        break;
    case 60:
        counts_recv[my_rank + 60] = 1024 * 1024 / 4;
        break;
    case 80:
        counts_recv[my_rank - 80] = 1024 * 1024 / 4;
        break;
    case 100:
        counts_recv[my_rank - 60] = 1024 * 1024 / 4;
        break;
    case 120:
        counts_recv[my_rank - 80] = 1024 * 1024 / 4;
        break;
    case 140:
        counts_recv[my_rank - 60] = 1024 * 1024 / 4;
        break;
    }

    // Define my displacements for reception (where to store in buffer each message received?)
    int displacements_recv[1024 * 1024 / 4];
    for (int i = 0; i < 1024 * 1024 / 4; ++i)
        displacements_recv[i]=0;
 
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
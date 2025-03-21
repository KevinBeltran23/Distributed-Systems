#include "mpi.h"
#include <stdio.h>

int main(int argc, char *argv[]) {
    int numprocs, rank, chunk_size, i, j;
    int max, mymax, rem;
    int matrix[500][500];
    int local_matrix[500][500];
    int second_matrix[500][500];
    int result[500 * 500];
    int global_result[500 * 500];
    MPI_Status status;

    /* Initialize MPI */
    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &numprocs);
    printf("Hello from process %d of %d \n", rank, numprocs);

    chunk_size = 500 / numprocs;

    if (rank == 0) { /* Only on the root task... */
        /* Initialize Matrix and Vector */
        for (i = 0; i < 500; i++) {
            for (j = 0; j < 500; j++) {
                matrix[i][j] = i + j;
                second_matrix[i][j] = i - j;
            }
        }
    }

    /* Distribute Matrix */
    /* The matrix is just sent to everyone */
    MPI_Bcast(second_matrix, 500 * 500, MPI_INT, 0, MPI_COMM_WORLD);

    /* Distribute Matrix */
    /* Assume the matrix is too big to broadcast. Send blocks of rows to each task,
       nrows/nprocs to each one */
    MPI_Scatter(matrix, 500 * chunk_size, MPI_INT, local_matrix, 500 * chunk_size, MPI_INT, 0, MPI_COMM_WORLD);

    /* Each processor has a chunk of rows, now multiply and build a part of the solution matrix */
    for (i = 0; i < chunk_size; i++) {
        for (j = 0; j < 500; j++) {
            result[i * 500 + j] = 0; // Initialize the result element
            for (int k = 0; k < 500; k++) {
                result[i * 500 + j] += local_matrix[i][k] * matrix[k][j]; // Matrix multiplication
            }
        }
    }

    /* Send result back to master */
    MPI_Gather(result, chunk_size * 500, MPI_INT, global_result, chunk_size * 500, MPI_INT, 0, MPI_COMM_WORLD);

    /* Display result */
    if (rank == 0) {
        for (i = 0; i < 500; i++) {
            for (j = 0; j < 500; j++) {
                printf(" %d \t ", global_result[i * 500 + j]);
            }
            printf("\n");
        }
    }

    MPI_Finalize();
    return 0;
}

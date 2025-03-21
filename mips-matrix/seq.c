#include <stdio.h>
#include <stdlib.h>
#include <time.h>

int main() {
    int m, p, q;
    
    // Seed random number generator
    srand(time(NULL));
    
    // Input matrix dimensions
    printf("Random Matrix Multiplication Program\n");
    printf("------------------------------------\n");
    
    printf("Enter the number of rows and columns for the first matrix: ");
    scanf("%d %d", &m, &p);
    
    printf("Enter the number of columns for the second matrix: ");
    scanf("%d", &q);
    
    int first[m][p], second[p][q], multiply[m][q];

    // Generate first matrix with random values
    printf("\nGenerating first matrix (%d x %d) with random values...\n", m, p);
    for (int c = 0; c < m; c++) {
        for (int k = 0; k < p; k++) {
            first[c][k] = (rand() % 10) + 1; // Random numbers from 1 to 10
        }
    }

    // Generate second matrix with random values
    printf("\nGenerating second matrix (%d x %d) with random values...\n", p, q);
    for (int k = 0; k < p; k++) {
        for (int d = 0; d < q; d++) {
            second[k][d] = (rand() % 10) + 1;
        }
    }

    // Matrix multiplication
    for (int c = 0; c < m; c++) {
        for (int d = 0; d < q; d++) {
            int sum = 0;
            for (int k = 0; k < p; k++) {
                sum += first[c][k] * second[k][d];
            }
            multiply[c][d] = sum;
        }
    }

    // Display matrices
    printf("\nFirst Matrix:\n");
    for (int c = 0; c < m; c++) {
        for (int k = 0; k < p; k++) {
            printf("%4d ", first[c][k]); // Adjusts spacing for alignment
        }
        printf("\n");
    }

    printf("\nSecond Matrix:\n");
    for (int k = 0; k < p; k++) {
        for (int d = 0; d < q; d++) {
            printf("%4d ", second[k][d]);
        }
        printf("\n");
    }

    // Output result matrix
    printf("\nResultant Matrix (Multiplication Output):\n");
    for (int c = 0; c < m; c++) {
        for (int d = 0; d < q; d++) {
            printf("%4d ", multiply[c][d]);
        }
        printf("\n");
    }

    printf("\nMatrix multiplication completed successfully!\n");

    return 0;
}

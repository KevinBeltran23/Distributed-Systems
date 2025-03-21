#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <mpi.h>

// Easily adjustable parameters via macros
#define MATRIX_SIZE 1000        
#define KERNEL_SIZE 5           
#define NUM_ITERATIONS 10       
#define SPEED 2.0               

// Function declarations
void initializeMatrix(float *matrix, int rows, int cols);
void initializeKernel(float *kernel, int kernelSize);
void computeRowDistribution(int myRank, int numProcs, float *data, float *kernel, int rows, int cols, int kernelSize, int iterations);
void printMatrix(float *matrix, int rows, int cols);

int main(int argc, char *argv[]) {
    int myRank, numProcs;
    double startTime, endTime;
    
    // MPI Initialization
    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &myRank);
    MPI_Comm_size(MPI_COMM_WORLD, &numProcs);
    
    // Parse command line arguments
    int rows = MATRIX_SIZE;
    int cols = MATRIX_SIZE;
    int kernelSize = KERNEL_SIZE;
    int iterations = NUM_ITERATIONS;
    
    if (argc > 1) rows = cols = atoi(argv[1]);
    if (argc > 2) kernelSize = atoi(argv[2]);
    if (argc > 3) iterations = atoi(argv[3]);
    
    // Make sure kernel size is odd
    if (kernelSize % 2 == 0) kernelSize++;
    
    // Allocate matrix and kernel
    float *data = NULL;
    float *kernel = NULL;
    
    if (myRank == 0) {
        // Only master process initializes full data
        data = (float *)malloc(rows * cols * sizeof(float));
        
        // Initialize data
        initializeMatrix(data, rows, cols);
        
        printf("Matrix size: %d x %d, Kernel size: %d x %d, Iterations: %d\n", 
               rows, cols, kernelSize, kernelSize, iterations);
        
        // Print initial matrix preview
        printf("Initial Matrix:\n");
        printMatrix(data, rows, cols);
    }
    
    // All processes need the kernel
    kernel = (float *)malloc(kernelSize * kernelSize * sizeof(float));
    
    // Only master initializes the kernel
    if (myRank == 0) {
        initializeKernel(kernel, kernelSize);
        
        // Print kernel
        printf("Kernel (%dx%d):\n", kernelSize, kernelSize);
        for (int i = 0; i < kernelSize; i++) {
            for (int j = 0; j < kernelSize; j++) {
                printf("%.4f ", kernel[i*kernelSize + j]);
            }
            printf("\n");
        }
        printf("\n");
    }
    
    // Broadcast kernel to all processes
    MPI_Bcast(kernel, kernelSize * kernelSize, MPI_FLOAT, 0, MPI_COMM_WORLD);
    
    // Synchronize all processes before starting
    MPI_Barrier(MPI_COMM_WORLD);
    startTime = MPI_Wtime();
    
    // Execute row distribution algorithm
    computeRowDistribution(myRank, numProcs, data, kernel, rows, cols, kernelSize, iterations);
    
    // Measure time
    MPI_Barrier(MPI_COMM_WORLD);
    endTime = MPI_Wtime();
    
    if (myRank == 0) {
        printf("Row-based distributed computation completed\n");
        printf("Execution time: %f seconds\n", endTime - startTime);
        
        // Print result matrix preview
        printf("Result Matrix:\n");
        printMatrix(data, rows, cols);
    }
    
    // Clean up
    if (data) free(data);
    if (kernel) free(kernel);
    
    MPI_Finalize();
    return 0;
}

// Initialize matrix with some values
void initializeMatrix(float *matrix, int rows, int cols) {
    for (int i = 0; i < rows; i++) {
        for (int j = 0; j < cols; j++) {
            matrix[i*cols + j] = (float)(i + j) / (rows + cols);
        }
    }
}

// Initialize convolution kernel
void initializeKernel(float *kernel, int kernelSize) {
    int center = kernelSize / 2;
    for (int i = 0; i < kernelSize; i++) {
        for (int j = 0; j < kernelSize; j++) {
            // Simple Gaussian-like kernel
            float distSq = (i - center) * (i - center) + (j - center) * (j - center);
            kernel[i*kernelSize + j] = exp(-distSq / (2.0f * center * center));
        }
    }
    
    // Normalize kernel (sum = 1.0)
    float sum = 0.0f;
    for (int i = 0; i < kernelSize * kernelSize; i++) {
        sum += kernel[i];
    }
    for (int i = 0; i < kernelSize * kernelSize; i++) {
        kernel[i] /= sum;
    }
}

// Row-based distributed convolution
void computeRowDistribution(int myRank, int numProcs, float *data, float *kernel, int rows, int cols, int kernelSize, int iterations) {
    int halfKernel = kernelSize / 2;
    
    // Calculate each process's row count
    int myRows = rows / numProcs;
    int startRow = myRank * myRows;
    int endRow = (myRank == numProcs - 1) ? rows : (myRank + 1) * myRows;
    myRows = endRow - startRow;
    
    // Allocate buffers for local data
    float *myData = (float *)malloc(myRows * cols * sizeof(float));
    float *myBuffer = (float *)malloc(myRows * cols * sizeof(float));
    
    // Buffers for exchanging boundary rows
    float *prevRow = (float *)malloc(cols * sizeof(float));
    float *nextRow = (float *)malloc(cols * sizeof(float));
    
    // Distribute data among processes
    MPI_Scatter(data, myRows * cols, MPI_FLOAT, 
               myData, myRows * cols, MPI_FLOAT, 
               0, MPI_COMM_WORLD);
    
    // Extract coefficients from the kernel for 5-point stencil
    // We'll use center, top, bottom, left, right values from the kernel
    float ccenter = kernel[halfKernel*kernelSize + halfKernel];
    float ctop = kernel[(halfKernel-1)*kernelSize + halfKernel];
    float cbottom = kernel[(halfKernel+1)*kernelSize + halfKernel];
    float cwest = kernel[halfKernel*kernelSize + (halfKernel-1)];
    float ceast = kernel[halfKernel*kernelSize + (halfKernel+1)];
    
    // Main iteration loop
    for (int iter = 0; iter < iterations; iter++) {
        // Initialize requests for non-blocking communication
        MPI_Request requests[4];
        MPI_Status statuses[4];
        int requestCount = 0;
        
        // Exchange boundary rows
        if (myRank > 0) {
            // Send first row to previous process
            MPI_Isend(myData, cols, MPI_FLOAT, 
                     myRank - 1, 0, MPI_COMM_WORLD, &requests[requestCount++]);
            // Receive last row from previous process
            MPI_Irecv(prevRow, cols, MPI_FLOAT, 
                     myRank - 1, 1, MPI_COMM_WORLD, &requests[requestCount++]);
        }
        
        if (myRank < numProcs - 1) {
            // Send last row to next process
            MPI_Isend(&myData[(myRows-1)*cols], cols, MPI_FLOAT, 
                     myRank + 1, 1, MPI_COMM_WORLD, &requests[requestCount++]);
            // Receive first row from next process
            MPI_Irecv(nextRow, cols, MPI_FLOAT, 
                     myRank + 1, 0, MPI_COMM_WORLD, &requests[requestCount++]);
        }
        
        // Calculate interior points while waiting for communication
        for (int i = 1; i < myRows - 1; i++) {
            for (int j = 1; j < cols - 1; j++) {
                // Apply 5-point stencil
                myBuffer[i*cols + j] = myData[i*cols + j] + 
                    (cbottom * myData[(i+1)*cols + j] + 
                     cwest * myData[i*cols + j-1] +
                     ceast * myData[i*cols + j+1] + 
                     ctop * myData[(i-1)*cols + j]) / SPEED;
            }
        }
        
        // Wait for communication to complete
        MPI_Waitall(requestCount, requests, statuses);
        
        // Calculate first row (depends on data from previous process)
        if (myRank > 0) {
            for (int j = 1; j < cols - 1; j++) {
                myBuffer[j] = myData[j] + 
                    (cbottom * myData[cols + j] + 
                     cwest * myData[j-1] +
                     ceast * myData[j+1] + 
                     ctop * prevRow[j]) / SPEED;
            }
        }
        
        // Calculate last row (depends on data from next process)
        if (myRank < numProcs - 1) {
            for (int j = 1; j < cols - 1; j++) {
                myBuffer[(myRows-1)*cols + j] = myData[(myRows-1)*cols + j] +
                    (cbottom * nextRow[j] +
                     cwest * myData[(myRows-1)*cols + j-1] +
                     ceast * myData[(myRows-1)*cols + j+1] +
                     ctop * myData[(myRows-2)*cols + j]) / SPEED;
            }
        }
        
        // Swap buffers for next iteration
        float *temp = myData;
        myData = myBuffer;
        myBuffer = temp;
    }
    
    // Gather results back to master
    MPI_Gather(myData, myRows * cols, MPI_FLOAT, 
              data, myRows * cols, MPI_FLOAT, 
              0, MPI_COMM_WORLD);
    
    // Free local resources
    free(myData);
    free(myBuffer);
    free(prevRow);
    free(nextRow);
}

// Print a small portion of the matrix for verification
void printMatrix(float *matrix, int rows, int cols) {
    int printSize = 5;
    printf("Matrix preview (top-left %dx%d):\n", printSize, printSize);
    for (int i = 0; i < printSize && i < rows; i++) {
        for (int j = 0; j < printSize && j < cols; j++) {
            printf("%.4f ", matrix[i*cols + j]);
        }
        printf("\n");
    }
    printf("\n");
}
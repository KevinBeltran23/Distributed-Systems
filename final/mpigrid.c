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
void computeTileDistribution(int myRank, int numProcs, float *data, float *kernel, int rows, int cols, int kernelSize, int iterations);
void printMatrix(float *matrix, int rows, int cols);
void exchangeGhostCells(int myRank, int numProcs, int procRows, int procCols, 
                        int myRow, int myCol, float *myTile, int tileRows, int tileCols);

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
    
    // Execute tile distribution algorithm
    computeTileDistribution(myRank, numProcs, data, kernel, rows, cols, kernelSize, iterations);
    
    // Measure time
    MPI_Barrier(MPI_COMM_WORLD);
    endTime = MPI_Wtime();
    
    if (myRank == 0) {
        printf("Tile-based distributed computation completed\n");
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

// Tile-based distributed convolution
void computeTileDistribution(int myRank, int numProcs, float *data, float *kernel, int rows, int cols, int kernelSize, int iterations) {
    int halfKernel = kernelSize / 2;
    
    // Determine grid size for processes
    int procRows, procCols;
    // Find the most square-like decomposition of numProcs
    procRows = (int)sqrt(numProcs);
    while (numProcs % procRows != 0) {
        procRows--;
    }
    procCols = numProcs / procRows;
    
    // Determine my position in the process grid
    int myRow = myRank / procCols;
    int myCol = myRank % procCols;
    
    // Calculate my tile size
    int tileRows = rows / procRows;
    int tileCols = cols / procCols;
    
    // Handle remainder rows/cols
    if (myRow == procRows - 1) {
        tileRows = rows - (procRows - 1) * tileRows;
    }
    if (myCol == procCols - 1) {
        tileCols = cols - (procCols - 1) * tileCols;
    }
    
    // Calculate my global position
    int startRow = myRow * (rows / procRows);
    int startCol = myCol * (cols / procCols);
    
    // Allocate buffers for local tile including ghost cells for neighbors
    float *myTile = (float *)malloc((tileRows + 2) * (tileCols + 2) * sizeof(float));
    float *myBuffer = (float *)malloc((tileRows + 2) * (tileCols + 2) * sizeof(float));
    
    // Initialize tile to zeros (for ghost cells)
    memset(myTile, 0, (tileRows + 2) * (tileCols + 2) * sizeof(float));
    memset(myBuffer, 0, (tileRows + 2) * (tileCols + 2) * sizeof(float));
    
    // Distribute tiles from master
    if (myRank == 0) {
        // Master keeps its own tile
        for (int i = 0; i < tileRows; i++) {
            for (int j = 0; j < tileCols; j++) {
                myTile[(i+1)*(tileCols+2) + (j+1)] = data[i*cols + j];
            }
        }
        
        // Send tiles to other processes
        for (int p = 1; p < numProcs; p++) {
            int pRow = p / procCols;
            int pCol = p % procCols;
            int pStartRow = pRow * (rows / procRows);
            int pStartCol = pCol * (cols / procCols);
            int pTileRows = (pRow == procRows - 1) ? (rows - (procRows - 1) * (rows / procRows)) : (rows / procRows);
            int pTileCols = (pCol == procCols - 1) ? (cols - (procCols - 1) * (cols / procCols)) : (cols / procCols);
            
            // Pack the tile data
            float *tempBuffer = (float *)malloc(pTileRows * pTileCols * sizeof(float));
            for (int i = 0; i < pTileRows; i++) {
                for (int j = 0; j < pTileCols; j++) {
                    tempBuffer[i*pTileCols + j] = data[(pStartRow + i)*cols + (pStartCol + j)];
                }
            }
            
            MPI_Send(tempBuffer, pTileRows * pTileCols, MPI_FLOAT, p, 0, MPI_COMM_WORLD);
            free(tempBuffer);
        }
    } else {
        // Receive my tile from master
        float *tempBuffer = (float *)malloc(tileRows * tileCols * sizeof(float));
        MPI_Recv(tempBuffer, tileRows * tileCols, MPI_FLOAT, 0, 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        
        // Copy to myTile leaving space for ghost cells
        for (int i = 0; i < tileRows; i++) {
            for (int j = 0; j < tileCols; j++) {
                myTile[(i+1)*(tileCols+2) + (j+1)] = tempBuffer[i*tileCols + j];
            }
        }
        
        free(tempBuffer);
    }
    
    // Extract coefficients from the kernel for 5-point stencil
    float ccenter = kernel[halfKernel*kernelSize + halfKernel];
    float ctop = kernel[(halfKernel-1)*kernelSize + halfKernel];
    float cbottom = kernel[(halfKernel+1)*kernelSize + halfKernel];
    float cwest = kernel[halfKernel*kernelSize + (halfKernel-1)];
    float ceast = kernel[halfKernel*kernelSize + (halfKernel+1)];
    
    // Main iteration loop
    for (int iter = 0; iter < iterations; iter++) {
        // Exchange ghost cells with neighbors
        exchangeGhostCells(myRank, numProcs, procRows, procCols, 
                          myRow, myCol, myTile, tileRows, tileCols);
        
        // Compute interior points (no dependency on ghost cells)
        for (int i = 2; i < tileRows; i++) {
            for (int j = 2; j < tileCols; j++) {
                // Apply 5-point stencil
                myBuffer[i*(tileCols+2) + j] = myTile[i*(tileCols+2) + j] + 
                    (cbottom * myTile[(i+1)*(tileCols+2) + j] + 
                     cwest * myTile[i*(tileCols+2) + (j-1)] +
                     ceast * myTile[i*(tileCols+2) + (j+1)] + 
                     ctop * myTile[(i-1)*(tileCols+2) + j]) / SPEED;
            }
        }
        
        // Compute boundary points (depend on ghost cells)
        // First row
        if (myRow > 0) {
            for (int j = 1; j < tileCols + 1; j++) {
                myBuffer[1*(tileCols+2) + j] = myTile[1*(tileCols+2) + j] + 
                    (cbottom * myTile[2*(tileCols+2) + j] + 
                     cwest * myTile[1*(tileCols+2) + (j-1)] +
                     ceast * myTile[1*(tileCols+2) + (j+1)] + 
                     ctop * myTile[0*(tileCols+2) + j]) / SPEED;
            }
        }
        
        // Last row
        if (myRow < procRows - 1) {
            for (int j = 1; j < tileCols + 1; j++) {
                myBuffer[tileRows*(tileCols+2) + j] = myTile[tileRows*(tileCols+2) + j] + 
                    (cbottom * myTile[(tileRows+1)*(tileCols+2) + j] + 
                     cwest * myTile[tileRows*(tileCols+2) + (j-1)] +
                     ceast * myTile[tileRows*(tileCols+2) + (j+1)] + 
                     ctop * myTile[(tileRows-1)*(tileCols+2) + j]) / SPEED;
            }
        }
        
        // Left column
        if (myCol > 0) {
            for (int i = 1; i < tileRows + 1; i++) {
                myBuffer[i*(tileCols+2) + 1] = myTile[i*(tileCols+2) + 1] + 
                    (cbottom * myTile[(i+1)*(tileCols+2) + 1] + 
                     cwest * myTile[i*(tileCols+2) + 0] +
                     ceast * myTile[i*(tileCols+2) + 2] + 
                     ctop * myTile[(i-1)*(tileCols+2) + 1]) / SPEED;
            }
        }
        
        // Right column
        if (myCol < procCols - 1) {
            for (int i = 1; i < tileRows + 1; i++) {
                myBuffer[i*(tileCols+2) + tileCols] = myTile[i*(tileCols+2) + tileCols] + 
                    (cbottom * myTile[(i+1)*(tileCols+2) + tileCols] + 
                     cwest * myTile[i*(tileCols+2) + (tileCols-1)] +
                     ceast * myTile[i*(tileCols+2) + (tileCols+1)] + 
                     ctop * myTile[(i-1)*(tileCols+2) + tileCols]) / SPEED;
            }
        }
        
        // Swap buffers for next iteration
        float *temp = myTile;
        myTile = myBuffer;
        myBuffer = temp;
    }
    
    // Gather results back to master
    if (myRank == 0) {
        // Extract my own tile result (without ghost cells)
        for (int i = 0; i < tileRows; i++) {
            for (int j = 0; j < tileCols; j++) {
                data[i*cols + j] = myTile[(i+1)*(tileCols+2) + (j+1)];
            }
        }
        
        // Receive tiles from other processes
        for (int p = 1; p < numProcs; p++) {
            int pRow = p / procCols;
            int pCol = p % procCols;
            int pStartRow = pRow * (rows / procRows);
            int pStartCol = pCol * (cols / procCols);
            int pTileRows = (pRow == procRows - 1) ? (rows - (procRows - 1) * (rows / procRows)) : (rows / procRows);
            int pTileCols = (pCol == procCols - 1) ? (cols - (procCols - 1) * (cols / procCols)) : (cols / procCols);
            
            float *tempBuffer = (float *)malloc(pTileRows * pTileCols * sizeof(float));
            MPI_Recv(tempBuffer, pTileRows * pTileCols, MPI_FLOAT, p, 2, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
            
            // Copy data to the right position in the global matrix
            for (int i = 0; i < pTileRows; i++) {
                for (int j = 0; j < pTileCols; j++) {
                    data[(pStartRow + i)*cols + (pStartCol + j)] = tempBuffer[i*pTileCols + j];
                }
            }
            
            free(tempBuffer);
        }
    } else {
        // Send my tile result to master (without ghost cells)
        float *tempBuffer = (float *)malloc(tileRows * tileCols * sizeof(float));
        for (int i = 0; i < tileRows; i++) {
            for (int j = 0; j < tileCols; j++) {
                tempBuffer[i*tileCols + j] = myTile[(i+1)*(tileCols+2) + (j+1)];
            }
        }
        
        MPI_Send(tempBuffer, tileRows * tileCols, MPI_FLOAT, 0, 2, MPI_COMM_WORLD);
        free(tempBuffer);
    }
    
    // Free local buffers
    free(myTile);
    free(myBuffer);
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

// Ghost cell exchange function - simplifies the communication pattern
void exchangeGhostCells(int myRank, int numProcs, int procRows, int procCols, 
                        int myRow, int myCol, float *myTile, int tileRows, int tileCols) {
    MPI_Request requests[8];  // Up to 8 requests (4 sends, 4 receives)
    MPI_Status statuses[8];
    int requestCount = 0;
    
    // Buffers for non-contiguous column data
    float *westCol = NULL, *eastCol = NULL;
    float *westGhost = NULL, *eastGhost = NULL;
    
    // North neighbor
    if (myRow > 0) {
        int northRank = (myRow - 1) * procCols + myCol;
        // Send my first row, receive into ghost row
        MPI_Isend(&myTile[1*(tileCols+2) + 1], tileCols, MPI_FLOAT, 
                 northRank, 0, MPI_COMM_WORLD, &requests[requestCount++]);
        MPI_Irecv(&myTile[0*(tileCols+2) + 1], tileCols, MPI_FLOAT, 
                 northRank, 1, MPI_COMM_WORLD, &requests[requestCount++]);
    }
    
    // South neighbor
    if (myRow < procRows - 1) {
        int southRank = (myRow + 1) * procCols + myCol;
        // Send my last row, receive into ghost row
        MPI_Isend(&myTile[tileRows*(tileCols+2) + 1], tileCols, MPI_FLOAT, 
                 southRank, 1, MPI_COMM_WORLD, &requests[requestCount++]);
        MPI_Irecv(&myTile[(tileRows+1)*(tileCols+2) + 1], tileCols, MPI_FLOAT, 
                 southRank, 0, MPI_COMM_WORLD, &requests[requestCount++]);
    }
    
    // West neighbor
    if (myCol > 0) {
        int westRank = myRow * procCols + (myCol - 1);
        // Create west column buffer to send
        westCol = (float *)malloc(tileRows * sizeof(float));
        westGhost = (float *)malloc(tileRows * sizeof(float));
        
        for (int i = 0; i < tileRows; i++) {
            westCol[i] = myTile[(i+1)*(tileCols+2) + 1];
        }
        
        // Send and receive west column 
        MPI_Isend(westCol, tileRows, MPI_FLOAT, westRank, 2, MPI_COMM_WORLD, 
                 &requests[requestCount++]);
        MPI_Irecv(westGhost, tileRows, MPI_FLOAT, westRank, 3, MPI_COMM_WORLD, 
                 &requests[requestCount++]);
    }
    
    // East neighbor 
    if (myCol < procCols - 1) {
        int eastRank = myRow * procCols + (myCol + 1);
        // Create east column buffer to send
        eastCol = (float *)malloc(tileRows * sizeof(float));
        eastGhost = (float *)malloc(tileRows * sizeof(float));
        
        for (int i = 0; i < tileRows; i++) {
            eastCol[i] = myTile[(i+1)*(tileCols+2) + tileCols];
        }
        
        // Send and receive east column
        MPI_Isend(eastCol, tileRows, MPI_FLOAT, eastRank, 3, MPI_COMM_WORLD, 
                 &requests[requestCount++]);
        MPI_Irecv(eastGhost, tileRows, MPI_FLOAT, eastRank, 2, MPI_COMM_WORLD, 
                 &requests[requestCount++]);
    }
    
    // Wait for all communication to complete
    MPI_Waitall(requestCount, requests, statuses);
    
    // Now copy the received ghost cells into place
    if (myCol > 0) {
        for (int i = 0; i < tileRows; i++) {
            myTile[(i+1)*(tileCols+2) + 0] = westGhost[i];
        }
        free(westCol);
        free(westGhost);
    }
    
    if (myCol < procCols - 1) {
        for (int i = 0; i < tileRows; i++) {
            myTile[(i+1)*(tileCols+2) + (tileCols+1)] = eastGhost[i];
        }
        free(eastCol);
        free(eastGhost);
    }
} 
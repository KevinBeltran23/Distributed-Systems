#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <time.h>

// Easily adjustable parameters via macros
#define MATRIX_SIZE 1000        
#define KERNEL_SIZE 5            
#define NUM_ITERATIONS 10        
#define SPEED 2.0               

// Function declarations
void initializeMatrix(float *matrix, int rows, int cols);
void initializeKernel(float *kernel, int kernelSize);
void computeSequential(float *data, float *result, float *kernel, int rows, int cols, int kernelSize, int iterations);
void printMatrix(float *matrix, int rows, int cols);
void testSequential();

int main(int argc, char *argv[]) {
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
    float *data = (float *)malloc(rows * cols * sizeof(float));
    float *result = (float *)malloc(rows * cols * sizeof(float));
    float *kernel = (float *)malloc(kernelSize * kernelSize * sizeof(float));
    
    // Initialize data and kernel
    initializeMatrix(data, rows, cols);
    initializeKernel(kernel, kernelSize);
    
    printf("Matrix size: %d x %d, Kernel size: %d x %d, Iterations: %d\n", 
           rows, cols, kernelSize, kernelSize, iterations);
    
    // Run tests with smaller sizes if requested
    if (rows == 0) {
        testSequential();
        return 0;
    }
    
    // Print initial matrix preview
    printf("Initial Matrix:\n");
    printMatrix(data, rows, cols);
    
    // Measure time
    clock_t start = clock();
    
    // Run sequential computation
    computeSequential(data, result, kernel, rows, cols, kernelSize, iterations);
    
    // Calculate elapsed time
    clock_t end = clock();
    double elapsed = (double)(end - start) / CLOCKS_PER_SEC;
    
    printf("Sequential computation completed\n");
    printf("Execution time: %f seconds\n", elapsed);
    
    // Print result matrix preview
    printf("Result Matrix:\n");
    printMatrix(data, rows, cols);
    
    // Clean up
    free(data);
    free(result);
    free(kernel);
    
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

// Sequential convolution with 5-point stencil (modified to match reference code)
void computeSequential(float *data, float *result, float *kernel, int rows, int cols, int kernelSize, int iterations) {
    int halfKernel = kernelSize / 2;
    float *src = data;
    float *dst = result;
    
    // Extract coefficients from the kernel for 5-point stencil
    float ccenter = kernel[halfKernel*kernelSize + halfKernel];
    float ctop = kernel[(halfKernel-1)*kernelSize + halfKernel];
    float cbottom = kernel[(halfKernel+1)*kernelSize + halfKernel];
    float cwest = kernel[halfKernel*kernelSize + (halfKernel-1)];
    float ceast = kernel[halfKernel*kernelSize + (halfKernel+1)];
    
    for (int iter = 0; iter < iterations; iter++) {
        // Process all elements using 5-point stencil
        for (int y = 0; y < rows; y++) {
            for (int x = 0; x < cols; x++) {
                int center = y * cols + x;
                int west = (x == 0) ? center : center - 1;
                int east = (x == cols - 1) ? center : center + 1;
                int top = (y == 0) ? center : center - cols;
                int bottom = (y == rows - 1) ? center : center + cols;
                
                dst[center] = src[center] + 
                              (ctop * src[top] +
                               cbottom * src[bottom] +
                               cwest * src[west] +
                               ceast * src[east]) / SPEED;
            }
        }
        
        // Swap buffers for next iteration
        float *temp = src;
        src = dst;
        dst = temp;
    }
    
    // If iterations is odd, copy result back to data
    if (iterations % 2 == 1) {
        memcpy(data, src, rows * cols * sizeof(float));
    }
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

// Test the sequential implementation with different kernel sizes
void testSequential() {
    printf("\n--- Testing Sequential Implementation ---\n");
    
    // Test with different matrix sizes
    int testSizes[] = {50, 100};
    // Test with different kernel sizes
    int testKernels[] = {3, 5};
    
    for (int s = 0; s < sizeof(testSizes)/sizeof(testSizes[0]); s++) {
        int rows = testSizes[s];
        int cols = testSizes[s];
        
        for (int k = 0; k < sizeof(testKernels)/sizeof(testKernels[0]); k++) {
            int kernelSize = testKernels[k];
            int iterations = 2;
            
            printf("\nTest with %dx%d matrix, %dx%d kernel, %d iterations\n", 
                   rows, cols, kernelSize, kernelSize, iterations);
            
            // Allocate memory
            float *data = (float *)malloc(rows * cols * sizeof(float));
            float *result = (float *)malloc(rows * cols * sizeof(float));
            float *kernel = (float *)malloc(kernelSize * kernelSize * sizeof(float));
            
            // Initialize data and kernel
            initializeMatrix(data, rows, cols);
            initializeKernel(kernel, kernelSize);
            
            printf("Initial Matrix:\n");
            printMatrix(data, rows, cols);
            
            printf("Kernel (%dx%d):\n", kernelSize, kernelSize);
            for (int i = 0; i < kernelSize; i++) {
                for (int j = 0; j < kernelSize; j++) {
                    printf("%.4f ", kernel[i*kernelSize + j]);
                }
                printf("\n");
            }
            printf("\n");
            
            // Run sequential convolution
            clock_t start = clock();
            computeSequential(data, result, kernel, rows, cols, kernelSize, iterations);
            clock_t end = clock();
            double elapsed = (double)(end - start) / CLOCKS_PER_SEC;
            
            printf("Result Matrix after convolution:\n");
            printMatrix(data, rows, cols);
            printf("Computation time: %f seconds\n", elapsed);
            
            // Clean up
            free(data);
            free(result);
            free(kernel);
        }
    }
    
    printf("--- Sequential Testing Complete ---\n\n");
} 
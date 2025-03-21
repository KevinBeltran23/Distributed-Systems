#!/bin/bash

# Create results directory if it doesn't exist
mkdir -p benchmark_results

# Display system information
echo "=== System Information ===" | tee benchmark_results/system_info.log
hostname | tee -a benchmark_results/system_info.log
lscpu | grep "Model name" | tee -a benchmark_results/system_info.log
echo "" | tee -a benchmark_results/system_info.log

# Array sizes to test
ROWS=1000
KERNEL_SIZE=5
ITERATIONS=10

echo "=== Benchmark Configuration ===" | tee benchmark_results/config.log
echo "Matrix Size: ${ROWS}x${ROWS}" | tee -a benchmark_results/config.log
echo "Kernel Size: ${KERNEL_SIZE}x${KERNEL_SIZE}" | tee -a benchmark_results/config.log
echo "Iterations: ${ITERATIONS}" | tee -a benchmark_results/config.log
echo "" | tee -a benchmark_results/config.log

# Compile the code
make clean
make all

# Run tests with 1 node and save output
echo "Running tests with 1 node..."
echo "=== Sequential Implementation (1 process) ===" > benchmark_results/test_1.log
./sequential ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_1.log 2>&1
echo "" >> benchmark_results/test_1.log
echo "=== Row-Based Implementation (1 process) ===" >> benchmark_results/test_1.log
mpiexec -n 1 ./mpirow ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_1.log 2>&1
echo "" >> benchmark_results/test_1.log
echo "=== Tile-Based Implementation (1 process) ===" >> benchmark_results/test_1.log
mpiexec -n 1 ./mpigrid ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_1.log 2>&1
echo "Tests with 1 node completed."

# Run tests with 2 nodes and save output
echo "Running tests with 2 nodes..."
echo "=== Sequential Implementation (1 process) ===" > benchmark_results/test_2.log
./sequential ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_2.log 2>&1
echo "" >> benchmark_results/test_2.log
echo "=== Row-Based Implementation (2 processes) ===" >> benchmark_results/test_2.log
mpiexec -n 2 ./mpirow ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_2.log 2>&1
echo "" >> benchmark_results/test_2.log
echo "=== Tile-Based Implementation (2 processes) ===" >> benchmark_results/test_2.log
mpiexec -n 2 ./mpigrid ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_2.log 2>&1
echo "Tests with 2 nodes completed."

# Run tests with 4 nodes and save output
echo "Running tests with 4 nodes..."
echo "=== Sequential Implementation (1 process) ===" > benchmark_results/test_4.log
./sequential ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_4.log 2>&1
echo "" >> benchmark_results/test_4.log
echo "=== Row-Based Implementation (4 processes) ===" >> benchmark_results/test_4.log
mpiexec -n 4 ./mpirow ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_4.log 2>&1
echo "" >> benchmark_results/test_4.log
echo "=== Tile-Based Implementation (4 processes) ===" >> benchmark_results/test_4.log
mpiexec -n 4 ./mpigrid ${ROWS} ${KERNEL_SIZE} ${ITERATIONS} >> benchmark_results/test_4.log 2>&1
echo "Tests with 4 nodes completed."

# Extract and format timing results for easy comparison
echo "=== Performance Summary ===" > benchmark_results/summary.log
echo "Implementation,Nodes,Time(s)" >> benchmark_results/summary.log

# Extract sequential times
seq_time=$(grep "Execution time:" benchmark_results/test_1.log | head -1 | awk '{print $3}')
echo "Sequential,1,$seq_time" >> benchmark_results/summary.log

# Extract row-based MPI times
row_time_1=$(grep "Execution time:" benchmark_results/test_1.log | grep -A 1 "Row-Based" | tail -1 | awk '{print $3}')
row_time_2=$(grep "Execution time:" benchmark_results/test_2.log | grep -A 1 "Row-Based" | tail -1 | awk '{print $3}')
row_time_4=$(grep "Execution time:" benchmark_results/test_4.log | grep -A 1 "Row-Based" | tail -1 | awk '{print $3}')
echo "Row-Based,1,$row_time_1" >> benchmark_results/summary.log
echo "Row-Based,2,$row_time_2" >> benchmark_results/summary.log
echo "Row-Based,4,$row_time_4" >> benchmark_results/summary.log

# Extract tile-based MPI times
grid_time_1=$(grep "Execution time:" benchmark_results/test_1.log | grep -A 1 "Tile-Based" | tail -1 | awk '{print $3}')
grid_time_2=$(grep "Execution time:" benchmark_results/test_2.log | grep -A 1 "Tile-Based" | tail -1 | awk '{print $3}')
grid_time_4=$(grep "Execution time:" benchmark_results/test_4.log | grep -A 1 "Tile-Based" | tail -1 | awk '{print $3}')
echo "Tile-Based,1,$grid_time_1" >> benchmark_results/summary.log
echo "Tile-Based,2,$grid_time_2" >> benchmark_results/summary.log
echo "Tile-Based,4,$grid_time_4" >> benchmark_results/summary.log

echo "Benchmarks completed. Results saved in benchmark_results directory."
echo "See benchmark_results/summary.log for a performance comparison." 
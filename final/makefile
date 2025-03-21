# Compiler and flags
CC = gcc
MPICC = mpicc
CFLAGS = 
LDFLAGS = -lm

# Executables
SEQ_EXEC = sequential
MPI_GRID_EXEC = mpigrid
MPI_ROW_EXEC = mpirow

# Source files
SEQ_SRC = sequential.c
MPI_GRID_SRC = mpigrid.c
MPI_ROW_SRC = mpirow.c

# Default target
all: $(SEQ_EXEC) $(MPI_GRID_EXEC) $(MPI_ROW_EXEC)

# Sequential stencil target
$(SEQ_EXEC): $(SEQ_SRC)
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

# MPI stencil target (for future use)
$(MPI_GRID_EXEC): $(MPI_GRID_SRC)
	$(MPICC) $(CFLAGS) -o $@ $< $(LDFLAGS)

$(MPI_ROW_EXEC): $(MPI_ROW_SRC)
	$(MPICC) $(CFLAGS) -o $@ $< $(LDFLAGS)

# Test with 1 node for each implementation
test_1: $(SEQ_EXEC) $(MPI_GRID_EXEC) $(MPI_ROW_EXEC)
	./$(SEQ_EXEC)
	mpiexec -n 1 ./$(MPI_ROW_EXEC)
	mpiexec -n 1 ./$(MPI_GRID_EXEC)

# Test with 2 nodes for each implementation
test_2: $(SEQ_EXEC) $(MPI_GRID_EXEC) $(MPI_ROW_EXEC)
	./$(SEQ_EXEC)
	mpiexec -n 2 ./$(MPI_ROW_EXEC)
	mpiexec -n 2 ./$(MPI_GRID_EXEC)

# Test with 4 nodes for each implementation
test_4: $(SEQ_EXEC) $(MPI_GRID_EXEC) $(MPI_ROW_EXEC)
	./$(SEQ_EXEC)
	mpiexec -n 4 ./$(MPI_ROW_EXEC)
	mpiexec -n 4 ./$(MPI_GRID_EXEC)

# Run all tests
test: test_1 test_2 test_4

# Run the sequential version
run_seq: $(SEQ_EXEC)
	./$(SEQ_EXEC)

# Run the MPI versions with different node counts
run_grid: $(MPI_GRID_EXEC)
	mpiexec -n 2 ./$(MPI_GRID_EXEC)

run_row: $(MPI_ROW_EXEC)
	mpiexec -n 2 ./$(MPI_ROW_EXEC)

# Clean up
clean:
	rm -f $(SEQ_EXEC) $(MPI_GRID_EXEC) $(MPI_ROW_EXEC) *.o
package main

import (
    "fmt"
    "math/rand"
	"time"
	//"math"
)

func main(){
	rand.Seed(time.Now().UnixNano())

	// make 2 random matrices
	m1 := randMatrix(10000, 10000)
	m2 := randMatrix(10000, 10000)

	// time how long the matrix multiply takes
	start := time.Now()
    result := matrixMul(m1, m2)
    duration := time.Since(start)

	// print matrices and time
	//fmt.Println("Matrix 1:")
	//printMatrix(m1);
	//fmt.Println("\nMatrix 2:")
	//printMatrix(m2);
	//fmt.Println("\nMultiplied Matrix")
	//printMatrix(result);
	fmt.Printf("\nExecution time: %v\n", duration)
}

// multiply two matrices
func matrixMul(m1, m2 [][] float32) [][] float32{
	// dimensions of result matrix
	rows := len(m1)
	cols := len(m2[0])

	// make array of proper size 
	result := make([][]float32, rows)
    for i := range result {
        result[i] = make([]float32, cols)
    }

	// go to every position in result matrix
	for i:=0; i<rows; i++ {
		for j:=0; j<cols; j++ {
			result[i][j] = 0

			// multiply every val in ith row in m1 with every val in jth col of m2
			for x:=0; x<len(m1[i]); x++{
				result[i][j] += m1[i][x]*m2[x][j];
			}
		}
	}

	return result
}


// helper: make a random (rows x cols) matrix
func randMatrix(rows, cols int) [][] float32{
	// make array of proper size
	result := make([][]float32, rows)
	for i := range result {
		result[i] = make([]float32, cols)
	}

	// make a random value for every position
	for i:=0; i<rows; i++ {
		for j:=0; j<cols; j++ {
			// note: math.MaxFloat32 gives values that are too large
			// result[i][j] = rand.Float32() * math.MaxFloat32
			result[i][j] = rand.Float32() * 10.0
		}
	}

	return result
}

// helper: print a matrix
func printMatrix(matrix [][] float32) {
	for _, row := range matrix {
        fmt.Println(row)
    }
}
package common

import (
	"math"
)

// Vector just binding to floats slice to add methods to it
type Vector []float64

// IsZero checks if the sum of the all values equals to zero
func (vec Vector) IsZero() bool {
	var sum float64 = 0.0
	for _, val := range vec {
		sum += val
		if sum != 0 {
			return false
		}
	}
	return true
}

// Add two vectors of the same dimnsionality
func (vec Vector) Add(rvec Vector) Vector {
	sum := make(Vector, len(vec))
	for i := range vec {
		sum[i] = vec[i] + rvec[i]
	}
	return sum
}

// ConstMul multiplicates vector with provided constant float
func (vec Vector) ConstMul(constant float64) Vector {
	newVec := make(Vector, len(vec))
	for i := range vec {
		newVec[i] = vec[i] * constant
	}
	return newVec
}

// DotProd calculates dot product between two vectors
func (vec Vector) DotProd(inpVec Vector) float64 {
	var dp float64 = 0.0
	for i := range vec {
		dp += vec[i] * inpVec[i]
	}
	return dp
}

// L2 calculates l2-distance between two vectors
func (vec Vector) L2(inpVec Vector) float64 {
	var l2 float64
	var diff float64
	for i := range vec {
		diff = vec[i] - inpVec[i]
		l2 += diff * diff
	}
	return math.Sqrt(l2)
}

// L2Norm calculates l2 norm of a vector
func (vec Vector) L2Norm() float64 {
	zeroVec := make(Vector, len(vec))
	return vec.L2(zeroVec)
}

// CosineSim calculates cosine similarity of the two given vectors
func (vec *Vector) CosineSim(inpVec Vector) float64 {
	cosine := vec.DotProd(inpVec) / (vec.L2Norm() * inpVec.L2Norm())
	return 1 - cosine
}

package common

import (
	"crypto/rand"
	"fmt"
	"math"
)

// GetRandomID generates random alphanumeric string
func GetRandomID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%x", b)
	return s, nil
}

// NewVector creates new vector by given slice of floats
func NewVector(inpVec []float64) Vector {
	return Vector{
		Values: inpVec,
		Size:   len(inpVec),
	}
}

// Add two vectors of the same dimnsionality
func (vec *Vector) Add(rvec Vector) Vector {
	sum := NewVector(make([]float64, vec.Size))
	for i := range vec.Values {
		sum.Values[i] = vec.Values[i] + rvec.Values[i]
	}
	return sum
}

// ConstMul multiplicates vector with provided constant float
func (vec *Vector) ConstMul(constant float64) Vector {
	newVec := NewVector(make([]float64, vec.Size))
	for i := range vec.Values {
		newVec.Values[i] = vec.Values[i] * constant
	}
	return newVec
}

// DotProd calculates dot product between two vectors
func (vec *Vector) DotProd(inpVec Vector) float64 {
	var dp float64 = 0.0
	for i := range vec.Values {
		dp += vec.Values[i] * inpVec.Values[i]
	}
	return dp
}

// L2 calculates l2-distance of two vectors
func (vec *Vector) L2(inpVec Vector) float64 {
	var l2 float64
	var diff float64
	for i := range vec.Values {
		diff = vec.Values[i] - inpVec.Values[i]
		l2 += diff * diff
	}
	return math.Sqrt(l2)
}

// L2Norm calculates l2 norm of a vector
func (vec *Vector) L2Norm() float64 {
	zeroVec := Vector{
		Values: make([]float64, vec.Size),
	}
	return vec.L2(zeroVec)
}

// CosineSim calculates cosine similarity of two given vectors
func (vec *Vector) CosineSim(inpVec Vector) float64 {
	cosine := vec.DotProd(inpVec) / (vec.L2Norm() * inpVec.L2Norm())
	return cosine
}

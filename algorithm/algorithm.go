package algorithm

import (
	"errors"
	"math"
	"math/rand"
)

func NewVector(inpVec []float64) *Vector {
	return &Vector{
		Values: inpVec,
		Size:   len(inpVec),
	}
}

func Add(lvec *Vector, rvec *Vector) *Vector {
	sum := NewVector(make([]float64, lvec.Size))
	for i := range lvec.Values {
		sum.Values[i] = lvec.Values[i] + rvec.Values[i]
	}
	return sum
}

func (vec *Vector) DotProd(inpVec *Vector) float64 {
	var dp float64 = 0.0
	for i := range vec.Values {
		dp += vec.Values[i] * inpVec.Values[i]
	}
	return dp
}

func (vec *Vector) L2(inpVec *Vector) float64 {
	var l2 float64 = 0.0
	for i := range vec.Values {
		l2 += vec.Values[i]*vec.Values[i] + inpVec.Values[i]*inpVec.Values[i]
	}
	return math.Sqrt(l2)
}

func (vec *Vector) CosineSim(inpVec *Vector) float64 {
	zeroVec := &Vector{
		Values: make([]float64, vec.Size),
	}
	cosine := vec.DotProd(inpVec) / (vec.L2(zeroVec) * inpVec.L2(zeroVec))
	return cosine
}

func (gen *RandomPlaneGenerator) GetRandomPlane() ([]float64, error) {
	if gen.dims <= 0 {
		return nil, errors.New("Dimensions number must be a positive integer")
	}
	coefs := make([]float64, gen.dims+1)
	for i := 0; i < gen.dims; i++ {
		coefs[i] = -1.0 + rand.Float64()*2
	}
	coefs[len(coefs)-1] = -1.0*gen.bias + rand.Float64()*gen.bias*2
	return coefs, nil
}

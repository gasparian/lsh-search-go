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

func (lsh *LSHIndex) getRandomPlane() *Vector {
	coefs := &Vector{
		Values: make([]float64, lsh.dims+1),
		Size:   lsh.dims + 1,
	}
	for i := 0; i < lsh.dims; i++ {
		coefs.Values[i] = -1.0 + rand.Float64()*2
	}
	coefs.Values[coefs.Size-1] = -1.0*lsh.bias + rand.Float64()*lsh.bias*2
	return coefs
}

func (lsh *LSHIndex) Build() error {
	if lsh.dims <= 0 {
		return errors.New("Dimensions number must be a positive integer")
	}
	var vec *Vector
	for i := 0; i < lsh.nPlanes; i++ {
		vec = lsh.getRandomPlane()
		lsh.Planes = append(lsh.Planes, *vec)
	}
	return nil
}

func (lsh *LSHIndex) GetHash(inpVec *Vector) uint64 {
	var hash uint64
	var vec Vector
	var dpSign bool
	for i := 0; i < lsh.nPlanes; i++ {
		vec = lsh.Planes[i]
		dpSign = math.Signbit(inpVec.DotProd(&vec))
		if !dpSign {
			hash |= (1 << i)
		}
	}
	return hash
}

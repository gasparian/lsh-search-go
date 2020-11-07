package algorithm

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"math"
	"math/rand"
	"os"
)

func NewVector(inpVec []float64) *Vector {
	return &Vector{
		Values: inpVec,
		Size:   len(inpVec),
	}
}

func (lvec *Vector) Add(rvec *Vector) *Vector {
	sum := NewVector(make([]float64, lvec.Size))
	for i := range lvec.Values {
		sum.Values[i] = lvec.Values[i] + rvec.Values[i]
	}
	return sum
}

func (vec *Vector) ConstMul(constant float64) *Vector {
	newVec := NewVector(make([]float64, vec.Size))
	for i := range vec.Values {
		newVec.Values[i] = vec.Values[i] * constant
	}
	return newVec
}

func (vec *Vector) DotProd(inpVec *Vector) float64 {
	var dp float64 = 0.0
	for i := range vec.Values {
		dp += vec.Values[i] * inpVec.Values[i]
	}
	return dp
}

func (vec *Vector) L2(inpVec *Vector) float64 {
	var l2 float64
	var diff float64
	for i := range vec.Values {
		diff = vec.Values[i] - inpVec.Values[i]
		l2 += diff * diff
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
	var l2 float64 = 0.0
	for i := 0; i < lsh.dims; i++ {
		coefs.Values[i] = -1.0 + rand.Float64()*2
		l2 += coefs.Values[i] * coefs.Values[i]
	}
	l2 = math.Sqrt(l2)
	bias := l2 * lsh.bias
	coefs.Values[coefs.Size-1] = -1.0*bias + rand.Float64()*bias*2
	return coefs
}

func GetPointPlaneDist(planeCoefs *Vector) *Vector {
	values := make([]float64, planeCoefs.Size-1)
	dCoef := planeCoefs.Values[planeCoefs.Size-1]
	var denom float64 = 0.0
	for i := range values {
		denom += planeCoefs.Values[i] * planeCoefs.Values[i]
	}
	for i := range values {
		values[i] = planeCoefs.Values[i] * dCoef / denom
	}
	return &Vector{
		Values: values,
		Size:   len(values),
	}
}

func (lsh *LSHIndex) Build() error {
	if lsh.dims <= 0 {
		return errors.New("Dimensions number must be a positive integer")
	}
	var coefs *Vector
	for i := 0; i < lsh.nPlanes; i++ {
		coefs = lsh.getRandomPlane()
		lsh.Planes = append(lsh.Planes, Plane{
			Coefs:      coefs,
			InnerPoint: GetPointPlaneDist(coefs),
		})
	}
	return nil
}

func (lsh *LSHIndex) Dump(path string) error {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(*lsh)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(buf.Bytes()); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}

func (lsh *LSHIndex) Load(path string) error {
	buf := &bytes.Buffer{}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	_, err = io.Copy(buf, f)
	if err != nil {
		return err
	}
	f.Close()
	dec := gob.NewDecoder(buf)
	err = dec.Decode(lsh)
	if err != nil {
		return err
	}
	return nil
}

func (lsh *LSHIndex) GetHash(inpVec *Vector) uint64 {
	var hash uint64
	var vec *Vector
	var plane *Plane
	var dpSign bool
	for i := 0; i < lsh.nPlanes; i++ {
		plane = &lsh.Planes[i]
		vec = inpVec.Add(plane.InnerPoint.ConstMul(-1.0))
		dpSign = math.Signbit(vec.DotProd(plane.Coefs))
		if !dpSign {
			hash |= (1 << i)
		}
	}
	return hash
}

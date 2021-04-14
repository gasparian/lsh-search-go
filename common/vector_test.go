package common

import (
	"gonum.org/v1/gonum/blas/blas64"
	"testing"
)

func TestNewVec(t *testing.T) {
	var v blas64.Vector
	v = NewVec([]float64{0.0, 42.0})
	if blas64.Asum(v) != 42.0 {
		t.Fatal("Corrupted conversion to blas vector")
	}
	v = NewVec(nil)
	if blas64.Asum(v) != 0.0 {
		t.Fatal("Corrupted conversion to blas vector: nil should return empty vector")
	}
}

func TestL2(t *testing.T) {
	v1 := NewVec([]float64{0.0, 0.0})
	v2 := NewVec([]float64{-4.0, 3.0})
	l2 := L2(v1, v2)
	if l2 != 5.0 {
		t.Fatal("L2 distance is wrong")
	}
}

func TestCosineSim(t *testing.T) {
	v1 := NewVec([]float64{0.0, 1.0})
	v2 := NewVec([]float64{0.0, 1.0})
	v3 := NewVec([]float64{1.0, 0.0})
	v4 := NewVec([]float64{0.0, -1.0})
	sim1 := CosineSim(v1, v2)
	if sim1 != 0.0 {
		t.Fatal("Cosine similarity must be 0.0 for equal vectors")
	}
	sim2 := CosineSim(v1, v3)
	if sim2 != 1.0 {
		t.Fatal("Cosine similarity must be 1.0 for orthogonal vectors")
	}
	sim3 := CosineSim(v1, v4)
	if sim3 != 2.0 {
		t.Fatal("Cosine similarity must be 2.0 for multidirectional vectors")
	}
}

func TestIsZeroVec(t *testing.T) {
	v1 := NewVec([]float64{0.0, 0.0})
	v2 := NewVec([]float64{0.0, 1.0})
	if !IsZeroVector(v1) {
		t.Fatal("Provided vector should be zero vector")
	}
	if IsZeroVector(v2) {
		t.Fatal("Provided vector should be non-zero vector")
	}
}

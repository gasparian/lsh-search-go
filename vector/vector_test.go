package vector

import (
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func TestNewVec(t *testing.T) {
	t.Parallel()
	var v blas64.Vector
	v = NewVec([]float64{0.0, 42.0})
	if math.Abs(blas64.Asum(v)-42.0) > tol {
		t.Error("Corrupted conversion to blas vector")
	}
	v = NewVec(nil)
	if blas64.Asum(v) != 0.0 {
		t.Error("Corrupted conversion to blas vector: nil should return empty vector")
	}
}

func TestL2(t *testing.T) {
	t.Parallel()
	v1 := NewVec([]float64{0.0, 0.0})
	v2 := NewVec([]float64{-4.0, 3.0})
	l2 := L2(v1, v2)
	if math.Abs(l2-5.0) > tol {
		t.Error("L2 distance is wrong")
	}
}

func TestCosineSim(t *testing.T) {
	t.Parallel()
	v1 := NewVec([]float64{0.0, 1.0})
	v2 := NewVec([]float64{0.0, 1.0})
	v3 := NewVec([]float64{1.0, 0.0})
	v4 := NewVec([]float64{0.0, -1.0})
	sim1 := CosineSim(v1, v2)
	if math.Abs(sim1-0.0) > tol {
		t.Error("Cosine similarity must be 0.0 for equal vectors")
	}
	sim2 := CosineSim(v1, v3)
	if math.Abs(sim2-1.0) > tol {
		t.Error("Cosine similarity must be 1.0 for orthogonal vectors")
	}
	sim3 := CosineSim(v1, v4)
	if math.Abs(sim3-2.0) > tol {
		t.Error("Cosine similarity must be 2.0 for multidirectional vectors")
	}
}

func TestIsZeroVec(t *testing.T) {
	t.Parallel()
	v1 := NewVec([]float64{0.0, 0.0})
	v2 := NewVec([]float64{0.0, 1.0})
	if !IsZeroVector([]float64{0.0, 0.0}) {
		t.Error("Provided vector should be zero vector")
	}
	if !IsZeroVectorBlas(v1) {
		t.Error("Provided vector should be zero vector")
	}
	if IsZeroVectorBlas(v2) {
		t.Error("Provided vector should be non-zero vector")
	}
}

func TestStats(t *testing.T) {
	t.Parallel()
	rand.Seed(time.Now().UnixNano())
	vecs := [][]float64{
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
		[]float64{0.0, 1.0},
		[]float64{0.0, 0.0},
	}
	trueStat := []float64{0.0, 0.5}
	statTol := 0.05
	N := 100
	means := make([][]float64, N)
	stds := make([][]float64, N)
	for i := 0; i < N; i++ {
		mean, std, err := GetMeanStd(vecs, 10)
		if err != nil {
			t.Error(err)
		}
		means[i] = mean
		stds[i] = std
	}
	sort.Slice(means, func(i, j int) bool {
		if len(means[i]) == 0 && len(means[j]) == 0 {
			return false
		}
		if len(means[i]) == 0 || len(means[j]) == 0 {
			return len(means[i]) == 0
		}
		return means[i][1] < means[j][1]
	})
	sort.Slice(stds, func(i, j int) bool {
		if len(stds[i]) == 0 && len(stds[j]) == 0 {
			return false
		}
		if len(stds[i]) == 0 || len(stds[j]) == 0 {
			return len(means[i]) == 0
		}
		return stds[i][1] < stds[j][1]
	})
	var meanStatOk bool = (math.Abs(trueStat[0]-means[N/2][0]) <= statTol) && (math.Abs(trueStat[1]-means[N/2][1]) <= statTol)
	var stdStatOk bool = (math.Abs(trueStat[0]-stds[N/2][0]) <= statTol) && (math.Abs(trueStat[1]-stds[N/2][1]) <= statTol)
	if !(meanStatOk && stdStatOk) {
		t.Error()
	}
	t.Log(means[N/2], stds[N/2])
}

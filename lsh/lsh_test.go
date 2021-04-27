package lsh

import (
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func TestGetHash(t *testing.T) {
	hasherInstance := HasherInstance{
		Planes: []Plane{
			Plane{
				Coefs: NewVec([]float64{1.0, 1.0, 1.0}),
				D:     5,
			},
		},
	}
	inpVec := NewVec([]float64{5.0, 1.0, 1.0})
	meanVec := NewVec([]float64{0.0, 0.0, 0.0})
	hash := hasherInstance.GetHash(inpVec, meanVec)
	if hash != 1 {
		t.Fatal("Wrong hash value, must be 1")
	}
	inpVec = NewVec([]float64{1.0, 1.0, 1.0})
	hash = hasherInstance.GetHash(inpVec, meanVec)
	if hash != 0 {
		t.Fatal("Wrong hash value, must be 0")
	}
}

func getNewHasher(config Config) (*Hasher, error) {
	hasher := New(config)
	mean := []float64{0.0, 0.0, 0.0}
	std := []float64{0.2, 0.3, 0.15}
	err := hasher.Generate(mean, std)
	if err != nil {
		return nil, err
	}
	return hasher, nil
}

func TestGenerateAngular(t *testing.T) {
	config := Config{
		DistanceMetric: Cosine,
		NPermutes:      2,
		NPlanes:        1,
		BiasMultiplier: 2.0,
		DistanceThrsh:  0.8,
		Dims:           3,
	}
	hasherAngular, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}

	isHasherEmpty := IsZeroVectorBlas(hasherAngular.Instances[0].Planes[0].Coefs) ||
		IsZeroVectorBlas(hasherAngular.Instances[0].Planes[0].Coefs)
	if isHasherEmpty {
		t.Fatal("One of the hasher instances is empty")
	}
}
func TestGenerateL2(t *testing.T) {
	config := Config{
		DistanceMetric: Euclidian,
		NPermutes:      2,
		NPlanes:        1,
		BiasMultiplier: 2.0,
		DistanceThrsh:  0.8,
		Dims:           3,
	}
	hasher, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}
	var distToOrigin float64
	maxDist := hasher.Bias * config.BiasMultiplier
	for _, hasherInstance := range hasher.Instances {
		for _, plane := range hasherInstance.Planes {
			distToOrigin = math.Abs(plane.D) / blas64.Nrm2(plane.Coefs)
			if distToOrigin > maxDist {
				t.Fatal("Generated plane is out of bounds defined by hasher config")
			}
		}
	}
}

func TestGetHashes(t *testing.T) {
	config := Config{
		DistanceMetric: Cosine,
		NPermutes:      2,
		NPlanes:        1,
		BiasMultiplier: 2.0,
		DistanceThrsh:  0.8,
		Dims:           3,
	}
	hasherAngular, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}
	inpVec := []float64{0.0, 0.0, 0.0}
	hashes := hasherAngular.GetHashes(inpVec)
	for _, v := range hashes {
		if v != 1 {
			t.Fatal("Hash should always be 1 at this case")
		}
	}
}

func TestGetDistAngular(t *testing.T) {
	config := Config{
		DistanceMetric: Cosine,
		NPermutes:      2,
		NPlanes:        1,
		BiasMultiplier: 2.0,
		DistanceThrsh:  0.8,
		Dims:           3,
	}
	hasherAngular, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}
	v1 := []float64{0.0, 0.0, 0.0}
	v2 := []float64{0.0, 1.0, 0.0}
	dist, ok := hasherAngular.GetDist(v1, v2)
	if ok {
		t.Fatal("Angular distance can't be calculated properly with zero vector")
	}
	v1 = []float64{0.0, 0.0, 2.0}
	v2 = []float64{0.0, 1.0, 0.0}
	dist, _ = hasherAngular.GetDist(v1, v2)
	if ok {
		t.Fatal("Measured dist must be greater than the threshold")
	}
	if dist != 1.0 {
		t.Fatal("Measured dist must be equal to 1.0")
	}
}

func TestGetDistL2(t *testing.T) {
	config := Config{
		DistanceMetric: Euclidian,
		NPermutes:      2,
		NPlanes:        1,
		BiasMultiplier: 2.0,
		DistanceThrsh:  1.1,
		Dims:           3,
	}
	hasher, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}
	v1 := []float64{0.0, 0.0, 0.0}
	v2 := []float64{0.0, 1.0, 0.0}
	dist, ok := hasher.GetDist(v1, v2)
	if !ok {
		t.Fatal("L2 distance must pass the threshold")
	}
	if dist != 1.0 {
		t.Fatal("L2 distance must be equal to 1.0")
	}
}

func TestDumpHasher(t *testing.T) {
	config := Config{
		DistanceMetric: Euclidian,
		NPermutes:      2,
		NPlanes:        1,
		BiasMultiplier: 2.0,
		DistanceThrsh:  1.1,
		Dims:           3,
	}
	hasher, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}
	coefToTest := hasher.Instances[0].Planes[0].D
	b, err := hasher.Dump()
	if err != nil {
		t.Fatalf("Could not serialize hasher: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("Smth went wrong serializing the hasher: resulting bytearray is empty")
	}

	err = hasher.Load(b)
	if err != nil {
		t.Fatalf("Could not deserialize hasher: %v", err)
	}
	if coefToTest != hasher.Instances[0].Planes[0].D {
		t.Fatal("Seems like the deserialized hasher differs from the initial one")
	}
}

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

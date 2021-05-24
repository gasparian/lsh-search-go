package lsh

import (
	"github.com/gasparian/lsh-search-go/store/kv"
	guuid "github.com/google/uuid"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
	"sort"
	"sync"
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
	hash := hasherInstance.getHash(inpVec)
	if hash != 1 {
		t.Fatal("Wrong hash value, must be 1")
	}
	inpVec = NewVec([]float64{1.0, 1.0, 1.0})
	hash = hasherInstance.getHash(inpVec)
	if hash != 0 {
		t.Fatal("Wrong hash value, must be 0")
	}
}

func getNewHasher(config HasherConfig) (*Hasher, error) {
	hasher := NewHasher(config)
	err := hasher.generate()
	if err != nil {
		return nil, err
	}
	return hasher, nil
}

func TestGenerateAngular(t *testing.T) {
	config := HasherConfig{
		NPermutes:           2,
		NPlanes:             1,
		PlaneBiasMultiplier: 2.0,
		Dims:                3,
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
	config := HasherConfig{
		NPermutes:           2,
		NPlanes:             2,
		PlaneBiasMultiplier: 1.0,
		Dims:                3,
	}
	hasher, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}
	var distToOrigin float64
	maxDist := 3.0
	for _, hasherInstance := range hasher.Instances {
		for _, plane := range hasherInstance.Planes {
			distToOrigin = math.Abs(plane.D) / blas64.Nrm2(plane.Coefs)
			if distToOrigin > maxDist {
				t.Fatalf("Generated plane is out of bounds defined by hasher config [%v, %v]", distToOrigin, maxDist)
			}
		}
	}
}

func TestCosineSim(t *testing.T) {
	cosine := NewCosine()
	dist := cosine.GetDist(
		[]float64{0.0, 0.0, 0.0},
		[]float64{0.0, 1.0, 0.0},
	)
	if math.Abs(dist-1.0) > tol {
		t.Fatal("Angular distance can't be calculated properly with zero vector")
	}
	dist = cosine.GetDist(
		[]float64{0.0, 0.0, 2.0},
		[]float64{0.0, 1.0, 0.0},
	)
	if math.Abs(dist-1.0) > tol {
		t.Fatal("Measured dist must be larger than the threshold")
	}

	dist = cosine.GetDist(
		[]float64{0.0, 1.0},
		[]float64{0.0, 1.0},
	)
	if math.Abs(dist-0.0) > tol {
		t.Error("Cosine similarity must be 0.0 for equal vectors")
	}
	dist = cosine.GetDist(
		[]float64{1.0, 0.0},
		[]float64{0.0, -1.0},
	)
	if math.Abs(dist-1.0) > tol {
		t.Error("Cosine similarity must be 1.0 for orthogonal vectors")
	}
	dist = cosine.GetDist(
		[]float64{0.0, 1.0},
		[]float64{0.0, -1.0},
	)
	if math.Abs(dist-2.0) > tol {
		t.Error("Cosine similarity must be 2.0 for multidirectional vectors")
	}
}

func TestL2(t *testing.T) {
	distanceThrsh := 1.1
	v1 := []float64{0.0, 0.0, 0.0}
	v2 := []float64{0.0, 1.0, 0.0}
	l2 := NewL2()
	dist := l2.GetDist(v1, v2)
	if dist > distanceThrsh {
		t.Fatal("L2 distance must pass the threshold")
	}
	if dist != 1.0 {
		t.Fatal("L2 distance must be equal to 1.0")
	}

	v1 = []float64{0.0, 0.0}
	v2 = []float64{-4.0, 3.0}
	dist = l2.GetDist(v1, v2)
	if math.Abs(dist-5.0) > tol {
		t.Error("L2 distance is wrong")
	}
}

func TestDumpHasher(t *testing.T) {
	config := HasherConfig{
		NPermutes:           2,
		NPlanes:             1,
		PlaneBiasMultiplier: 2.0,
		Dims:                3,
	}
	hasher, err := getNewHasher(config)
	if err != nil {
		t.Fatalf("Smth went wrong with planes generation: %v", err)
	}
	coefToTest := hasher.Instances[0].Planes[0].D
	b, err := hasher.dump()
	if err != nil {
		t.Fatalf("Could not serialize hasher: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("Smth went wrong serializing the hasher: resulting bytearray is empty")
	}

	err = hasher.load(b)
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

func TestIsZeroVec(t *testing.T) {
	t.Parallel()
	v1 := NewVec([]float64{0.0, 0.0})
	v2 := NewVec([]float64{0.0, 1.0})
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
	N := 500
	means := make([][]float64, N)
	stds := make([][]float64, N)
	for i := 0; i < N; i++ {
		mean, std, err := GetMeanStdSampled(vecs, 10)
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

func TestScaler(t *testing.T) {
	scaler := NewStandartScaler([]float64{1.0, 1.0}, []float64{0.5, 0.5})
	vec := []float64{1.5, 1.5}
	scaled := scaler.Scale(vec)
	if len(scaled) != len(vec) {
		t.Fatal("Vectors length differs")
	}
	scaledSum := NewVec(scaled)
	if blas64.Asum(scaledSum)-2 > tol {
		t.Errorf("Scaled vector sum should be ~2.0, got %v", scaledSum)
	}
}

func testLSH(metric Metric, config Config, maxNN int, distanceThrsh float64, t *testing.T) Indexer {
	s := kv.NewKVStore()
	lsh, err := NewLsh(config, s, metric)
	if err != nil {
		t.Fatal(err)
	}

	inpVecs := [][]float64{
		[]float64{0.1, 0.1},
		[]float64{0.1, 0.08},
		[]float64{0.11, 0.09},
		[]float64{0.09, 0.11},
		[]float64{-0.1, 0.1},
		[]float64{-0.1, 0.08},
	}

	mean, std, err := GetMeanStdSampled(inpVecs, len(inpVecs))
	if err != nil {
		t.Fatal(err)
	}

	std = []float64{1.0, 1.0} // don't scale, only shift
	t.Log(mean, std)
	scaler := NewStandartScaler(mean, std)

	trainSet := make([]Record, len(inpVecs))
	for i, vec := range inpVecs {
		trainSet[i] = Record{
			ID:  guuid.NewString(),
			Vec: scaler.Scale(vec),
		}
	}

	t.Run("LshTrain", func(t *testing.T) {
		err := lsh.Train(trainSet)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("LshSearch", func(t *testing.T) {
		nns, err := lsh.Search(trainSet[0].Vec, maxNN, distanceThrsh)
		if err != nil {
			t.Fatal(err)
		}
		if len(nns) < 3 || len(nns) > 4 {
			t.Errorf("Query point must have 3-4 neighbors, got %v", len(nns))
		}
	})
	return lsh
}

func TestLshCosine(t *testing.T) {
	t.Parallel()
	const (
		distanceThrsh = 0.2
		maxNN         = 4
	)
	config := Config{
		LshConfig: LshConfig{
			BatchSize: 2,
		},
		HasherConfig: HasherConfig{
			NPermutes:           10,
			NPlanes:             5,
			PlaneBiasMultiplier: 1.0,
			Dims:                2,
		},
	}
	metric := NewCosine()
	lsh := testLSH(metric, config, maxNN, distanceThrsh, t)

	t.Run("LshSearchConcurrent", func(t *testing.T) {
		q := []float64{0.08, 0.1}
		N := 10
		errs := make(chan error, N)
		wg := sync.WaitGroup{}
		wg.Add(N)
		for i := 0; i < N; i++ {
			go func(maxNN int, distanceThrsh float64) {
				defer wg.Done()
				_, err := lsh.Search(q, maxNN, distanceThrsh)
				errs <- err
			}(maxNN, distanceThrsh)
		}
		wg.Wait()
		close(errs)

		for {
			err, ok := <-errs
			if !ok {
				break
			}
			if err != nil {
				t.Error(err)
			}
		}
	})
}

func TestLshL2(t *testing.T) {
	t.Parallel()
	const (
		distanceThrsh = 0.02
		maxNN         = 4
	)
	config := Config{
		LshConfig: LshConfig{
			BatchSize: 2,
		},
		HasherConfig: HasherConfig{
			NPermutes:           10,
			NPlanes:             10,
			PlaneBiasMultiplier: 1.0,
			Dims:                2,
		},
	}
	metric := NewL2()
	_ = testLSH(metric, config, maxNN, distanceThrsh, t)
}

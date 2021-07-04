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

func TestPlane(t *testing.T) {
	p := plane{
		n: NewVec([]float64{1.5, -1}),
		d: 2,
	}
	inpVec := NewVec([]float64{0.0, 0.0})
	if !p.getProductSign(inpVec) {
		t.Error("Product sign must be negative")
	}
	inpVec.Data = []float64{4.0, 0.0}
	if p.getProductSign(inpVec) {
		t.Error("Product sign must be positive")
	}
}

func TestGetHash(t *testing.T) {
	vecs := [][]float64{
		[]float64{-1.0, -1.0},
		[]float64{2.0, -1.0},
	}
	hasherInstance := buildTree(vecs, HasherConfig{KMinVecs: 2, isAngularMetric: false})
	hash := hasherInstance.getHash(NewVec(vecs[0]))
	if hash != 1 {
		t.Fatal("Wrong hash value, must be 1")
	}
	hash = hasherInstance.getHash(NewVec(vecs[1]))
	if hash != 0 {
		t.Fatal("Wrong hash value, must be 0")
	}
}

// TODO: fix tests according to the new cosine sim. calculation algorithm
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
	if math.Abs(dist) > tol {
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
		NTrees:   2,
		KMinVecs: 2,
		Dims:     2,
	}
	vecs := [][]float64{
		[]float64{-1.0, -1.0},
		[]float64{2.0, -1.0},
	}
	hasher := NewHasher(config)
	hasher.build(vecs)
	coefToTest := hasher.trees[0].plane.d
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
	if coefToTest != hasher.trees[0].plane.d {
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
	t.Log("Stats: ", means[N/2], stds[N/2])
}

func TestScaler(t *testing.T) {
	scaler := NewStandartScaler([]float64{1.0, 1.0}, []float64{0.5, 0.5}, 2)
	vec := []float64{1.5, 1.5}
	scaled := scaler.Scale(vec)
	if scaled.N != len(vec) {
		t.Fatal("Vectors length differs")
	}
	scaledSum := blas64.Asum(scaled)
	if scaledSum-2 > tol {
		t.Errorf("Scaled vector sum should be ~2.0, got %v", scaledSum)
	}
}

func testLSH(metric Metric, config Config, maxNN int, distanceThrsh float64, trainSet [][]float64, trainIds []string, t *testing.T) {
	s := kv.NewKVStore()
	lsh, err := NewLsh(config, s, metric)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("LshTrain", func(t *testing.T) {
		err := lsh.Train(trainSet, trainIds)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("LshSearch", func(t *testing.T) {
		nns, err := lsh.Search(trainSet[0], maxNN, distanceThrsh)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("LSH nearest neighbors: ", nns)
		if len(nns) < 3 || len(nns) > 4 {
			t.Fatalf("Query point must have 3-4 neighbors, got %v", len(nns))
		}
	})

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
				t.Fatal(err)
			}
		}
	})
}

func getTestLSHData() ([][]float64, []string) {
	vecs := [][]float64{
		[]float64{0.1, 0.1},
		[]float64{0.1, 0.08},
		[]float64{0.11, 0.09},
		[]float64{0.09, 0.11},
		[]float64{-0.1, 0.1},
		[]float64{-0.1, 0.08},
	}
	ids := make([]string, len(vecs))
	for i := range vecs {
		ids[i] = guuid.NewString()
	}
	return vecs, ids
}

func TestLshCosine(t *testing.T) {
	t.Parallel()
	const (
		distanceThrsh = 0.2
		maxNN         = 4
	)
	inpVecs, trainIds := getTestLSHData()

	config := Config{
		IndexConfig: IndexConfig{
			BatchSize:     2,
			MaxCandidates: 10,
		},
		HasherConfig: HasherConfig{
			NTrees:   10,
			KMinVecs: 2,
			Dims:     2,
		},
	}
	metric := NewCosine()

	testLSH(metric, config, maxNN, distanceThrsh, inpVecs, trainIds, t)
}

func TestLshL2(t *testing.T) {
	t.Parallel()
	const (
		distanceThrsh = 0.02
		maxNN         = 4
	)
	inpVecs, trainIds := getTestLSHData()
	config := Config{
		IndexConfig: IndexConfig{
			BatchSize:     2,
			MaxCandidates: 10,
		},
		HasherConfig: HasherConfig{
			NTrees:   10,
			KMinVecs: 2,
			Dims:     2,
		},
	}
	metric := NewL2()
	testLSH(metric, config, maxNN, distanceThrsh, inpVecs, trainIds, t)
}

package annbench_test

import (
	bench "github.com/gasparian/lsh-search-go/annbench"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store/kv"
	guuid "github.com/google/uuid"
	"gonum.org/v1/hdf5"
	// "math"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"
)

type NNMockConfig struct {
	DistanceThrsh float64
	MaxNN         int
}

// TODO: use kv storage as in lsh indexer - to compare them more fair
type NNMock struct {
	mx             sync.RWMutex
	config         NNMockConfig
	index          map[string][]float64
	distanceMetric lsh.Metric
}

func NewNNMock(config NNMockConfig, metric lsh.Metric) *NNMock {
	return &NNMock{
		config:         config,
		index:          make(map[string][]float64),
		distanceMetric: metric,
	}
}

func (nn *NNMock) Train(records []lsh.Record) error {
	nn.mx.Lock()
	defer nn.mx.Unlock()

	for _, rec := range records {
		nn.index[rec.ID] = rec.Vec
	}
	return nil
}

func (nn *NNMock) Search(query []float64) ([]lsh.Record, error) {
	nn.mx.RLock()
	defer nn.mx.RUnlock()

	closestSet := make(map[string]bool)
	closest := make([]lsh.Record, 0)
	for id, vec := range nn.index {
		if len(closest) >= nn.config.MaxNN {
			return closest, nil
		}
		if closestSet[id] {
			continue
		}
		dist := nn.distanceMetric.GetDist(vec, query)
		if dist <= nn.config.DistanceThrsh {
			closestSet[id] = true
			closest = append(closest, lsh.Record{ID: id, Vec: vec})
		}
	}
	return closest, nil
}

type Indexer interface {
	Train(records []lsh.Record) error
	Search(query []float64) ([]lsh.Record, error)
}

type benchConfig struct {
	datasetPath       string
	sampleSize        int
	batchSize         int
	nPlanes           int
	nPermutes         int
	maxNN             int
	trainDim          int
	neighborsDim      int
	metric            lsh.Metric
	maxDist           float64
	lshBiasMultiplier float64
}

type benchData struct {
	train        []lsh.Record
	test         [][]float64
	trainIndices map[string]int
	neighbors    [][]int
	distances    [][]float64
	mean         []float64
	std          []float64
}

func prepHdf5BenchDataset(config *benchConfig) (*benchData, error) {
	data := &benchData{}
	absPath, _ := filepath.Abs(config.datasetPath)
	f, err := hdf5.OpenFile(absPath, hdf5.F_ACC_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	train := []float32{}
	err = bench.GetVectorsFromHDF5(f, "train", &train)
	if err != nil {
		return nil, err
	}
	data.train = make([]lsh.Record, len(train)/config.trainDim)
	for i := 0; i <= len(train)-config.trainDim; i = i + config.trainDim {
		data.train[i/config.trainDim] = lsh.Record{
			ID:  guuid.NewString(),
			Vec: lsh.ConvertTo64(train[i : i+config.trainDim]),
		}
	}
	train = nil

	data.mean, data.std, err = lsh.GetMeanStd(data.train, config.sampleSize)
	if err != nil {
		return nil, err
	}

	test := []float32{}
	err = bench.GetVectorsFromHDF5(f, "test", &test)
	if err != nil {
		return nil, err
	}
	data.test = make([][]float64, len(test)/config.trainDim)
	for i := 0; i <= len(test)-config.trainDim; i = i + config.trainDim {
		data.test[i/config.trainDim] = lsh.ConvertTo64(test[i : i+config.trainDim])
	}
	test = nil

	neighbors := []int32{}
	err = bench.GetVectorsFromHDF5(f, "neighbors", &neighbors)
	if err != nil {
		return nil, err
	}
	data.neighbors = make([][]int, len(neighbors)/config.neighborsDim)
	for i := 0; i <= len(neighbors)-config.neighborsDim; i = i + config.neighborsDim {
		arr := lsh.ConvertToInt(neighbors[i : i+config.neighborsDim])
		sort.Ints(arr)
		data.neighbors[i/config.neighborsDim] = arr
	}
	neighbors = nil

	data.trainIndices = make(map[string]int)
	for i := range data.train {
		data.trainIndices[data.train[i].ID] = i
	}

	// distances := []float32{}
	// err = bench.GetVectorsFromHDF5(f, "distances", &distances)
	// if err != nil {
	// 	return nil, err
	// }
	// data.distances = make([][]float64, len(distances)/config.neighborsDim)
	// for i := 0; i <= len(distances)-config.neighborsDim; i = i + config.neighborsDim {
	// 	data.distances[i/config.neighborsDim] = lsh.ConvertTo64(distances[i : i+config.neighborsDim])
	// }
	// distances = nil
	data.distances = make([][]float64, 0)

	return data, nil
}

func testIndexer(t *testing.T, indexer Indexer, data *benchData) {
	start := time.Now()
	t.Log("Creating search index...")
	indexer.Train(data.train)
	t.Logf("Training finished in %v", time.Since(start))

	t.Log("Predicting...")
	precision, recall, avgPredTime := 0.0, 0.0, 0.0
	for i := range data.test[:10] { // TODO:
		start = time.Now()
		closest, err := indexer.Search(data.test[i])
		if err != nil {
			t.Fatal(err)
		}
		closestPointsArr := make([]int, 0)
		for _, cl := range closest {
			closestPointsArr = append(closestPointsArr, data.trainIndices[cl.ID])
		}
		// measure Recall
		sort.Ints(closestPointsArr)
		p, r := bench.PrecisionRecall(closestPointsArr, data.neighbors[i])
		precision += p
		recall += r
		avgPredTime += float64(time.Since(start).Milliseconds())
	}
	// testDataLen := float64(len(data.test)) // TODO:
	testDataLen := float64(10)
	precision /= testDataLen
	recall /= testDataLen
	avgPredTime /= testDataLen

	t.Log("Done! Precision: ", precision, "Recall: ", recall)
	t.Logf("Average prediction time is %v ms", avgPredTime)
}

func runBenchTest(t *testing.T, config *benchConfig, data *benchData) {
	t.Run("NN", func(t *testing.T) {
		nn := NewNNMock(NNMockConfig{
			DistanceThrsh: config.maxDist,
			MaxNN:         config.maxNN,
		}, config.metric)
		testIndexer(t, nn, data)
	})

	t.Run("LSH", func(t *testing.T) {
		lshConfig := lsh.Config{
			LshConfig: lsh.LshConfig{
				DistanceThrsh: config.maxDist,
				MaxNN:         config.maxNN,
				BatchSize:     config.batchSize,
			},
			HasherConfig: lsh.HasherConfig{
				NPermutes:      config.nPermutes,
				NPlanes:        config.nPlanes,
				BiasMultiplier: config.lshBiasMultiplier,
				Dims:           config.trainDim,
			},
		}
		lshConfig.Mean = data.mean
		lshConfig.Std = data.std
		s := kv.NewKVStore()
		lshIndex, err := lsh.NewLsh(lshConfig, s, config.metric)
		if err != nil {
			t.Fatal(err)
		}
		testIndexer(t, lshIndex, data)
	})
}

// func TestEuclidian(t *testing.T) {
// 	config := &benchConfig{
// 		datasetPath:       "../test-data/fashion-mnist-784-euclidean.hdf5",
// 		sampleSize:        60000,
// 		batchSize:         500,
// 		nPlanes:           20,
// 		nPermutes:         10,
// 		maxNN:             100,
// 		maxDist:           3000,
// 		trainDim:          784,
// 		neighborsDim:      100,
// 		lshBiasMultiplier: 1.0,
// 		metric:            lsh.NewL2(),
// 	}
// 	data, err := prepHdf5BenchDataset(config)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	runBenchTest(t, config, data)
// }

func TestAngular(t *testing.T) {
	config := &benchConfig{
		datasetPath:       "../test-data/nytimes-256-angular.hdf5",
		sampleSize:        60000,
		batchSize:         500,
		nPlanes:           20,
		nPermutes:         10,
		maxNN:             100,
		maxDist:           0.8,
		trainDim:          256,
		neighborsDim:      100,
		lshBiasMultiplier: 1.0,
		metric:            lsh.NewCosine(),
	}
	data, err := prepHdf5BenchDataset(config)
	if err != nil {
		t.Fatal(err)
	}
	data.std = []float64{}
	// NOTE: optionally, you can measure distance without adjustment by passing empty mean array
	data.mean = []float64{}

	// NOTE: uncomment to look at the ground truth distances values
	// min, max := math.MaxFloat64, 0.0
	// for _, d := range data.distances {
	// 	sorted := sort.Float64Slice(d)
	// 	if sorted[0] < min {
	// 		min = sorted[0]
	// 	}
	// 	if sorted[len(d)-1] > max {
	// 		max = sorted[len(d)-1]
	// 	}
	// }
	// t.Log(min, max)

	runBenchTest(t, config, data)
}

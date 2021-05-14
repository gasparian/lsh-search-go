package annbench_test

import (
	bench "github.com/gasparian/lsh-search-go/annbench"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store/kv"
	guuid "github.com/google/uuid"
	"gonum.org/v1/hdf5"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"
)

const (
	SAMPLE_SIZE = 60000
	BATCH_SIZE  = 500
	N_PLANES    = 20
	N_PERMUTS   = 10
	MAX_NN      = 100
	MAX_DIST    = 3000
)

type NNMockConfig struct {
	DistanceMetric int
	DistanceThrsh  float64
	MaxNN          int
}

// TODO: use kv storage as in lsh indexer - to compare them more fair
type NNMock struct {
	mx     sync.RWMutex
	config NNMockConfig
	index  map[string][]float64
}

func NewNNMock(config NNMockConfig) *NNMock {
	return &NNMock{
		config: config,
		index:  make(map[string][]float64),
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
		var dist float64 = -1
		switch nn.config.DistanceMetric {
		case lsh.Cosine:
			dist = lsh.CosineDist(vec, query)
		case lsh.Euclidian:
			dist = lsh.L2(vec, query)
		}
		if dist < 0 {
			return nil, lsh.DistanceErr
		}
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

func testIndexer(t *testing.T, indexer Indexer, indicesMap map[string]int, trainSplitted []lsh.Record, testSplitted [][]float64, neighborsSplitted [][]int) {
	start := time.Now()
	t.Log("Creating search index...")
	indexer.Train(trainSplitted)
	t.Logf("Training finished in %v", time.Since(start))

	t.Log("Predicting...")
	precision, recall, avgPredTime := 0.0, 0.0, 0.0
	for i := range testSplitted {
		start = time.Now()
		closest, err := indexer.Search(testSplitted[i])
		if err != nil {
			t.Fatal(err)
		}
		closestPointsArr := make([]int, 0)
		for _, cl := range closest {
			closestPointsArr = append(closestPointsArr, indicesMap[cl.ID])
		}
		// measure Recall
		sort.Ints(closestPointsArr)
		p, r := bench.PrecisionRecall(closestPointsArr, neighborsSplitted[i])
		precision += p
		recall += r
		avgPredTime += float64(time.Since(start).Milliseconds())
	}
	precision /= float64(len(testSplitted))
	recall /= float64(len(testSplitted))
	avgPredTime /= float64(len(testSplitted))

	t.Log("Done! Precision: ", precision, "Recall: ", recall)
	t.Logf("Average prediction time is %v ms", avgPredTime)
}

func TestFashionMnist(t *testing.T) {
	// Read train/test data from the fashion mnist dataset
	t.Log("Opening dataset...")
	absPath, _ := filepath.Abs("../test-data/fashion-mnist-784-euclidean.hdf5")
	f, err := hdf5.OpenFile(absPath, hdf5.F_ACC_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	trainDim := 784
	neighborsDim := 100

	train := []float32{}
	err = bench.GetVectorsFromHDF5(f, "train", &train)
	if err != nil {
		t.Fatal(err)
	}
	trainSplitted := make([]lsh.Record, len(train)/trainDim)
	for i := 0; i <= len(train)-trainDim; i = i + trainDim {
		trainSplitted[i/trainDim] = lsh.Record{
			ID:  guuid.NewString(),
			Vec: lsh.ConvertTo64(train[i : i+trainDim]),
		}
	}
	train = nil

	mean, std, err := lsh.GetMeanStd(trainSplitted, SAMPLE_SIZE)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Train set ready (%v entries)", len(trainSplitted))

	test := []float32{}
	err = bench.GetVectorsFromHDF5(f, "test", &test)
	if err != nil {
		t.Fatal(err)
	}
	testSplitted := make([][]float64, len(test)/trainDim)
	for i := 0; i <= len(test)-trainDim; i = i + trainDim {
		testSplitted[i/trainDim] = lsh.ConvertTo64(test[i : i+trainDim])
	}
	test = nil

	t.Logf("Test set is ready (%v entries)", len(testSplitted))

	neighbors := []int32{}
	err = bench.GetVectorsFromHDF5(f, "neighbors", &neighbors)
	if err != nil {
		t.Fatal(err)
	}
	neighborsSplitted := make([][]int, len(neighbors)/neighborsDim)
	for i := 0; i <= len(neighbors)-neighborsDim; i = i + neighborsDim {
		arr := lsh.ConvertToInt(neighbors[i : i+neighborsDim])
		sort.Ints(arr)
		neighborsSplitted[i/neighborsDim] = arr
	}
	neighbors = nil

	t.Logf("Ground truth data set is ready (%v entries)", len(neighborsSplitted))

	t.Log("Populating indeces map")
	indicesMap := make(map[string]int)
	for i := range trainSplitted {
		indicesMap[trainSplitted[i].ID] = i
	}

	t.Run("NN", func(t *testing.T) {
		nn := NewNNMock(NNMockConfig{
			DistanceMetric: lsh.Euclidian,
			DistanceThrsh:  MAX_DIST,
			MaxNN:          MAX_NN,
		})
		testIndexer(t, nn, indicesMap, trainSplitted, testSplitted, neighborsSplitted)
	})

	t.Run("LSH", func(t *testing.T) {
		config := lsh.Config{
			LshConfig: lsh.LshConfig{
				DistanceMetric: lsh.Euclidian,
				DistanceThrsh:  MAX_DIST,
				MaxNN:          MAX_NN,
				BatchSize:      BATCH_SIZE,
			},
			HasherConfig: lsh.HasherConfig{
				NPermutes:      N_PERMUTS,
				NPlanes:        N_PLANES,
				BiasMultiplier: 1.0,
				Dims:           784,
			},
		}
		config.Mean = mean
		config.Std = std
		s := kv.NewKVStore()
		lshIndex, err := lsh.NewLsh(config, s)
		if err != nil {
			t.Fatal(err)
		}
		testIndexer(t, lshIndex, indicesMap, trainSplitted, testSplitted, neighborsSplitted)
	})
}

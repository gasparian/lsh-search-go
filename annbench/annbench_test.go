package annbench_test

import (
	"github.com/cheggaaa/pb/v3"
	bench "github.com/gasparian/lsh-search-go/annbench"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	guuid "github.com/google/uuid"
	"gonum.org/v1/hdf5"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

const (
	SAMPLE_SIZE = 60000
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

func TestFashionMnist(t *testing.T) {
	// Read train/test data from the fashion mnist dataset
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

	// mean, std, err := lsh.GetMeanStd(trainSplitted, SAMPLE_SIZE)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	t.Log(len(trainSplitted))

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

	t.Log(len(testSplitted))

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

	t.Log(len(neighborsSplitted))

	t.Log("Populating indeces map")
	indicesMap := make(map[string]int)
	bar := pb.StartNew(len(trainSplitted))
	for i := range trainSplitted {
		bar.Increment()
		indicesMap[trainSplitted[i].ID] = i
	}
	bar.Finish()

	t.Run("NN", func(t *testing.T) {
		nn := NewNNMock(NNMockConfig{
			DistanceMetric: lsh.Euclidian,
			DistanceThrsh:  MAX_DIST,
			MaxNN:          MAX_NN,
		})

		nn.Train(trainSplitted)

		bar = pb.StartNew(len(testSplitted))
		precision, recall := 0.0, 0.0
		for i := range testSplitted {
			bar.Increment()
			closest, err := nn.Search(testSplitted[i])
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
		}
		bar.Finish()
		precision /= float64(len(testSplitted))
		recall /= float64(len(testSplitted))

		t.Log("Precision: ", precision, "Recall: ", recall)
	})

	// t.Run("LSH", func(t *testing.T) {
	// 	// Create LSH index
	// 	lshIndexMnist := lsh.New(lsh.Config{
	// 		DistanceMetric: lsh.Euclidian,
	// 		NPermutes:      N_PERMUTS,
	// 		NPlanes:        N_PLANES,
	// 		BiasMultiplier: 1.0,
	// 		DistanceThrsh:  MAX_DIST,
	// 		Dims:           784,
	// 	})

	// 	// Generate planes for hashing
	// 	err = lshIndexMnist.Generate(mean, std)
	// 	if err != nil {
	// 		log.Panic(err)
	// 	}

	// 	log.Println("Bias: ", lshIndexMnist.Bias)

	// 	// Prepare map to store search index
	// 	// TODO: make concurrent map and store it inside hasher object?
	// 	m := make(map[int]map[uint64][]*[]float64)
	// 	for i := 0; i < N_PERMUTS; i++ {
	// 		m[i] = make(map[uint64][]*[]float64)
	// 	}

	// 	// Populate index (train dataset)
	// 	log.Println("Populating index...")
	// 	// TODO: fill indeces map in a separate loop outside
	// 	indicesMap := make(map[*[]float64]int)
	// 	bar := pb.StartNew(len(trainSplitted))
	// 	for i := range trainSplitted {
	// 		bar.Increment()
	// 		hashes := lshIndexMnist.GetHashes(trainSplitted[i])
	// 		for perm, hash := range hashes {
	// 			m[perm][hash] = append(m[perm][hash], &trainSplitted[i])
	// 		}
	// 		indicesMap[&trainSplitted[i]] = i
	// 	}
	// 	bar.Finish()

	// 	// for i, v := range m {
	// 	// 	for hash, s := range v {
	// 	// 		log.Println(i, hash, len(s))
	// 	// 	}
	// 	// }

	// 	// Get test hashes
	// 	log.Println("Making pedictions...")
	// 	bar = pb.StartNew(len(testSplitted))
	// 	precision, recall := 0.0, 0.0
	// 	for i := range testSplitted {
	// 		bar.Increment()
	// 		hashes := lshIndexMnist.GetHashes(testSplitted[i])
	// 		closest := make(map[int]bool)
	// 		for perm, hash := range hashes {
	// 			if len(closest) == MAX_NN {
	// 				break
	// 			}
	// 			nn := m[perm][hash]
	// 			for j := range nn {
	// 				if closest[indicesMap[nn[j]]] {
	// 					continue
	// 				}
	// 				_, isClose := lshIndexMnist.GetDist(*(nn[j]), testSplitted[i]) // TODO: now it's standalone func
	// 				if isClose {
	// 					closest[indicesMap[nn[j]]] = true
	// 					if len(closest) == MAX_NN {
	// 						break
	// 					}
	// 				}
	// 			}
	// 		}
	// 		closestPointsArr := make([]int, 0)
	// 		for k := range closest {
	// 			closestPointsArr = append(closestPointsArr, k)
	// 		}
	// 		// measure Recall
	// 		sort.Ints(closestPointsArr)
	// 		p, r := bench.PrecisionRecall(closestPointsArr, neighborsSplitted[i])
	// 		precision += p
	// 		recall += r
	// 	}
	// 	bar.Finish()
	// 	precision /= float64(len(testSplitted))
	// 	recall /= float64(len(testSplitted))

	// 	log.Println("Precision: ", precision, "Recall: ", recall)
	// })
}

func TestLastFM(t *testing.T) {

}

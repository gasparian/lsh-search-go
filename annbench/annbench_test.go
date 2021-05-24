package annbench_test

import (
	bench "github.com/gasparian/lsh-search-go/annbench"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store"
	"github.com/gasparian/lsh-search-go/store/kv"
	guuid "github.com/google/uuid"
	"gonum.org/v1/hdf5"
	"math"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type NNMockConfig struct {
	DistanceThrsh float64
	MaxNN         int
}

type NNMock struct {
	mx             sync.RWMutex
	config         NNMockConfig
	index          store.Store
	distanceMetric lsh.Metric
}

func NewNNMock(config NNMockConfig, store store.Store, metric lsh.Metric) *NNMock {
	return &NNMock{
		config:         config,
		index:          store,
		distanceMetric: metric,
	}
}

func (nn *NNMock) Train(records []lsh.Record) error {
	err := nn.index.Clear()
	if err != nil {
		return err
	}
	for _, rec := range records {
		nn.index.SetVector(rec.ID, rec.Vec)
		nn.index.SetHash(0, 0, rec.ID)
	}
	return nil
}

func (nn *NNMock) Search(query []float64) ([]lsh.Record, error) {
	nn.mx.RLock()
	config := nn.config
	nn.mx.RUnlock()

	closestSet := make(map[string]bool)
	closest := make([]lsh.Record, 0)

	iter, _ := nn.index.GetHashIterator(0, 0)
	for {
		if len(closest) >= config.MaxNN {
			break
		}
		id, opened := iter.Next()
		if !opened {
			break
		}

		if closestSet[id] {
			continue
		}
		vec, err := nn.index.GetVector(id)
		if err != nil {
			return nil, err
		}
		dist := nn.distanceMetric.GetDist(vec, query)
		if dist <= config.DistanceThrsh {
			closestSet[id] = true
			closest = append(closest, lsh.Record{ID: id, Vec: vec})
		}
	}
	return closest, nil
}

type benchDataConfig struct {
	datasetPath  string
	sampleSize   int
	trainDim     int
	neighborsDim int
}

type searchConfig struct {
	metric            lsh.Metric
	maxDist           float64
	lshBiasMultiplier float64
	nDims             int
	nPlanes           int
	nPermutes         int
	maxNN             int
	batchSize         int
	meanVec           []float64
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

func prepHdf5BenchDataset(config *benchDataConfig) (*benchData, error) {
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

	data.mean, data.std, err = lsh.GetMeanStdSampledRecords(data.train, config.sampleSize)
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

	distances := []float32{}
	err = bench.GetVectorsFromHDF5(f, "distances", &distances)
	if err != nil {
		return nil, err
	}
	data.distances = make([][]float64, len(distances)/config.neighborsDim)
	for i := 0; i <= len(distances)-config.neighborsDim; i = i + config.neighborsDim {
		data.distances[i/config.neighborsDim] = lsh.ConvertTo64(distances[i : i+config.neighborsDim])
	}
	distances = nil
	// data.distances = make([][]float64, 0)

	return data, nil
}

type prediction struct {
	records []lsh.Record
	idx     int
}

func testIndexer(t *testing.T, indexer lsh.Indexer, data *benchData) {
	start := time.Now()
	t.Log("Creating search index...")
	indexer.Train(data.train)
	t.Logf("Training finished in %v", time.Since(start))

	t.Log("Predicting...")
	N := 1000 // TODO: for debug only
	batchSize := 100
	var elapsedTimeMs int64
	predCh := make(chan prediction, N)
	wg := sync.WaitGroup{}
	wg.Add(len(data.test[:N])/batchSize + len(data.test[:N])%batchSize)
	for i := 0; i < len(data.test[:N]); i += batchSize {
		end := i + batchSize
		if end > len(data.test[:N]) {
			end = len(data.test[:N])
		}
		go func(batch [][]float64, startIdx int, wg *sync.WaitGroup) {
			defer wg.Done()
			for j := range batch {
				start := time.Now()
				closest, err := indexer.Search(batch[j])
				if err != nil {
					panic(err)
				}
				predCh <- prediction{records: closest, idx: startIdx + j}
				atomic.AddInt64(&elapsedTimeMs, int64(time.Since(start)/time.Millisecond))
			}
		}(data.test[i:end], i, &wg)
	}
	wg.Wait()
	close(predCh)

	precision, recall := 0.0, 0.0
	for pred := range predCh {
		closestPointsArr := make([]int, 0)
		for _, cl := range pred.records {
			closestPointsArr = append(closestPointsArr, data.trainIndices[cl.ID])
		}
		sort.Ints(closestPointsArr)
		p, r := bench.PrecisionRecall(closestPointsArr, data.neighbors[pred.idx])
		precision += p
		recall += r
	}

	testDataLen := float64(len(data.test[:N]))

	precision /= testDataLen
	recall /= testDataLen
	avgPredTime := float64(elapsedTimeMs) / testDataLen

	t.Log("Done! Precision: ", precision, "Recall: ", recall)
	t.Logf("Prediction finished in %v s", elapsedTimeMs/1000)
	t.Logf("Average prediction time is %v ms", avgPredTime)
}

func testNearestNeighbors(t *testing.T, config *searchConfig, data *benchData) {
	s := kv.NewKVStore()
	nn := NewNNMock(
		NNMockConfig{
			DistanceThrsh: config.maxDist,
			MaxNN:         config.maxNN,
		},
		s, config.metric,
	)
	testIndexer(t, nn, data)
}

func testLSH(t *testing.T, config *searchConfig, data *benchData) {
	lshConfig := lsh.Config{
		LshConfig: lsh.LshConfig{
			DistanceThrsh: config.maxDist,
			MaxNN:         config.maxNN,
			BatchSize:     config.batchSize,
			MeanVec:       config.meanVec,
		},
		HasherConfig: lsh.HasherConfig{
			NPermutes:      config.nPermutes,
			NPlanes:        config.nPlanes,
			BiasMultiplier: config.lshBiasMultiplier,
			Dims:           config.nDims,
		},
	}
	s := kv.NewKVStore()
	lshIndex, err := lsh.NewLsh(lshConfig, s, config.metric)
	if err != nil {
		t.Fatal(err)
	}
	testIndexer(t, lshIndex, data)
}

func getFloat64Range(data [][]float64) (float64, float64) {
	min, max := math.MaxFloat64, 0.0
	for _, d := range data {
		sorted := sort.Float64Slice(d)
		if sorted[0] < min {
			min = sorted[0]
		}
		if sorted[len(d)-1] > max {
			max = sorted[len(d)-1]
		}
	}
	return min, max
}

// func TestEuclideanFashionMnist(t *testing.T) {
// 	dataConfig := &benchDataConfig{
// 		datasetPath:  "../test-data/fashion-mnist-784-euclidean.hdf5",
// 		sampleSize:   60000,
// 		trainDim:     784,
// 		neighborsDim: 100,
// 	}
// 	data, err := prepHdf5BenchDataset(dataConfig)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	minStd, maxStd := getFloat64Range([][]float64{data.std})
// 	t.Log("Dimensions std's range: ", minStd, maxStd)

// 	config := &searchConfig{
// 		lshBiasMultiplier: maxStd * 3.0,
// 		metric:            lsh.NewL2(),
// 		nDims:             784,
// 		batchSize:         250,
// 		nPlanes:           12,
// 		nPermutes:         10,
// 		maxNN:             100,
// 		maxDist:           3000,
// 		// meanVec:           data.mean,
// 		meanVec: nil,
// 	}

// 	// NOTE: look at the ground truth distances values
// 	minDist, maxDist := getFloat64Range(data.distances)
// 	t.Log("Ground truth distances range: ", minDist, maxDist)

// t.Run("NN", func(t *testing.T) {
//     testNearestNeighbors(t, config, data)
// })
// t.Run("LSH", func(t *testing.T) {
//     testLSH(t, config, data)
// })
// }

func TestAngularNYTimes(t *testing.T) {
	dataConfig := &benchDataConfig{
		datasetPath:  "../test-data/nytimes-256-angular.hdf5",
		sampleSize:   60000,
		trainDim:     256,
		neighborsDim: 100,
	}
	data, err := prepHdf5BenchDataset(dataConfig)
	if err != nil {
		t.Fatal(err)
	}
	minStd, maxStd := getFloat64Range([][]float64{data.std})
	t.Log("Dimensions std's range: ", minStd, maxStd)

	config := &searchConfig{
		lshBiasMultiplier: 4.0,
		metric:            lsh.NewCosine(),
		nDims:             256,
		batchSize:         250,
		nPlanes:           100,
		nPermutes:         10,
		maxNN:             100,
		maxDist:           0.85,
		meanVec:           data.mean,
		// meanVec: nil,
	}

	// NOTE: look at the ground truth distances values
	minDist, maxDist := getFloat64Range(data.distances)
	t.Log("Ground truth distances range: ", minDist, maxDist)

	// t.Run("NN", func(t *testing.T) {
	// 	testNearestNeighbors(t, config, data)
	// })
	t.Run("LSH", func(t *testing.T) {
		testLSH(t, config, data)
	})
}

// NOTE: warning - it will eat a LOT of RAM
// func TestEuclideanSift(t *testing.T) {
// 	config := &benchDataConfig{
// 		datasetPath:       "../test-data/sift-128-euclidean.hdf5",
// 		sampleSize:        200000,
// 		batchSize:         250,
// 		nPlanes:           40,
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

// 	// NOTE: look at the ground truth distances values
// 	minDist, maxDist := getDistRange(data.distances)
// 	t.Log("Ground truth distances range: ", minDist, maxDist)

// t.Run("NN", func(t *testing.T) {
//     testNearestNeighbors(t, config, data)
// })
// t.Run("LSH", func(t *testing.T) {
// 	testLSH(t, config, data)
// })
// }

package annbench_test

import (
	bench "github.com/gasparian/lsh-search-go/annbench"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store/kv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	tol = 1e-6
)

func testIndexer(t *testing.T, indexer lsh.Indexer, data *bench.BenchData, config *bench.SearchConfig) {
	start := time.Now()
	t.Log("Creating search index...")
	indexer.Train(data.TrainVecs, data.TrainIds)
	t.Logf("Training finished in %v", time.Since(start))

	t.Log("Predicting...")
	start = time.Now()
	N := 10000 // NOTE: for debug it's convenient to change this to lower value in sake of speed up (default is 10k)
	batchSize := 1000
	var elapsedTimeMs int64
	predCh := make(chan bench.Prediction, N)
	wg := sync.WaitGroup{}
	for i := 0; i < len(data.Test[:N]); i += batchSize {
		wg.Add(1)
		end := i + batchSize
		if end > len(data.Test[:N]) {
			end = len(data.Test[:N])
		}
		go func(batch [][]float64, startIdx int, wg *sync.WaitGroup) {
			defer wg.Done()
			for j := range batch {
				start := time.Now()
				closest, err := indexer.Search(batch[j], config.MaxNN, config.MaxDist)
				if err != nil {
					panic(err)
				}
				atomic.AddInt64(&elapsedTimeMs, int64(time.Since(start)/time.Millisecond))
				predCh <- bench.Prediction{Neighbors: closest, Idx: startIdx + j}
			}
		}(data.Test[i:end], i, &wg)
	}
	wg.Wait()
	close(predCh)

	precision, recall := 0.0, 0.0
	for pred := range predCh {
		closestPointsArr := make([]int, 0)
		for _, cl := range pred.Neighbors {
			closestPointsArr = append(closestPointsArr, data.TrainIndices[cl.ID])
		}
		p, r := bench.DistanceBasedPrecisionRecall(
			closestPointsArr,
			data.Neighbors[pred.Idx][:config.MaxNN],
			pred.Neighbors,
			data.Distances[pred.Idx][:config.MaxNN],
			config.Epsilon,
		)
		precision += p
		recall += r
	}
	overallElapsedTime := time.Since(start)

	testDataLen := float64(len(data.Test[:N]))

	precision /= testDataLen
	recall /= testDataLen
	avgPredTime := float64(elapsedTimeMs) / testDataLen

	t.Log("Done! Precision: ", precision, "Recall: ", recall)
	t.Logf("Concurrent prediction finished in %v", overallElapsedTime)
	t.Logf("Average prediction time is %v ms", avgPredTime)
}

func testNearestNeighbors(t *testing.T, config *bench.SearchConfig, data *bench.BenchData) {
	s := kv.NewKVStore()
	nn := bench.NewNNMock(config.MaxCandidates, s, config.Metric)
	testIndexer(t, nn, data, config)
}

func testLSH(t *testing.T, config *bench.SearchConfig, data *bench.BenchData) {
	lshConfig := lsh.Config{
		IndexConfig: lsh.IndexConfig{
			BatchSize:     config.BatchSize,
			MaxCandidates: config.MaxCandidates,
		},
		HasherConfig: lsh.HasherConfig{
			NTrees:   config.NTrees,
			KMinVecs: config.KMinVecs,
			Dims:     config.NDims,
		},
	}
	s := kv.NewKVStore()
	lshIndex, err := lsh.NewLsh(lshConfig, s, config.Metric)
	if err != nil {
		t.Fatal(err)
	}
	testIndexer(t, lshIndex, data, config)
}

func TestEuclideanFashionMnist(t *testing.T) {
	dataConfig := &bench.BenchDataConfig{
		DatasetPath:  "../test-data/fashion-mnist-784-euclidean.hdf5",
		SampleSize:   30000,
		TrainDim:     784,
		NeighborsDim: 100,
	}
	data, err := bench.PrepHdf5BenchDataset(dataConfig)
	if err != nil {
		t.Fatal(err)
	}

	minStd, maxStd := bench.GetFloat64Range([][]float64{data.Std})
	t.Log("Dimensions std's range: ", minStd, maxStd)

	// NOTE: look at the ground truth distances values
	minDist, maxDist := bench.GetFloat64Range(data.Distances)
	t.Log("Ground truth distances range: ", minDist, maxDist)

	config := &bench.SearchConfig{
		Metric:        lsh.NewL2(),
		MaxNN:         10,
		MaxDist:       2200,
		MaxCandidates: 30000,
		Epsilon:       0.05,
	}
	// t.Run("NN", func(t *testing.T) {
	// 	testNearestNeighbors(t, config, data)
	// })

	config = &bench.SearchConfig{
		NDims:         784,
		BatchSize:     500,
		KMinVecs:      200,
		NTrees:        10,
		Metric:        lsh.NewL2(),
		MaxNN:         10,
		Epsilon:       0.05,
		MaxDist:       2200,
		MaxCandidates: 5000,
	}
	t.Run("LSH", func(t *testing.T) {
		testLSH(t, config, data)
	})
}

func TestEuclideanSift(t *testing.T) {
	dataConfig := &bench.BenchDataConfig{
		DatasetPath:  "../test-data/sift-128-euclidean.hdf5",
		SampleSize:   200000,
		TrainDim:     128,
		NeighborsDim: 100,
	}
	data, err := bench.PrepHdf5BenchDataset(dataConfig)
	if err != nil {
		t.Fatal(err)
	}
	minStd, maxStd := bench.GetFloat64Range([][]float64{data.Std})
	t.Log("Dimensions std's range: ", minStd, maxStd)
	minMean, maxMean := bench.GetFloat64Range([][]float64{data.Mean})
	t.Log("Dimensions mean's range: ", minMean, maxMean)

	// NOTE: look at the ground truth distances values
	minDist, maxDist := bench.GetFloat64Range(data.Distances)
	t.Log("Ground truth distances range: ", minDist, maxDist)

	config := &bench.SearchConfig{
		Metric:        lsh.NewL2(),
		NDims:         128,
		MaxDist:       400,
		MaxCandidates: 200000,
		Epsilon:       0.05,
	}

	t.Run("NN", func(t *testing.T) {
		testNearestNeighbors(t, config, data)
	})

	config = &bench.SearchConfig{
		Metric:        lsh.NewL2(),
		NDims:         128,
		BatchSize:     500,
		NTrees:        40,
		KMinVecs:      300,
		MaxNN:         10,
		MaxDist:       300,
		Epsilon:       0.05,
		MaxCandidates: 10000,
	}
	t.Run("LSH", func(t *testing.T) {
		testLSH(t, config, data)
	})
}

func TestAngularNYTimes(t *testing.T) {
	dataConfig := &bench.BenchDataConfig{
		DatasetPath:  "../test-data/nytimes-256-angular.hdf5",
		SampleSize:   60000,
		TrainDim:     256,
		NeighborsDim: 100,
	}
	data, err := bench.PrepHdf5BenchDataset(dataConfig)
	if err != nil {
		t.Fatal(err)
	}
	minStd, maxStd := bench.GetFloat64Range([][]float64{data.Std})
	t.Log("Dimensions std's range: ", minStd, maxStd)
	minMean, maxMean := bench.GetFloat64Range([][]float64{data.Mean})
	t.Log("Dimensions mean's range: ", minMean, maxMean)

	// NOTE: look at the ground truth distances values
	minDist, maxDist := bench.GetFloat64Range(data.Distances)
	t.Log("Ground truth distances range: ", minDist, maxDist)

	config := &bench.SearchConfig{
		Metric:        lsh.NewCosine(),
		MaxNN:         10,
		MaxDist:       0.9,
		MaxCandidates: 30000,
	}
	// t.Run("NN", func(t *testing.T) {
	// 	testNearestNeighbors(t, config, data)
	// })

	config = &bench.SearchConfig{
		Metric:        lsh.NewCosine(),
		NDims:         256,
		BatchSize:     500,
		NTrees:        200,
		KMinVecs:      250,
		MaxNN:         10,
		MaxDist:       0.85,
		Epsilon:       0.05,
		MaxCandidates: 20000,
	}
	t.Run("LSH", func(t *testing.T) {
		testLSH(t, config, data)
	})
}

func TestAngularGlove(t *testing.T) {
	dataConfig := &bench.BenchDataConfig{
		DatasetPath:  "../test-data/glove-200-angular.hdf5",
		SampleSize:   200000,
		TrainDim:     200,
		NeighborsDim: 100,
	}
	data, err := bench.PrepHdf5BenchDataset(dataConfig)
	if err != nil {
		t.Fatal(err)
	}
	minStd, maxStd := bench.GetFloat64Range([][]float64{data.Std})
	t.Log("Dimensions std's range: ", minStd, maxStd)
	minMean, maxMean := bench.GetFloat64Range([][]float64{data.Mean})
	t.Log("Dimensions mean's range: ", minMean, maxMean)

	// NOTE: look at the ground truth distances values
	minDist, maxDist := bench.GetFloat64Range(data.Distances)
	t.Log("Ground truth distances range: ", minDist, maxDist)

	config := &bench.SearchConfig{
		Metric:        lsh.NewCosine(),
		MaxNN:         10,
		MaxDist:       0.9,
		MaxCandidates: 100000,
		Epsilon:       0.05,
	}
	// t.Run("NN", func(t *testing.T) {
	// 	testNearestNeighbors(t, config, data)
	// })

	config = &bench.SearchConfig{
		Metric:        lsh.NewCosine(),
		NDims:         200,
		BatchSize:     500,
		NTrees:        40,
		KMinVecs:      500,
		MaxNN:         10,
		MaxDist:       0.75,
		Epsilon:       0.05,
		MaxCandidates: 20000,
	}
	t.Run("LSH", func(t *testing.T) {
		testLSH(t, config, data)
	})
}

package annbench_test

import (
	bench "github.com/gasparian/lsh-search-go/annbench"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store/kv"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testIndexer(t *testing.T, indexer lsh.Indexer, data *bench.BenchData, maxNN int, maxDist float64) {
	start := time.Now()
	t.Log("Creating search index...")
	indexer.Train(data.Train)
	t.Logf("Training finished in %v", time.Since(start))

	t.Log("Predicting...")
	start = time.Now()
	N := 10000 // NOTE: for debug it's convenient to change this to lower value in sake of speed up
	batchSize := 100
	var elapsedTimeMs int64
	predCh := make(chan bench.Prediction, N)
	wg := sync.WaitGroup{}
	wg.Add(len(data.Test[:N])/batchSize + len(data.Test[:N])%batchSize)
	for i := 0; i < len(data.Test[:N]); i += batchSize {
		end := i + batchSize
		if end > len(data.Test[:N]) {
			end = len(data.Test[:N])
		}
		go func(batch [][]float64, startIdx int, wg *sync.WaitGroup) {
			defer wg.Done()
			for j := range batch {
				start := time.Now()
				closest, err := indexer.Search(batch[j], maxNN, maxDist)
				if err != nil {
					panic(err)
				}
				predCh <- bench.Prediction{Records: closest, Idx: startIdx + j}
				atomic.AddInt64(&elapsedTimeMs, int64(time.Since(start)/time.Millisecond))
			}
		}(data.Test[i:end], i, &wg)
	}
	wg.Wait()
	close(predCh)

	precision, recall := 0.0, 0.0
	for pred := range predCh {
		closestPointsArr := make([]int, 0)
		for _, cl := range pred.Records {
			closestPointsArr = append(closestPointsArr, data.TrainIndices[cl.ID])
		}
		sort.Ints(closestPointsArr)
		p, r := bench.PrecisionRecall(closestPointsArr, data.Neighbors[pred.Idx])
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
	nn := bench.NewNNMock(s, config.Metric)
	testIndexer(t, nn, data, config.MaxNN, config.MaxDist)
}

func testLSH(t *testing.T, config *bench.SearchConfig, data *bench.BenchData) {
	lshConfig := lsh.Config{
		IndexConfig: lsh.IndexConfig{
			BatchSize: config.BatchSize,
			Bias:      config.Bias,
		},
		HasherConfig: lsh.HasherConfig{
			NPermutes:           config.NPermutes,
			NPlanes:             config.NPlanes,
			PlaneBiasMultiplier: config.LshPlaneBiasMultiplier,
			Dims:                config.NDims,
		},
	}
	s := kv.NewKVStore()
	lshIndex, err := lsh.NewLsh(lshConfig, s, config.Metric)
	if err != nil {
		t.Fatal(err)
	}
	testIndexer(t, lshIndex, data, config.MaxNN, config.MaxDist)
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

	config := &bench.SearchConfig{
		LshPlaneBiasMultiplier: 1.0,
		Metric:                 lsh.NewL2(),
		NDims:                  784,
		BatchSize:              250,
		NPlanes:                12,
		NPermutes:              10,
		MaxNN:                  100,
		MaxDist:                3000,
		Bias:                   data.Mean,
	}

	// NOTE: look at the ground truth distances values
	minDist, maxDist := bench.GetFloat64Range(data.Distances)
	t.Log("Ground truth distances range: ", minDist, maxDist)

	t.Run("NN", func(t *testing.T) {
		testNearestNeighbors(t, config, data)
	})
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
		LshPlaneBiasMultiplier: 0.0,
		Metric:                 lsh.NewCosine(),
		NDims:                  256,
		BatchSize:              250,
		NPlanes:                12,
		NPermutes:              12,
		MaxNN:                  100,
		MaxDist:                0.9,
		Bias:                   data.Mean,
	}

	t.Run("NN", func(t *testing.T) {
		testNearestNeighbors(t, config, data)
	})
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
		LshPlaneBiasMultiplier: 1.0,
		Metric:                 lsh.NewL2(),
		NDims:                  128,
		BatchSize:              500,
		NPlanes:                20,
		NPermutes:              10,
		MaxNN:                  100,
		MaxDist:                400,
		Bias:                   data.Mean,
	}

	t.Run("NN", func(t *testing.T) {
		testNearestNeighbors(t, config, data)
	})
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
		LshPlaneBiasMultiplier: 0.0,
		Metric:                 lsh.NewCosine(),
		NDims:                  200,
		BatchSize:              500,
		NPlanes:                20,
		NPermutes:              10,
		MaxNN:                  100,
		MaxDist:                0.9,
		Bias:                   data.Mean,
	}

	t.Run("NN", func(t *testing.T) {
		testNearestNeighbors(t, config, data)
	})
	t.Run("LSH", func(t *testing.T) {
		testLSH(t, config, data)
	})
}

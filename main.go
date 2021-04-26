package main

import (
	"github.com/cheggaaa/pb/v3"
	bench "github.com/gasparian/lsh-search-go/annbench"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"gonum.org/v1/hdf5"
	"log"
	"sort"
	// "math"
	"path/filepath"
	// "reflect"
)

const (
	SAMPLE_SIZE = 30000
	N_PLANES    = 30
	N_PERMUTS   = 10
	MAX_NN      = 100
)

func main() {
	// Read train/test data from the fashion mnist dataset
	absPath, _ := filepath.Abs("../lsh-search-go/test-data/fashion-mnist-784-euclidean.hdf5")
	f, err := hdf5.OpenFile(absPath, hdf5.F_ACC_RDONLY)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	trainDim := 784
	neighborsDim := 100

	train := []float32{}
	err = bench.GetVectorsFromHDF5(f, "train", &train)
	if err != nil {
		log.Panic(err)
	}
	trainSplitted := make([][]float64, len(train)/trainDim)
	for i := 0; i <= len(train)-trainDim; i = i + trainDim {
		trainSplitted[i/trainDim] = lsh.ConvertTo64(train[i : i+trainDim])
	}
	train = nil

	mean, std, err := lsh.GetMeanStd(trainSplitted, SAMPLE_SIZE)
	if err != nil {
		log.Panic(err)
	}

	log.Println(len(trainSplitted))

	test := []float32{}
	err = bench.GetVectorsFromHDF5(f, "test", &test)
	if err != nil {
		log.Panic(err)
	}
	testSplitted := make([][]float64, len(test)/trainDim)
	for i := 0; i <= len(test)-trainDim; i = i + trainDim {
		testSplitted[i/trainDim] = lsh.ConvertTo64(test[i : i+trainDim])
	}
	test = nil

	log.Println(len(testSplitted))

	neighbors := []int32{}
	err = bench.GetVectorsFromHDF5(f, "neighbors", &neighbors)
	if err != nil {
		log.Panic(err)
	}
	neighborsSplitted := make([][]int, len(neighbors)/neighborsDim)
	for i := 0; i <= len(neighbors)-neighborsDim; i = i + neighborsDim {
		neighborsSplitted[i/neighborsDim] = lsh.ConvertToInt(neighbors[i : i+neighborsDim])
	}
	neighbors = nil

	log.Println(len(neighborsSplitted))

	distances := []float32{}
	err = bench.GetVectorsFromHDF5(f, "distances", &distances)
	if err != nil {
		log.Panic(err)
	}
	distancesSplitted := make([][]float64, len(distances)/neighborsDim)
	for i := 0; i <= len(distances)-neighborsDim; i = i + neighborsDim {
		distancesSplitted[i/neighborsDim] = lsh.ConvertTo64(distances[i : i+neighborsDim])
	}
	distances = nil

	distMean, _, err := lsh.GetMeanStd(distancesSplitted, 5000)
	if err != nil {
		log.Panic(err)
	}
	distMax := 0.0
	for _, val := range distMean {
		if val > distMax {
			distMax = val
		}
	}

	log.Println(len(distancesSplitted), distMax)

	// Create LSH index
	lshIndexMnist := lsh.New(lsh.Config{
		DistanceMetric: lsh.Euclidian,
		NPermutes:      N_PERMUTS,
		NPlanes:        N_PLANES,
		BiasMultiplier: 2,
		DistanceThrsh:  distMax,
		Dims:           784,
	})

	// Generate planes
	err = lshIndexMnist.Generate(
		lsh.NewVec(mean),
		lsh.NewVec(std),
	)
	if err != nil {
		log.Panic(err)
	}

	// Prepare map to store search index
	// TODO: make concurrent map and store it inside hasher object?
	m := make(map[int]map[uint64][]*[]float64)
	for i := 0; i < N_PERMUTS; i++ {
		m[i] = make(map[uint64][]*[]float64)
	}

	// Populate index
	log.Println("Populating index...")
	indicesMap := make(map[*[]float64]int)
	bar := pb.StartNew(len(trainSplitted))
	for i := range trainSplitted {
		bar.Increment()
		hashes := lshIndexMnist.GetHashes(trainSplitted[i])
		for perm, hash := range hashes {
			m[perm][hash] = append(m[perm][hash], &trainSplitted[i])
		}
		indicesMap[&trainSplitted[i]] = i
	}
	bar.Finish()

	// Get test hashes
	log.Println("Making pedictions...")
	// bar = pb.StartNew(len(testSplitted))
	// neighborsPredicted := make([][]int32, len(testSplitted))
	// for i, vec := range testSplitted {
	for i, vec := range trainSplitted {
		// bar.Increment()
		hashes := lshIndexMnist.GetHashes(vec)
		closest := make([]int, 0) // TODO: must be set?
		for perm, hash := range hashes {
			nn := m[perm][hash]
			for j := range nn {
				if len(closest) == MAX_NN {
					break
				}
				dist, isClose := lshIndexMnist.GetDist(*nn[j], vec)

				if isClose {
					log.Println(dist)
					closest = append(closest, indicesMap[nn[j]])
				}
			}
		}
		// measure Recall
		sort.Ints(closest)
		groundTruth := neighborsSplitted[i]
		sort.Ints(groundTruth)
		log.Println(len(closest), len(groundTruth), bench.Recall(closest, groundTruth))
	}
	// bar.Finish()
}

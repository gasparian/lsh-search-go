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
	SAMPLE_SIZE = 60000
	N_PLANES    = 20
	N_PERMUTS   = 10
	MAX_NN      = 100
	MAX_DIST    = 3000
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
		arr := lsh.ConvertToInt(neighbors[i : i+neighborsDim])
		sort.Ints(arr)
		neighborsSplitted[i/neighborsDim] = arr
	}
	neighbors = nil

	log.Println(len(neighborsSplitted))

	// Create LSH index
	lshIndexMnist := lsh.New(lsh.Config{
		DistanceMetric: lsh.Euclidian,
		NPermutes:      N_PERMUTS,
		NPlanes:        N_PLANES,
		BiasMultiplier: 1.0,
		DistanceThrsh:  MAX_DIST,
		Dims:           784,
	})

	// Generate planes for hashing
	err = lshIndexMnist.Generate(mean, std)
	if err != nil {
		log.Panic(err)
	}

	log.Println("Bias: ", lshIndexMnist.Bias)

	// testHash := lshIndexMnist.GetHashes(testSplitted[0])
	// valHashTrue := lshIndexMnist.GetHashes(trainSplitted[neighborsSplitted[0][0]])
	// distTrue, isCloseTrue := lshIndexMnist.GetDist(testSplitted[0], trainSplitted[neighborsSplitted[0][0]])
	// valHashFalse := lshIndexMnist.GetHashes(trainSplitted[neighborsSplitted[5000][0]])
	// distFalse, isCloseFalse := lshIndexMnist.GetDist(testSplitted[0], trainSplitted[neighborsSplitted[5000][0]])

	// log.Println("Test hash: ", testHash)
	// log.Println("Val hash true: ", valHashTrue)
	// log.Println("Dist true: ", distTrue, isCloseTrue)
	// log.Println("Val hash false: ", valHashFalse)
	// log.Println("Dist false: ", distFalse, isCloseFalse)

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
		cpy := make([]float64, len(trainSplitted[i]))
		copy(cpy, trainSplitted[i])
		hashes := lshIndexMnist.GetHashes(cpy)
		// hashes := lshIndexMnist.GetHashes(trainSplitted[i])
		for perm, hash := range hashes {
			m[perm][hash] = append(m[perm][hash], &trainSplitted[i])
		}
		indicesMap[&trainSplitted[i]] = i
	}
	bar.Finish()

	// for i, v := range m {
	// 	for hash, s := range v {
	// 		log.Println(i, hash, len(s))
	// 	}
	// }

	// Get test hashes
	log.Println("Making pedictions...")
	bar = pb.StartNew(len(testSplitted))
	precision, recall := 0.0, 0.0
	for i := range testSplitted {
		bar.Increment()
		cpy := make([]float64, len(testSplitted[i]))
		copy(cpy, testSplitted[i])
		hashes := lshIndexMnist.GetHashes(cpy)
		// hashes := lshIndexMnist.GetHashes(vec)
		closest := make(map[int]bool)
		for perm, hash := range hashes {
			if len(closest) == MAX_NN {
				break
			}
			nn := m[perm][hash]
			for j := range nn {
				if closest[indicesMap[nn[j]]] {
					continue
				}
				copy(cpy, testSplitted[i])
				_, isClose := lshIndexMnist.GetDist(*(nn[j]), cpy)
				// log.Println(dist, isClose)
				if isClose {
					closest[indicesMap[nn[j]]] = true
					if len(closest) == MAX_NN {
						break
					}
				}
			}
		}
		closestPointsArr := make([]int, 0)
		for k := range closest {
			closestPointsArr = append(closestPointsArr, k)
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

	log.Println("Precision: ", precision, "Recall: ", recall)
}

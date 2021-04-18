package main

import (
	bench "github.com/gasparian/similarity-search-go/lsh/annbench"
	// lsh "github.com/gasparian/similarity-search-go/lsh/lsh"
	vc "github.com/gasparian/similarity-search-go/lsh/vector"
	"gonum.org/v1/hdf5"
	"log"
	"path/filepath"
)

const (
	SAMPLE_SIZE = 50000
	N_PLANES    = 30
	N_PERMUTS   = 10
	MAX_NN      = 100
)

func main() {

	// Read train/test data from the fashion mnist dataset
	absPath, _ := filepath.Abs("../similarity-search-go/test-data/fashion-mnist-784-euclidean.hdf5")
	f, err := hdf5.OpenFile(absPath, hdf5.F_ACC_RDONLY)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	train := []float32{}
	err = bench.GetVectorsFromHDF5(f, "train", &train)
	if err != nil {
		log.Panic(err)
	}
	trainSplitted := make([][]float64, len(train)/784)
	for i := 0; i <= len(train)-784; i = i + 784 {
		trainSplitted[i/784] = vc.ConvertTo64(train[i : i+784])
	}
	train = nil

	mean, err := vc.GetStat(trainSplitted, []float64{}, 0.1, 30000)
	if err != nil {
		log.Panic(err)
	}
	std, err := vc.GetStat(trainSplitted, mean, 0.1, 30000)
	if err != nil {
		log.Panic(err)
	}

	log.Println(len(trainSplitted), mean, std)

	// test := []float32{}
	// err = bench.GetVectorsFromHDF5(f, "test", &test)
	// if err != nil {
	// 	log.Panic(err)
	// }
	// neighbors := []int32{}
	// err = bench.GetVectorsFromHDF5(f, "neighbors", &neighbors)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// log.Println(len(train), len(test), len(neighbors))

	// // Prepare map to store search index
	// m := make(map[int]map[uint64]*bench.MnistFeatureVec)
	// for i := 0; i < N_PERMUTS; i++ {
	// 	m[i] = make(map[uint64]*bench.MnistFeatureVec)
	// }

	// // Create LSH index
	// lshIndexMnist := lsh.New(lsh.Config{
	// 	IsAngularDistance: 0,
	// 	NPermutes:         N_PERMUTS,
	// 	NPlanes:           N_PLANES,
	// 	BiasMultiplier:    1,
	// 	DistanceThrsh:     20000,
	// 	Dims:              784,
	// })
	// // Get mean and std from the train dataset

	// // Generate planes
	// err = lshIndexMnist.Generate(
	// 	vc.NewVec(mean),
	// 	vc.NewVec(std),
	// )
	// if err != nil {
	// 	log.Panic(err)
	// }
}

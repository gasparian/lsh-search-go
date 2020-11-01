package main

import (
	"fmt"
	"unsafe"
	"vector-search-go/data"

	"gonum.org/v1/hdf5"
)

func main() {
	f, err := hdf5.OpenFile("./data/deep-image-96-angular.hdf5", hdf5.F_ACC_RDWR)
	if err != nil {
		panic(err)
	}
	// Objects in the file:
	// distances
	// neighbors
	// test
	// train

	dataset, err := f.OpenDataset("train")
	if err != nil {
		panic(err)
	}
	fileSpace := dataset.Space()

	numTicks := fileSpace.SimpleExtentNPoints()
	numTicks /= (int)(unsafe.Sizeof(data.FeatureVec{})) / 4
	fmt.Printf("Reading %d ticks\n", numTicks)

	ticks := make([]data.FeatureVec, numTicks)
	err = dataset.Read(&ticks)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sample: %v\n", ticks[0])

	meanVal, err := data.CalcMeanStd(&ticks, 1e4, &[]float32{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Mean val: %v\n", meanVal)

	stdVal, err := data.CalcMeanStd(&ticks, 1e4, &meanVal)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Std val: %v\n", stdVal)

	// release main dataset
	dataset.Close()

	err = data.WriteNewDataHDF5(f, "Mean", &meanVal)
	if err != nil {
		panic(err)
	}

	err = data.WriteNewDataHDF5(f, "Std", &stdVal)
	if err != nil {
		panic(err)
	}

	f.Close()
}

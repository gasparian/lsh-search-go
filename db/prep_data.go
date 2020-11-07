package data

import (
	"errors"
	"math"
	"sync"

	"gonum.org/v1/hdf5"
)

type FeatureVec = [96]float32
type NeighborsIds = [100]uint32
type DistanceVec = [100]float32

// MinInt returns minimal integer of two
func MinInt(x, y int) int {
	if x > y {
		return y
	}
	return x
}

type concurrentSlice struct {
	sync.RWMutex
	items []float32
}

// Concurrently calculates mean and std of slice of slices
func CalcMeanStd(vals *[]FeatureVec, chunksize int, meanVec *[]float32) ([]float32, error) {
	size := len(*vals)
	if size == 0 {
		return nil, errors.New("Slice can't be empty")
	}
	ndims := len((*vals)[0])
	meanVecSize := len(*meanVec)
	if meanVecSize > 0 {
		if ndims != len(*meanVec) {
			return nil, errors.New("Mean vector and values must be the same size")
		}
	}
	result := &concurrentSlice{
		items: make([]float32, ndims),
	}
	var wg sync.WaitGroup
	var idx, end int
	for idx < size {
		wg.Add(1)
		end = MinInt(idx+chunksize, size)
		go func(startIdx, endIdx int) {
			var tmpVal float32
			tmpSl := make([]float32, ndims)
			defer wg.Done()
			for i := startIdx; i < endIdx; i++ {
				for j := 0; j < ndims; j++ {
					if meanVecSize > 0 {
						tmpVal = (*vals)[i][j] - (*meanVec)[j]
						tmpSl[j] += tmpVal * tmpVal
					} else {
						tmpSl[j] += (*vals)[i][j]
					}
				}
			}
			result.Lock()
			for j := 0; j < ndims; j++ {
				result.items[j] += tmpSl[j]
			}
			result.Unlock()
		}(idx, end)
		idx = end
	}
	wg.Wait()

	floatSize := (float32)(size)
	for j := 0; j < ndims; j++ {
		if meanVecSize > 0 {
			result.items[j] = (float32)(math.Sqrt((float64)(result.items[j]))) / floatSize
		} else {
			result.items[j] /= floatSize
		}
	}
	return result.items, nil
}

// WriteNewDataHDF5 add new fields to the existing hdf5 file
func WriteNewDataHDF5(file *hdf5.File, dsName string, vals *[]float32) error {
	if len(*vals) == 0 {
		return errors.New("Cannot write empty slice")
	}
	// create the memory data type
	dtype, err := hdf5.NewDatatypeFromValue((*vals)[0])
	if err != nil {
		return err
	}

	var dset *hdf5.Dataset
	dset, err = file.OpenDataset(dsName)
	if err != nil {
		dims := []uint{(uint)(len(*vals))}
		space, err := hdf5.CreateSimpleDataspace(dims, nil)
		if err != nil {
			return err
		}

		dset, err = file.CreateDataset(dsName, dtype, space)
		if err != nil {
			return err
		}
	}

	err = dset.Write(vals)
	if err != nil {
		return err
	}

	dset.Close()
	return nil
}

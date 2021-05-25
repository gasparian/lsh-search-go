package annbench

import (
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store"
	guuid "github.com/google/uuid"
	"gonum.org/v1/hdf5"
	"math"
	"path/filepath"
	"sort"
	"sync"
)

type BenchDataConfig struct {
	DatasetPath  string
	SampleSize   int
	TrainDim     int
	NeighborsDim int
}

type SearchConfig struct {
	Metric                 lsh.Metric
	MaxDist                float64
	LshPlaneBiasMultiplier float64
	NDims                  int
	NPlanes                int
	NPermutes              int
	MaxNN                  int
	BatchSize              int
	Bias                   []float64
}

type BenchData struct {
	Train        []lsh.Record
	Test         [][]float64
	TrainIndices map[string]int
	Neighbors    [][]int
	Distances    [][]float64
	Mean         []float64
	Std          []float64
}

type Prediction struct {
	Records []lsh.Record
	Idx     int
}

type NNMock struct {
	mx             sync.RWMutex
	index          store.Store
	distanceMetric lsh.Metric
}

func NewNNMock(store store.Store, metric lsh.Metric) *NNMock {
	return &NNMock{
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

func (nn *NNMock) Search(query []float64, maxNN int, distanceThrsh float64) ([]lsh.Record, error) {
	closestSet := make(map[string]bool)
	closest := make([]lsh.Record, 0)

	iter, _ := nn.index.GetHashIterator(0, 0)
	for {
		if len(closest) >= maxNN {
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
		if dist <= distanceThrsh {
			closestSet[id] = true
			closest = append(closest, lsh.Record{ID: id, Vec: vec})
		}
	}
	return closest, nil
}

func GetFloat64Range(data [][]float64) (float64, float64) {
	min, max := math.MaxFloat64, -math.MaxFloat64
	cpy := make([]float64, len(data[0]))
	for _, d := range data {
		copy(cpy, d)
		sort.Float64Slice.Sort(cpy)
		if cpy[0] < min {
			min = cpy[0]
		}
		if cpy[len(d)-1] > max {
			max = cpy[len(d)-1]
		}
	}
	return min, max
}

// Recall returns ratio of relevant predictions over the all true relevant items
// both arrays MUST BE SORTED
func PrecisionRecall(prediction, groundTruth []int) (float64, float64) {
	valid := 0
	for _, val := range prediction {
		idx := sort.SearchInts(groundTruth, val)
		if idx < len(groundTruth) {
			valid++
		}
	}
	precision := 0.0
	if len(prediction) > 0 {
		precision = float64(valid) / float64(len(prediction))
	}
	recall := float64(valid) / float64(len(groundTruth))
	return precision, recall
}

// Objects inside the hdf5:
// train
// test
// distances
// neighbors

// GetVectorsFromHDF5 returns slice of feature vectors, from the hdf5 table
func GetVectorsFromHDF5(table *hdf5.File, datasetName string, vecs interface{}) error {
	dataset, err := table.OpenDataset(datasetName)
	if err != nil {
		return err
	}
	defer dataset.Close()

	fileSpace := dataset.Space()
	numTicks := fileSpace.SimpleExtentNPoints()

	switch vecs := vecs.(type) {
	case *[]float32:
		*vecs = make([]float32, numTicks)
	case *[]int32:
		*vecs = make([]int32, numTicks)
	}

	err = dataset.Read(vecs)
	if err != nil {
		return err
	}

	return nil
}

func PrepHdf5BenchDataset(config *BenchDataConfig) (*BenchData, error) {
	data := &BenchData{}
	absPath, _ := filepath.Abs(config.DatasetPath)
	f, err := hdf5.OpenFile(absPath, hdf5.F_ACC_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	train := []float32{}
	err = GetVectorsFromHDF5(f, "train", &train)
	if err != nil {
		return nil, err
	}
	data.Train = make([]lsh.Record, len(train)/config.TrainDim)
	for i := 0; i <= len(train)-config.TrainDim; i = i + config.TrainDim {
		data.Train[i/config.TrainDim] = lsh.Record{
			ID:  guuid.NewString(),
			Vec: lsh.ConvertTo64(train[i : i+config.TrainDim]),
		}
	}
	train = nil

	data.Mean, data.Std, err = lsh.GetMeanStdSampledRecords(data.Train, config.SampleSize)
	if err != nil {
		return nil, err
	}

	test := []float32{}
	err = GetVectorsFromHDF5(f, "test", &test)
	if err != nil {
		return nil, err
	}
	data.Test = make([][]float64, len(test)/config.TrainDim)
	for i := 0; i <= len(test)-config.TrainDim; i = i + config.TrainDim {
		data.Test[i/config.TrainDim] = lsh.ConvertTo64(test[i : i+config.TrainDim])
	}
	test = nil

	neighbors := []int32{}
	err = GetVectorsFromHDF5(f, "neighbors", &neighbors)
	if err != nil {
		return nil, err
	}
	data.Neighbors = make([][]int, len(neighbors)/config.NeighborsDim)
	for i := 0; i <= len(neighbors)-config.NeighborsDim; i = i + config.NeighborsDim {
		arr := lsh.ConvertToInt(neighbors[i : i+config.NeighborsDim])
		sort.Ints(arr)
		data.Neighbors[i/config.NeighborsDim] = arr
	}
	neighbors = nil

	data.TrainIndices = make(map[string]int)
	for i := range data.Train {
		data.TrainIndices[data.Train[i].ID] = i
	}

	distances := []float32{}
	err = GetVectorsFromHDF5(f, "distances", &distances)
	if err != nil {
		return nil, err
	}
	data.Distances = make([][]float64, len(distances)/config.NeighborsDim)
	for i := 0; i <= len(distances)-config.NeighborsDim; i = i + config.NeighborsDim {
		data.Distances[i/config.NeighborsDim] = lsh.ConvertTo64(distances[i : i+config.NeighborsDim])
	}
	distances = nil
	// data.distances = make([][]float64, 0)

	return data, nil
}

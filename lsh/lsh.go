package lsh

import (
	"container/heap"
	"errors"
	"github.com/gasparian/lsh-search-go/store"
	"math"
	"sync"
)

var (
	DistanceErr = errors.New("Distance can't be calculated")
)

// Neighbor represent neighbor vector with distance to the query vector
type Neighbor struct {
	Vec  []float64
	ID   string
	Dist float64
}

type NeighborMinHeap []*Neighbor

func (h NeighborMinHeap) Len() int {
	return len(h)
}

func (h NeighborMinHeap) Less(i, j int) bool {
	return h[i].Dist < h[j].Dist
}

func (h NeighborMinHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *NeighborMinHeap) Push(x interface{}) {
	*h = append(*h, x.(*Neighbor))
}

func (h *NeighborMinHeap) Pop() interface{} {
	old := *h
	tailIndex := old.Len() - 1
	tail := old[tailIndex]
	old[tailIndex] = nil
	*h = old[:tailIndex]
	return tail
}

// Metric holds implementation of needed distance metric
type Metric interface {
	GetDist(l, r []float64) float64
	IsAngular() bool
}

// Indexer holds implementation of NN search index
type Indexer interface {
	Train(vecs [][]float64, ids []string) error
	Search(query []float64, maxNN int, distanceThrsh float64) ([]Neighbor, error)
}

// IndexConfig ...
type IndexConfig struct {
	mx            *sync.RWMutex
	BatchSize     int
	MaxCandidates int
}

func (c *IndexConfig) getBatchSize() int {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.BatchSize
}

func (c *IndexConfig) getMaxCandidates() int {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.MaxCandidates
}

// Config holds all needed constants for creating the Hasher instance
type Config struct {
	IndexConfig
	HasherConfig
}

// LSHIndex holds buckets with vectors and hasher instance
type LSHIndex struct {
	config         IndexConfig
	index          store.Store
	hasher         *Hasher
	distanceMetric Metric
}

// New creates new instance of hasher and index, where generated hashes will be stored
func NewLsh(config Config, store store.Store, metric Metric) (*LSHIndex, error) {
	config.HasherConfig.isAngularMetric = metric.IsAngular()
	hasher := NewHasher(config.HasherConfig)
	config.IndexConfig.mx = new(sync.RWMutex)
	return &LSHIndex{
		config:         config.IndexConfig,
		hasher:         hasher,
		index:          store,
		distanceMetric: metric,
	}, nil
}

// Train fills new search index with vectors
func (lsh *LSHIndex) Train(vecs [][]float64, ids []string) error {
	err := lsh.index.Clear()
	if err != nil {
		return err
	}
	lsh.hasher.build(vecs)
	batchSize := lsh.config.getBatchSize()
	wg := sync.WaitGroup{}
	for i := 0; i < len(vecs); i += batchSize {
		wg.Add(1)
		end := i + batchSize
		if end > len(vecs) {
			end = len(vecs)
		}
		go func(vecs [][]float64, ids []string, wg *sync.WaitGroup) {
			defer wg.Done()
			for i := range vecs {
				hashes := lsh.hasher.getHashes(vecs[i])
				lsh.index.SetVector(ids[i], vecs[i])
				for perm, hash := range hashes {
					bucketName := getBucketName(perm, hash)
					lsh.index.SetHash(bucketName, ids[i])
				}
			}
		}(vecs[i:end], ids[i:end], &wg)
	}
	wg.Wait()
	return nil
}

// Search returns NNs for the query point
func (lsh *LSHIndex) Search(query []float64, maxNN int, distanceThrsh float64) ([]Neighbor, error) {
	maxCandidates := lsh.config.getMaxCandidates()
	hashes := lsh.hasher.getHashes(query)
	closestSet := make(map[string]bool)
	minHeap := new(NeighborMinHeap)
	for perm, hash := range hashes {
		if minHeap.Len() >= maxCandidates {
			break
		}
		// NOTE: look in the neigbors' "bucket" too
		var neighborPos int = 0
		if hash > 0 {
			neighborPos = int(math.Floor(math.Log2(float64(hash))))
		}
		neighborHash := hash ^ (1 << neighborPos)
		bucketsNames := []string{
			getBucketName(perm, hash),
			getBucketName(perm, neighborHash),
		}
		for _, bucketName := range bucketsNames {
			iter, err := lsh.index.GetHashIterator(bucketName)
			if err != nil {
				continue // NOTE: it's normal when we couldn't find bucket for the query point
			}
			for {
				if minHeap.Len() >= maxCandidates {
					break
				}
				id, opened := iter.Next()
				if !opened {
					break
				}
				if closestSet[id] {
					continue
				}
				vec, err := lsh.index.GetVector(id)
				if err != nil {
					return nil, err
				}
				dist := lsh.distanceMetric.GetDist(vec, query)
				if dist <= distanceThrsh {
					closestSet[id] = true
					heap.Push(
						minHeap,
						&Neighbor{
							ID:   id,
							Vec:  vec,
							Dist: dist,
						},
					)
				}
			}

		}
	}
	closest := make([]Neighbor, 0)
	for i := 0; i < maxNN && minHeap.Len() > 0; i++ {
		closest = append(closest, *heap.Pop(minHeap).(*Neighbor))
	}
	return closest, nil
}

// DumpHasher serializes hasher
func (lsh *LSHIndex) DumpHasher() ([]byte, error) {
	return lsh.hasher.dump()
}

// LoadHasher fills hasher from byte array
func (lsh *LSHIndex) LoadHasher(inp []byte) error {
	return lsh.hasher.load(inp)
}

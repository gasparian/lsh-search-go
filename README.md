![lsh tests](https://github.com/gasparian/lsh-search-go/actions/workflows/test.yml/badge.svg?branch=master)
## lsh-search-go  

One of the most important things in the field of search and recommender systems is a problem of fast search in high-dimensional vector spaces.  
The simplest possible search algorithm would be just NN search with linear space and time complexity. But in practice we want to perform search in << O(N) time.  
So there is exist a set of algorithms called "approximate nearest neighbors" (aka "ANN"). And we can divide it into two subsets:  
 - [locality-sensitive hashing](https://en.wikipedia.org/wiki/Locality-sensitive_hashing) (space partitioning methods);  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - like local search over proximity graphs, for example [hierarchical navigatable small world graphs](https://arxiv.org/pdf/1603.09320.pdf);  

I've decided to go with the LSH algorithm first, since:  
  1. It's pretty simple to understand and implement it.  
  2. At some cases LSH can be more suitable for production usage, since it's index can be easily stored in any database.  

So this repo contains library that has the functionality to create LSH index and perform search by given query vector.  

### Local sensitive hashing short reference   

Recipe for creating LSH search index:  
  1. Generate *k* random hyper planes.  
  2. Calculate bit hash for each point in a dataset, relying on it's position relative to the each generated plane. Store points with the same hash in a separate hash table.  
  3. Repeat the process *l* times --> so at the end we store *l* search indeces.  
  4. For each query point generate *l* hashes of length *k* and search for nearest neighbors in prepared hash tables.  

We can expect that nearby vectors have the higher probability to be in the same bucket.  
Storing all vectors in many hash tables requires much space, but searching for nearby vectors can then be done faster, as many distant vectors are not considered for reductions. So we will always have the trade-off between space usage and search time.  

Here is visual example of space partitioning:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-go/blob/master/pics/biased.jpg" height=400/> </p>  

I prefer to use simple rules while tuning the algorithm:  
  - more planes permutations you create --> more space you use, but more accurate the model could become;  
  - more planes you generate --> more "buckets" with less points you get --> search becomes faster, but can be less accurate (more false negative errors, potentially);  
  - larger distance threshold you make --> more "candidate" points you will have during the search phase, so you can satisfy the "max. nearest neighbors" condition faster, but decrease accuracy.  

### API  

The storage and hashing parts are **decoupled** from each other.  
You need to implement only two interfaces:  
  1. [store](https://github.com/gasparian/lsh-search-go/blob/master/store/store.go), in order to use any storage you prefer.  
  2. [metric](https://github.com/gasparian/lsh-search-go/blob/master/lsh/lsh.go), to use your custom distance metric.  

LSH index object has a super-simple interface:  
 - `NewLsh` is for creating the new instance of index by given config;  
 - `Train(records []lsh.Record) error` for filling search index with vectors (each `lsh.Record` must contain unique id and `[]float64` vector itself);  
 - `Search(query []float64) ([]lsh.Record, error)` to find `MaxNN` nearest neighbors to the query vector;  

Here is the usage example:  
```go
...
import (
    "log"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store/kv"
)

// Create train dataset as a pair of unique id and vector
var trainData []lsh.Record = ...
sampleSize := 100000
mean, std, _ := lsh.GetMeanStdSampledRecords(trainData, sampleSize)
var queryPoint []float64 = ...

// Define search parameters
lshConfig := lsh.Config{
	LshConfig: lsh.LshConfig{
		DistanceThrsh: 3000, // Distance threshold in non-normilized space
		MaxNN:         100,  // Maximum number of nearest neighbors to find
		BatchSize:     250,  // How much points to process in a single goroutine during the training phase
		MeanVec:       mean, // Optionally, you can use some bias vector, to "shift" the data before
                             // hash calculation on train and search
	},
	HasherConfig: lsh.HasherConfig{
		NPermutes:      10,  // Number of planes permutations to generate
		NPlanes:        12,  // Number of planes in a single permutation to generate
		BiasMultiplier: 1.0, // Sets how far from each other will planes be generated
		Dims:           784, // Space dimensionality
	},
}
// Store implementation, you can use yours
s := kv.NewKVStore()
// Metric implementation, L2 is good for the current dataset
metric := lsh.NewL2()
lshIndex, err := lsh.NewLsh(lshConfig, s, metric)
if err != nil {
	log.Fatal(err)
}

// Create search index; It will take some significant amount of time
lshIndex.Train(trainData)

// Perform search
closest, err := lshIndex.Search(queryPoint)
if err != nil {
	log.Fatal(err)
}
// Example of closest neighbors for 2D:
/*
[
	{096389f9-8d59-4799-a479-d8ec6d9de435 [0.07666666666666666 -0.003333333333333327]}
	{703eed19-cacc-47cf-8cf3-797b2576441f [0.06666666666666667 0.006666666666666682]}
	{1a447147-6576-41ef-8c2e-20fab35a9fc6 [0.05666666666666666 0.016666666666666677]}
	{b5c64ce0-0e32-4fa6-9180-1d04fdc553d1 [0.06666666666666667 -0.013333333333333322]}
]
*/
```  

### Building, testing and running benchmark  

To perform regular unit-tests, first install go deps:  
```
make install-go-deps
```  
And then run tests for `lsh` and `storage` packages:  
```
make test
```  
If you want to run benchmarks, where LSH compared to the regular NN search, first install hdf-5 for opening bench datasets:  
```
make install-hdf5 && make download-annbench-data
```  
And just run go test:  
```
make annbench
```  

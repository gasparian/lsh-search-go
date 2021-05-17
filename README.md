![lsh tests](https://github.com/gasparian/lsh-search-go/actions/workflows/test.yml/badge.svg?branch=master)
## lsh-search-go  

One of the most important things in the field of search and recommender systems is a problem of fast search in high-dimensional vector spaces.  
The simplest possible search algorithm would be just NN search with linear space and time complexity. But in practice we want to perform search in << O(N) time.  
So there is exist a set of algorithms called "approximate nearest neighbors" (aka "ANN"). And we can divide these set of algorithms into two subsets:  
 - [local sensetive hashing](https://en.wikipedia.org/wiki/Locality-sensitive_hashing) - or space partitioning methods;  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - like local search over proximity graphs, for example [hierarchical navigatable small world graphs](https://arxiv.org/pdf/1603.09320.pdf);  

I've decided to go with the LSH algorithm first, since it's pretty simple to understand and implement.  
So this repo contains library that has the functionality to create LSH index and perform search by the query vector.  

### Local sensitive hashing short reference   

Recipe for creating LSH search index:  
  1. Generate *k* random hyper planes.  
  2. Calculate bit hash for each point in a dataset, relying on it's position relative to the each generated plane. Store points with the same hash in a separate hash table.  
  3. Repeat the process *l* times --> so at the end we store *l* search indeces.  
  4. For each query point generate *l* hashes of length *k* and search for nearest neighbors in prepared hash tables.  

We can expect that nearby vectors have the higher probability to be in the same bucket.  
Storing all vectors in many hash tables requires much space, but searching for nearby vectors can then be done exponentially faster, as many distant vectors are not considered for reductions. So we will always have the trade-off between space usage and search time.  

Here are visual examples of the space partitioning for angular and non-angular distance metrics:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-go/blob/master/pics/non-biased.jpg" height=400/>  <img src="https://github.com/gasparian/lsh-search-go/blob/master/pics/biased.jpg" height=400/> </p>  

### API  

The storage and hashing parts are decoupled from each other.  
You need to implement only two interfaces:  
  1. [store](https://github.com/gasparian/lsh-search-go/blob/master/store/store.go), in order to use any storage you prefer.  
  2. [metric](https://github.com/gasparian/lsh-search-go/blob/master/lsh/lsh.go), to use your custom distance metric.  

LSH index object has super-simple interface:  
 - `NewLsh` for creating the new instance of index by given config;  
 - `Train(records []lsh.Record) error` for filling search index with vectors (`lsh.Record` must contain unique id and vector itself);  
 - `Search(query []float64) ([]lsh.Record, error)` to find `MaxNN` nearest neighbors to the query vector;  

// TODO: add more comments on code

And it's always better to show just the example of usage:  
```go
...
import (
    "log"
	lsh "github.com/gasparian/lsh-search-go/lsh"
	"github.com/gasparian/lsh-search-go/store/kv"
)

// Create train dataset as a pair of unique id and vector
var trainData []lsh.Record = ...
var queryPoint []float64 = ...

// Define search parameters
lshConfig := lsh.Config{
	LshConfig: lsh.LshConfig{
		DistanceThrsh: 3000,
		MaxNN:         100,
		BatchSize:     500,
	},
	HasherConfig: lsh.HasherConfig{
		NPermutes:      10,
		NPlanes:        20,
		BiasMultiplier: 1.0,
		Dims:           784,
	},
}
// Use pre-calculated dataset stats
lshConfig.Mean = []float64{...}
lshConfig.Std = []float64{...}
s := kv.NewKVStore()
metric := lsh.NewL2()
lshIndex, err := lsh.NewLsh(lshConfig, s, metric)
if err != nil {
	log.Fatal(err)
}

// Create search index
lshIndex.Train(trainData)

// Perform search
closest, err := lshIndex.Search(queryPoint)
if err != nil {
	log.Fatal(err)
}
```  

### Building and running benchmark  

Install hdf5 and go deps necessary for testing:  
```
make install-deps
```  
Download datasets:  
```
make download-annbench-data
```  
And just run test in `annbench` folder:  
```
make test path=./annbench
```  

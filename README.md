![lsh tests](https://github.com/gasparian/lsh-search-go/actions/workflows/test.yml/badge.svg?branch=master)
## lsh-search-go  

Here is my take on implementing approximate nearest neighbors search algorithm from scratch.  
The simplest possible search algorithm would be just NN search with linear space and time complexity. But in practice we want to perform search in << O(N) time.  
So there is exist a set of algorithms called "approximate nearest neighbors" (aka "ANN", here is cool [presentation](https://www.youtube.com/watch?v=cn15P8vgB1A&ab_channel=RioICM2018) by on of the key researches in that field). And we can divide it into two subsets:  
 - [locality-sensitive hashing](https://www.cs.princeton.edu/courses/archive/spring13/cos598C/Gionis.pdf), ([presentation](https://www.youtube.com/watch?v=t_8SpFV0l7A&ab_channel=MicrosoftResearch));  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - like local search over proximity graphs, for example [hierarchical navigatable small world graphs](https://arxiv.org/pdf/1603.09320.pdf) (great [presentation](https://www.youtube.com/watch?v=m8YfUnwJ1qw&t=313s&ab_channel=ODSAIRu) by Yandex Research);  

I've decided to go with the LSH algorithm first, since:  
  1. It's pretty simple to understand and implement it.  
  2. At some cases LSH can be more suitable for production usage, since it's index can be easily stored in any database.  
  The largest downside I see here - is that LSH needs too much memory to store the index.  

So this repo contains library that has the functionality to create LSH index and perform search by given query vector.  
And kudos to https://github.com/erikbern, who popularized the topic of ANN search in recent time, with [annoy](https://github.com/spotify/annoy) and [ann-benchmarks](https://github.com/erikbern/ann-benchmarks).  

### Locality sensitive hashing short reference   

LSH implies space partitioning with random hyperplanes and search across "buckets" formed by intersections of those planes.  
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
  - more planes permutations you create --> more space you use, more time for creating search index you need, but more accurate the model could become;  
  - more planes you generate --> more "buckets" with less points you get --> search becomes faster, but can be less accurate (more false negative errors, potentially);  
  - larger distance threshold you make --> more "candidate" points you will have during the search phase, so you can satisfy the "max. nearest neighbors" condition faster, but decrease accuracy.  

### API  

The storage and hashing parts are **decoupled** from each other.  
You need to implement only two interfaces:  
  1. [store](https://github.com/gasparian/lsh-search-go/blob/master/store/store.go), in order to use any storage you prefer.  
  2. [metric](https://github.com/gasparian/lsh-search-go/blob/d32f31c39cdb89cc8132901ddcdd7090a7454264/lsh/lsh.go#L20), to use your custom distance metric.  

LSH index object has a super-simple [interface](https://github.com/gasparian/lsh-search-go/blob/d32f31c39cdb89cc8132901ddcdd7090a7454264/lsh/lsh.go#L25):  
 - `NewLsh(config lsh.Config) (*LSHIndex, error)` is for creating the new instance of index by given config;  
 - `Train(records []lsh.Record) error` for filling search index with vectors (each `lsh.Record` must contain unique id and `[]float64` vector itself);  
 - `Search(query []float64, maxNN int, distanceThrsh float64) ([]lsh.Record, error)` to find `MaxNN` nearest neighbors to the query vector;  

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
sampleSize := 20000
mean, std, _ := lsh.GetMeanStdSampledRecords(trainData, sampleSize)
var queryPoint []float64 = ...

const (
    distanceThrsh = 3000 // Distance threshold in non-normilized space
    maxNN         = 100  // Maximum number of nearest neighbors to find
)

// Define search parameters
lshConfig := lsh.Config{
    IndexConfig: lsh.IndexConfig{
        BatchSize:     250,  // How much points to process in a single goroutine 
                             // during the training phase
        Bias:          mean, // Optionally, you can use some bias vector, 
                             // to "center" the data points before the
                             // hash calculation on train and search, 
                             // since planes are generated near the 
                             // center of coordinates.
                             // Usually I use mean vector here.
                             // (you can pass nil or the empty slice)
        Std:           std,  // Std used for standart scaling, can be nil or empty slice
                             // (e.g. when you use angular metrics)
        MaxCandidates: 1000, // Maximum number of points that will be stored
                             // in a min heap, where we then get MaxNN vectors
    },
    HasherConfig: lsh.HasherConfig{
        NPermutes: 10,  // Number of planes permutations to generate
        NPlanes:   12,  // Number of planes in a single permutation to generate
        Dims:      784, // Space dimensionality
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
closest, err := lshIndex.Search(queryPoint, maxNN, distanceThrsh)
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

### Testing  

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

Search parameters that you can find [here](https://github.com/gasparian/lsh-search-go/blob/master/annbench/annbench_test.go) has been selected empirically, based on precision and recall metrics measured on validation dataset. So don't rack your brains too much ;)  

### Results  

*TODO: fill tables with new measurements and datasets*  

I used 16 core/60Gb RAM machine for tests and in-memory store implementation (`kv.KVStore`).  
During experiments I used the following datasets:  

| Dataset           | N dimensions |  Train examples | Test examples |   Metric  |
|-------------------|:------------:|----------------:|:-------------:|:----------|
| Fashion MNIST     |      784     |      60000      |     10000     | Euclidean |
| NY times          |      256     |     290000      |     10000     | Cosine    |
| SIFT              |      128     |     1000000     |     10000     | Euclidean |
| GloVe             |      200     |     1183514     |     10000     | Cosine    |

[Fashion MNIST](https://github.com/zalandoresearch/fashion-mnist):  
| Approach                | Traning time, s | Avg. search time, ms |  Precision | Recall |
|-------------------------|:---------------:|---------------------:|:----------:|:-------|
| Exact nearest neighbors |       XXX       |         XXXXX        |     1      |   1    |
| LSH                     |      XXXXX      |         XXXX         |    XXXX    |  XXXX  |  

[NY times](https://archive.ics.uci.edu/ml/datasets/bag+of+words):  
| Approach                | Traning time, s | Avg. search time, ms |  Precision | Recall |
|-------------------------|:---------------:|---------------------:|:----------:|:-------|
| Exact nearest neighbors |       XXX       |        XXXXXX        |     1      |   1    |
| LSH                     |      XXXXX      |        XXXXX         |    XXXX    |  XXXX  |  


[SIFT](https://corpus-texmex.irisa.fr/):  
| Approach                | Traning time, s | Avg. search time, ms |  Precision | Recall |
|-------------------------|:---------------:|---------------------:|:----------:|:-------|
| Exact nearest neighbors |       XXX       |        XXXXXX        |     1      |   1    |
| LSH                     |      XXXXX      |        XXXXX         |    XXXX    |  XXXX  |  

[GloVe](http://nlp.stanford.edu/projects/glove/):  
| Approach                | Traning time, s | Avg. search time, ms |  Precision | Recall |
|-------------------------|:---------------:|---------------------:|:----------:|:-------|
| Exact nearest neighbors |       XXX       |        XXXXXX        |     1      |   1    |
| LSH                     |      XXXXX      |        XXXXX         |    XXXX    |  XXXX  |  

I picked parameters manually, to get the best tradeoff between speed and accuracy.  

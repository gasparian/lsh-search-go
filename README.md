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
My implementation is closer to the earlier versions of [Annoy](https://github.com/spotify/annoy), since during index construction, I picking up two random points and calculate the plane that lies in the middle between those points, and then repeat this process recursevely for points that lies on each side of this newly generated plane.   
So we can expect that nearby vectors have the higher probability to be in the same "bucket".  
To maximize the number of detected nearest neighbors during the search, usually it's enough to run ~10-100 plane generations (`NTrees` parameter) with different random seed.  
For each query vector we generate a set of hashes (one per single "tree"), based on dot-product between each plane and the query vector, and then we add all the candidates to the min-heap to finally get *k*-closest vectors to our query vector.  

Here is just super-simple visual example of space partitioning:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-go/blob/master/pics/biased.jpg" height=400/> </p>  

I prefer to use simple rules while tuning the algorithm:  
  - more "trees" you create --> more space you use, more time for creating search index you need, but more accurate the model could become (search time becomes unsignificantly higher too, though);  
  - decreasing the minimum amount of points in a "bucket" can make search faster, but it can be less accurate (more false negative errors, potentially);  
  - larger distance threshold you make --> more "candidate" points you will have during the search phase, so you can satisfy the "max. nearest neighbors" condition faster, but potentially decrease the accuracy.  

### API  

The storage and hashing parts are **decoupled** from each other.  
You need to implement only two interfaces:  
  1. [store](https://github.com/gasparian/lsh-search-go/blob/master/store/store.go), in order to use any storage you prefer.  
  2. [metric](https://github.com/gasparian/lsh-search-go/blob/master/lsh/lsh.go#L20), to use your custom distance metric.  

LSH index object has a super-simple [interface](https://github.com/gasparian/lsh-search-go/blob/d32f31c39cdb89cc8132901ddcdd7090a7454264/lsh/lsh.go#L25):  
 - `NewLsh(config lsh.Config) (*LSHIndex, error)` is for creating the new instance of index by given config;  
 - `Train(records [][]float64, ids []string) error` for filling search index with vectors (each `lsh.Record` must contain unique id and `[]float64` vector itself);  
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
var trainVecs [][]float64 = ...
var trainIds []string = ...
sampleSize := 20000
var queryPoint []float64 = ...

const (
    distanceThrsh = 2200 // Distance threshold in non-normilized space
    maxNN         = 10   // Maximum number of nearest neighbors to find
)

// Define search parameters
lshConfig := lsh.Config{
    IndexConfig: lsh.IndexConfig{
        BatchSize:     250,  // How much points to process in a single goroutine 
                             // during the training phase
        MaxCandidates: 5000, // Maximum number of points that will be stored
                             // in a min heap, where we then get MaxNN vectors
    },
    HasherConfig: lsh.HasherConfig{
        NTrees:   10,        // Number of planes trees (planes permutations) to generate
        KMinVecs: 500,       // Minimum number of points to stop growing planes tree
        Dims:     784,       // Space dimensionality
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
lshIndex.Train(trainVecs, trainIds)

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
And just run go test passing the needed test name:  
```
make annbench test=TestEuclideanFashionMnist
```  

Search parameters that you can find [here](https://github.com/gasparian/lsh-search-go/blob/master/annbench/annbench_test.go) has been selected "empirically", based on precision and recall metrics measured on validation dataset.  

### Results  

I used 16 core/60Gb RAM machine for tests and in-memory store implementation (`kv.KVStore`).  
The following datasets has been used:  

| Dataset           | N dimensions |  Train examples | Test examples |   Metric  |
|-------------------|:------------:|:---------------:|:-------------:|:----------|
| Fashion MNIST     |      784     |     60000       |     10000     | Euclidean |
| SIFT              |      128     |     1000000     |     10000     | Euclidean |
| NY times          |      256     |     290000      |     10000     | Cosine    |
| GloVe             |      200     |     1183514     |     10000     | Cosine    |  

I end up using **10 closest** nearest neighbors to calculate the metrics.   
Both precision and recall has been calculated using distance-based definition of these metrics, like in the [ANN-Benchmarks](https://arxiv.org/pdf/1807.05614.pdf) paper. See the example of "approximate" recall:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-go/blob/master/pics/recall_metric.png" width=600/> </p>  

In all experiments ε=0.05.  
I picked parameters manually, to get the best tradeoff between speed and accuracy.  

[Fashion MNIST](https://github.com/zalandoresearch/fashion-mnist):  
| Approach                | Traning time, s | Avg. search time, ms | Max Candidates |  Precision  |  Recall  |
|-------------------------|:---------------:|:--------------------:|:--------------:|:-----------:|:---------|
| Exact nearest neighbors |       0.37      |         767          |     30000      |    0.999    |  0.998   |
| LSH                     |      12.29      |         24.6         |     5000       |    0.947    |  0.946   |  

[SIFT](https://corpus-texmex.irisa.fr/):  
| Approach                | Traning time, s | Avg. search time, ms | Max Candidates |  Precision  |  Recall  |
|-------------------------|:---------------:|:--------------------:|:--------------:|:-----------:|:---------|
| Exact nearest neighbors |       6.38      |        4881          |     200000     |    0.990    |  0.990   |
| LSH                     |       653       |        104           |     10000      |    0.940    |  0.935   |  

So seems like it works with both datasets, giving the **30-50x** speed up, with just a slightly lower metrics values.  

For cosine datasets results are slightly worser - only **~10x** speed up. Also for both datasets I generated way more trees (>100) comparing to the previous two datasets.  
Unfortunately, I can't say yet exactly why it happening, but if you have an idea - get it touch with me!)  
[NY times](https://archive.ics.uci.edu/ml/datasets/bag+of+words):  
| Approach                | Traning time, s | Avg. search time, ms | Max Candidates |  Precision* |  Recall* |
|-------------------------|:---------------:|:--------------------:|:--------------:|:-----------:|:---------|
| Exact nearest neighbors |       1.79      |        1222          |     100000     |    0.958    |  0.957   |
| LSH                     |       ???       |        ???           |      ????      |    ?????    |  ?????   |  

[GloVe](http://nlp.stanford.edu/projects/glove/):  ???
| Approach                | Traning time, s | Avg. search time, ms | Max Candidates |  Precision  |  Recall  |
|-------------------------|:---------------:|:--------------------:|:--------------:|:-----------:|:---------|
| Exact nearest neighbors |       7.47      |        4782          |     100000     |    0.999    |   0.999  |
| LSH                     |       ????      |        ?????         |     ?????      |    ????     |   ????   |  

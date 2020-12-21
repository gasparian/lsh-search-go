# lsh-search-engine

### Proposal  

One of the both most common and interesting topics in machine learning is a problem of search in high-dimensional vector spaces.  
So the goal of this project is to build the simple vector search service.  
We want to perform the search in logarithmic time on average, and we have two basic groups of algorithms to do this:  
 - [local sensetive hashing](https://en.wikipedia.org/wiki/Locality-sensitive_hashing);  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - local search over proximity graphs, smth like hierarchical navigatable small world graphs;  

I've decided to go first with the LSH since it's pretty convenient to serialize the Hasher, store hashes and perform search over these hashes with some already existed and well-developed relational/document/key-value database. Generally speaking, we just need to implement the hashing algorithm and communication with the db. As for database - I've chosen mongodb to store both the benchmark dataset and hashes. Basically, it can be any database that you are familiar with.  

### Building and running  

To run the app, the only thing you need to be installed on your host machine - is docker engine.  
The list of objects inside the hdf5:  
 - `train` - train points;  
 - `test` - test points;  
 - `neighbors` - 100 nearest points for each point;  
 - `distances` - 100 distances (angular) to the nearest points;  

Everything runs inside a docker. Just launch it with:  
 - `./build_docker.sh && ./run_docker.sh` if you want to launch the main app;  
 - `cd ./db && ./launch.sh` if you want to launch the database (mongodb);  
Don't forget to add the actual db socket in the config.  

Also, for more convenient development, you can run the app locally. First, install deps:  
```
sudo apt-get install libhdf5-serial-dev
go mod init lsh-search-engine
go mod tidy
```  
Then compile and run, passing args from config file (targets are: `main.go` or `bench_data_prep_main.go` or `annbench_main.go`):  
```
go build -o ./main ./main.go
export $(grep -v '^#' config.env | xargs) && ./main
```  

In order to run [benchmarks](https://github.com/erikbern/ann-benchmarks), first download the benchmark dataset:  
```
wget http://ann-benchmarks.com/deep-image-96-angular.hdf5 -P ./data
```   
Then run the prepared script to load data from hdf5 to the mongodb:  
```
cd ./data
go mod tidy && build -o /usr/bin/run_prep_data run_prep_data.go
./run_data_prep
```  

### API Reference  
*TO DO*  

### Local sensitive hashing reference   

LSH algorithm implies generation of random plane equation coefs. So, depending on the similarity metric, often we just need to define "bias" coef "d" as zero (for "angular" metric) or non-zero.  
Also, we need to limit coefs range, based on data points deviation.
Here are example visualizations:  
<img src="https://github.com/gasparian/lsh-search-engine/blob/master/pics/non-biased.png" height=300 >  <img src="https://github.com/gasparian/lsh-search-engine/blob/master/pics/biased.png" height=300 >  

*TO DO: Complexity*

*TO DO: Quality metrics*

### Dev. plan:   

1. Prepare the [ANN benchmark dataset](http://ann-benchmarks.com/deep-image-96-angular.hdf5):  
    - ~~download dataset and write script for stats calculating using the hdf5;~~  
    - ~~add mongodb in project~~;  
    - ~~write a script to fill mongodb with the benchmarked dataset. Search index will be stored as new fields in the documents~~;  
    - add unit tests for basic db functions;  
2. Implement the LSH algorithm:  
    - ~~write functions for random planes generation~~;  
    - ~~write functions to perform basic vector operations~~;  
    - ~~add ability to store generated plane coefs on disk~~;  
    - add unit tests for public functions;  
3. Make main app API:  
    - ~~(build) the app needs to get dataset stats from the db (using mongo's aggregations) and iterate over the documents to update the search index collection~~;  
    - ~~(get) returns the NNs' "ids" of the queried data point~~;  
    - ~~(put) calculates the hashes and adds the document into the index collection~~;  
    - ~~(pop) clean the search index by the given point "name"~~;  
    - ~~rewrite the functions to save and load search index to hold the slice of Hasher objects instead of a single one~~;  
    - ~~add ability to store the build sync. status and the LSH index object in special collection in the same mongodb~~;  
    - ~~add proper logging~~;  
    - ~~rafactor mongo client~~;  
    - ~~update work with mongo client in bench prep code~~;  
    - ~~add decorator to handlers to measure the response time~~;  
    - ~~add hasher update on pop/put/get (store timestamp of the last change and compare the local one with the actual)~~;  
    - ~~make docker image even lighter - deploy only binaries using [docker scratch](https://github.com/phrozen/geohash/blob/master/server/Dockerfile)~~;  
    - replace vector with the gonum's blas based [implementation](https://godoc.org/gonum.org/v1/gonum/mat#VecDense);  
    - add unit tests for API methods;  
4. Add search quality testing using the test part of the benchmark dataset:  
    - ~~implement client~~;  
    - ~~write code for recall calculation, depending on threshold value~~;  
    - add time analisys and logging during the benchmark;  
    - add unit tests for metrics calculation funcs;  
5. Decouple db with "original" data and lsh:  
    - drop cursors where they're not needed;  
    - store vectors alongside with hashes;  
    - add ability to provide an array of vectors in the put and pop queries;  
    - add shell script for populating hashes from the client and decouple the lsh from db as much as possible;  
6. Make readme section on "how it works".  

### Notes:  
 - Below I'll show how to talk with mongodb via console, to make quick checks on the dataset.  
   So first you better check the monogodb [docs](https://docs.mongodb.com/manual/mongo/).  
   Then get inside the docker:  
   ```
   docker exec -ti mongo mongo
   ```  
   Select needed db:  
   ```
   show dbs
   use ann_bench
   ```  
   You can create/drop indexes:  
   ```
   db.train.createIndex({OrigId: 1})
   # index on array may take much time
   db.train.createIndex({featureVec: 1})
   db.train.dropIndex({featureVec: 1})
   ```  
   Empty find query will return all records, bounded by the limit value:  
   ```
   db.train.find().limit(2)
   ```  
   Also extra-useful thing is query analysis:  
   ```
   db.train.find("origId": 1).limit(2).explain("executionStats)
   ```  
   Clean the collection:  
   ```
   db.train.remove({})
   ```  
   Make aggregations, like getting mean and std vectors on the random data sample:  
   ```
   db.train.aggregate([
     {
       $sample: {
         size: 100000
       }
     },
     {
       $unwind: {
         path: "$featureVec",
         includeArrayIndex: "i"
       }
     },
     {
       $group: {
         _id: "$i",
         avg: {
           $avg: "$featureVec"
         },
         std: {
           $stdDevSamp: "$featureVec"
         }
       }
     },
     {
       $sort: {
         "_id": 1
       }
     },
     {
       $group: {
         _id: null,
         avg: {
           $push: "$avg"
         },
         std: {
           $push: "$std"
         }
       }
     }
   ])
   ```  
 - The mongodb go client is a connection pool already so it is thread safe: https://github.com/mongodb/mongo-go-driver/blob/master/mongo/client.go#L42  
 Quote from the code:  
```
 // Client is a handle representing a pool of connections to a MongoDB deployment. It is safe for concurrent use by
 // multiple goroutines.
 //
 // The Client type opens and closes connections automatically and maintains a pool of idle connections. For
 // connection pool configuration options, see documentation for the ClientOptions type in the mongo/options package.
```  
 - use mongo's `find` only with limiting, otherwise - db starts lagging. Not sure why...;  
 - monitor mongodb mem usage:  
 ```
 db.serverStatus().mem
    {
    	"bits" : 64,
    	"resident" : 907,
    	"virtual" : 1897,
    	"supported" : true,
    	"mapped" : 0,
    	"mappedWithJournal" : 0
    }
```  
 - if the mongo consumes too much ram while running inside the docker - just try to specify the WiredTiger mem cache  `-wiredTigerCacheSizeGB 2.5` to some lower value, like `(docker_mem_limit - 1) / 2`;  
 - don't forget to define indexes. In my case its `OrigID` and `Hashes.hash#` fields;  
# lsh-search-service

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
go mod init lsh-search-service
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

LSH algorithm implies generation of random plane equation coefs. So, depending on similarity metric, we just need to define "bias" coef "D" as zero (for "angular" metric) or non-zero (limited by the datapoints deviation).  
Here are visual examples of the planes generation for angular and non-angular distance metrics:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/non-biased.jpg" height=400/>  <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/biased.jpg" height=400/> </p>  

*TO DO: Complexity*

*TO DO: Quality metrics*

### Dev. plan:   

1. ~~Prepare the [ANN benchmark dataset](http://ann-benchmarks.com/deep-image-96-angular.hdf5)~~  
2. ~~Implement the LSH algorithm~~  
3. ~~Make main app API~~  
4. ~~Add search quality testing using the test part of the benchmark dataset~~  
5. Tests (~100 unit tests in total):  
    - lsh algorithm (~22);  
    - db (~40);  
    - client (~14);  
    - API (~14);  
    - Run benchmark! (~10);  
6. Additional things / refactoring:  
    - ~~decouple db and app~~;  
    - Add context with timeout everywhere in the db code;  
    - Make readme section on "how it works" (See readme to-do's);  

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
   db.train.createIndex({Info: 1})
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
   db.train.find("secondaryId": 1).limit(2).explain("executionStats)
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
 - don't forget to define indexes. In my case its `SecondaryID` and `Hashes.hash#` fields;  
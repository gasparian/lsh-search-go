# vector-search-go

### Proposal  

One of the both most common and interesting topics in machine learning is a problem of search in high-dimensional vector spaces.  
The goal of this project is to build the simple and reliable vector search service.  
It can be used as a core of recommender systems and semantic search applications.   
To create the search index, I'll use [LSH](https://en.wikipedia.org/wiki/Locality-sensitive_hashing) (local sensetive hashing).  

### Usage  

To run the app, the only thing you need to be installed on your host machine - is docker.  

If youre not logged as root, you can add yourself to the sudoers:  
```
sudo usermod -aG sudo ${USER}
```  

Download benchmark dataset:  
```
wget http://ann-benchmarks.com/deep-image-96-angular.hdf5 -P ./data
```   
The list of objects inside the hdf5:  
 - `train` - train points;  
 - `test` - test points;  
 - `neighbors` - 100 nearest points for each point;  
 - `distances` - 100 distances (angular) to the nearest points;  

Everything runs inside a docker. Just launch it with:  
 - `./launch.sh` if you want to launch the main app;  
 - `./db/launch.sh` if you want to launch the database;  
Don't forget to add the actual db socket in the config.  

In order to get stats of the test dataset (I've already placed the stats inside `config.toml`), after entering the running container, you must compile and run prepared script:  
```
cd ./data
go mod tidy && build -o /usr/bin/run_prep_data run_prep_data.go
./run_data_prep
```  
If you want to clean up connect to the mongodb through cli, first check out monogodb [manual](https://docs.mongodb.com/manual/mongo/).  
Then get inside the docker:  
```
docker exec -ti mongo mongo
```  
Then drop needed db or collections:  
```
show dbs
use ann_bench
db.vectors_train.find().limit(2)
```  
Clean the collection:  
```
db.vectors_train.remove({})
```  

Get mean and std vectors on random data sample:  
```
db.train.aggregate([
  {
    $sample: {
      size: 10000
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

### Reference   

LSH algorithm implies generation of random plane equation coefs. So, depending on the similarity metric, often we just need to define "bias" coef "d" as zero (for "angular" metric) or non-zero.  
Also, we need to limit coefs range, based on data points deviation.
Here are example visualizations:  
<img src="https://github.com/gasparian/vector-search-go/blob/master/pics/non-biased.png" height=300 >  <img src="https://github.com/gasparian/vector-search-go/blob/master/pics/biased.png" height=300 >  

*Complexety*

*Quality metrics*

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
    - (build) the app needs to get dataset stats from the db (using mongo's aggregations) and iterate over the documents to update the search index field. Should stop and block other running tasks;  
    - (get) app returns the NNs' "names" of the queried data point;  
    - (put) app adds new document into the db, and calculates the hashes;  
    - (pop) clean the search index by the given point "name";  
    - add unit tests for API methods;  
4. Add search quality testing using the test part of the benchmark dataset:  
    - write a script that sends the test data points to the seach index, and compares the answers with the ground truth;  
    - script must also write out the log with all needed mertrics (FPR, FNG, ROC, f1 and etc.);  
    - add unit tests for metrics calculation funcs;  
5. Add monitoring to the service:  
    - add perf check on the remotely running service;  
    - add metrics and dashboard for overall usage analytics;  

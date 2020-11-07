# vector-search-go

### Proposal  

One of the both most common and interesting topics in machine learning is a problem of search in high-dimensional vector spaces.  
The goal of this project is to build the simple and reliable vector search service.  
It can be used as a core of recommender systems and semantic search applications.   
To create the search index, I'll use [LSH](https://en.wikipedia.org/wiki/Locality-sensitive_hashing) (local sensetive hashing).  

### Usage  

To run the app, the only thing you need to be installed on your host machine - is docker.  

Download benchmark dataset:  
```
wget http://ann-benchmarks.com/deep-image-96-angular.hdf5 -P ./data
```   

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

### Reference   

LSH algorithm implies generation of random plane equation coefs. So, depending on the similarity metric, often we just need to define "bias" coef "d" as zero (for "angular" metric) or non-zero.  
Also, we need to limit coefs range, based on data points deviation.
Here are example visualizations:  
<img src="https://github.com/gasparian/vector-search-go/blob/master/pics/non-biased.png" height=300 >  <img src="https://github.com/gasparian/vector-search-go/blob/master/pics/biased.png" height=300 >  

### Dev. plan:   

1. Prepare the [ANN benchmark dataset](http://ann-benchmarks.com/deep-image-96-angular.hdf5):  
    - ~~download dataset and write script for stats calculating using the hdf5;~~  
    - ~~add mongodb in project~~;  
    - write a script to fill mongodb with the benchmarked dataset. Search index will be stored as new fields in the documents;  
    - add unit tests for basic db functions;  
2. Make comprehensive config file and parser for it (toml):  
    - mean and std vectors (?);  
    - url of the db;  
    - number of hyper-planes to split the space;  
    - target distance metric;  
    - number of LSH permutations (there will new buckets in the db with separate index);  
    - defaults (like number of results in a response and etc.);  
    - add unittests for the config parser (all functions);  
3. Implement the LSH algorithm:  
    - ~~write functions for random planes generation~~;  
    - ~~write functions to perform basic vector operations~~;  
    - ~~add ability to store generated plane coefs on disk~~;  
    - add unit tests for public functions;  
4. Make main app API:  
    - app must read and parses the config file;  
    - (create) the app needs to get dataset stats from the db (using mongo's aggregations) and iterate over the documents to update the search index field;  
    - (get) app returns the NNs' "names" of the queried point;  
    - (put) app adds new document into the db, and calculates the hashes;  
    - (pop) clean the search index by the given point "name";  
    - add unit tests for API methods;  
5. Add search quality testing using the test part of the benchmark dataset:  
    - write a script that sends the test data points to the seach index, and compares the answers with the ground truth;  
    - script must also write out the log with all needed mertrics (FPR, FNG, ROC, f1 and etc.);  
    - add unit tests for metrics calculation funcs;  
6. Add monitoring to the service:  
    - add perf check on the remotely running service;  
    - add metrics and dashboard for overall usage analytics;  

# vector-search-go

### Proposal  

One of the both most common and interesting topics in machine learning is a problem of search in high-dimensional vector spaces.  
The goal of this project is to build the fast vector search service with kv storage.  
To create the search index, I'll use LSH (local sensetive hashing).  

### Usage  
Download benchmark dataset:  
```
wget http://ann-benchmarks.com/deep-image-96-angular.hdf5 -P ./data
```   

Everything runs inside a docker. Just launch it with:  
 - `./launch.sh ./main.go` if you want to launch the main app;  
 - `./launch.sh ./db/main_db.go` if you want to launch the database;  
Don't forget to add the actual db socket in the config.  

After entering the running container, you can run `./run_data_prep` on the machine with benchmark dataset to get its' stats, if you didn't have one in the `config.toml` already.  

### Reference   
LSH algorithm implies generation of random plane equation coefs. So, depending on similarity metric, often we just need to define "bias" coef as zero (for "angular" metric) or non-zero. Here are example visualizations:  
<img src="https://github.com/gasparian/vector-search-go/blob/master/pics/non-biased.png" height=280 >  <img src="https://github.com/gasparian/vector-search-go/blob/master/pics/biased.png" height=280 >  

### Steps  

1. ~~Download [ANN benchmark](http://ann-benchmarks.com/deep-image-96-angular.hdf5) dataset and calculate mean and std~~.  
2. Prepare [db](https://github.com/boltdb/bolt) for creating new buckets with search indeces and storing search tree leaves - we need to keep resulting vectors/ids inside buckets.  
3. Make comprehensive config file and parser for it (toml?):  
    - mean and std vectors;  
    - url of the db;  
    - number of hyper-planes ot split;  
    - target distance metric;  
    - number of LSH permutations (there will new buckets in the db with separate index);  
4. Implement the LSH algorithm:  
    - angular distance metric;  
    - euclidian distance metric;  
5. Make API for building search index:  
    - client will open hdf5 and make get requests to the indexer;  
    - indexer must read config file and make LSH hashes by given vectors;   
6. Add to the API methods to query the nearest neighbours.  
7. Add monitoring to the service and convenient config files.  
 
### TO DO:  
 - API skeleton for the main app;  
 - API sekelton for the db, make it run independently, but using the same config;  
 - make config parser;  

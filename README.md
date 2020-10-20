# vector-search-go

### Proposal  

One of the both most common and interesting topics in machine learning is a problem of search in high-dimensional vector spaces.  
The goal of this project is to build the fast image search service with distributed storage.  

For the first one - I'll use [Tensorflow](https://syslog.ravelin.com/go-tensorflow-74d1101fab3f)+OpenCV/ffmpeg (??) for image processing.   

To create the search index, I'll use LSH (local sensetive hashing), with a distributed index.  
Myabe the good choice can be [boltdb](https://github.com/boltdb/bolt). To keep a tree there, you can keep two buckets with `parentId <-> Id` and `Id <-> value` pairs. Anyway investigation needed here!  

### Steps  

1. Dowbload ANN benchmark dataset and create a db to store it.  
2. Prepare db for storing search tree leaves - since we can have several objects with the same hash, let's keep paths to those images in buckets, assigned to the keys (== hashes).  
3. Implement the LSH algorithm.    
4. Make API for building search index (build it every time from scratch, using the db with extracted vectors).   
5. Add to the API methods to query the nearest neighbours.  
6. Add monitoring to the service and convenient config files.  
 
#### Useful links:  

- [ANN-benchmarks](https://github.com/erikbern/ann-benchmarks);  

#### Notes  


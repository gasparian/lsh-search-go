# visual-search-go

### Proposal  

One of the both most common and interesting topics in machine learning is a problem of search in high-dimensional vector spaces.  
The goal of this project is to build the fast image search service based on perceptual hashes.  
There are a lot of approaches, but, obviously, one of the most convenient ways to create perceptual hash is feature extraction using pretrained CNNs.  

Possible use cases:  
 - finding close dublicates;  
 - finding original images (for example high-res, having only low-res);  
 
The project consists of two major, separable parts:  
 - images hashing service;  
 - hashes search engine;  

For the first one - I'll use [Tensorflow](https://syslog.ravelin.com/go-tensorflow-74d1101fab3f)+OpenCV/ffmpeg (??) for image processing.   

To create the search index, I'll use LSH (local sensetive hashing), with a distributed index (ideally).  
Finally, I want to avoid keep the index in-memory, so I need to use some on-disk key-value storage, with a some sort of LRU/MFU in-memory cache.  
Myabe the good choice can be [boltdb](https://github.com/boltdb/bolt). To keep a tree there, you can keep two buckets with `parentId -> Id` and `Id -> value` pairs. Investigation needed  here!  
Here is a high-level diagram sketch of a whole service:  
<p align="center"> <img src="https://github.com/gasparian/visual-search-go/blob/master/imgs/random - images-search2.jpg" height="500" /> </p>  

### Steps  

1. Build API for images' feature extraction. Use [tiny-imagenet](http://cs231n.stanford.edu/tiny-imagenet-200.zip) dataset for experiments.  
2. Convert tiny-imagenet dataset into vectors.  
3. Implement naive O(N) KNN search. Store all vectors in some db or just in-memory map.  
4. Make simple web-interface for making image query and showing the search result.  
5. Implement the LSH algorithm, store the hash-map in memory.  
6. Build (... or use existing?) kv-store with `put` and `get` operations.  
7. Add an LRU/MFU cache to that.  
8. Make a distributed version of the kv-store (aka search index).  
9. Make API for building search index from scratch and expand existing operations with `pop`.  
10. Makesome sort of service config, docker files if needed + docs!  

 

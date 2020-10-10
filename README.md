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
Myabe the good choice can be [boltdb](https://github.com/boltdb/bolt). To keep a tree there, you can keep two buckets with `parentId <-> Id` and `Id <-> value` pairs. Anyway investigation needed here!  
Here is a high-level diagram sketch of a whole service:  
<p align="center"> <img src="https://github.com/gasparian/visual-search-go/blob/master/imgs/random - images-search2.jpg" height="500" /> </p>  

### Steps  

1. Build API for images' feature extraction. Use [tiny-imagenet](http://cs231n.stanford.edu/tiny-imagenet-200.zip) dataset for experiments.  
2. Make sure that we can use any type of pretrained CNN without any pain - so refactor the prediction API if needed.  
3. Convert tiny-imagenet dataset into vectors and store it at some kv-db. Maybe it's worth to use [this distrib-kv](https://github.com/YuriyNasretdinov/distribkv) implenetation based in bolt-db.  
4. Implement naive O(N) KNN search. Store all vectors in some db or just in-memory map.  
5. Make simple web-interface for making image query and showing the search result.  
6. Enhance the distrib-kv service with ability to store buckets inside buckets (this property needed to store search index).  
7. Add the consistent hashing to this db.  
8. Implement the LSH algorithm and learn how to store it in a right way!  
9. Make API for building search index from scratch.  
10. Make sure that all needed tests exists.  
11. Add monitoring.  
 

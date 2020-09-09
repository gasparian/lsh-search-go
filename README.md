# visual-search-go

### Proposal  

One of the most common (and interesting) topics in machine learning is a problem of search in high-dimensional vector spaces.  
The goal of this project is to build fast image search service based on perceptual hashes.  
There are a lot of approaches, but, obviously, one of the most convenient ways to create perceptual hash is feature extraction using pretrained CNNs.  

Possible use cases:  
 - finding dublicates;  
 - finding original images (for example high-res, having only low-res);  
 - finding sources where searched image appears;  
 
The project consists of two major, separable parts:  
 - images hashing service;  
 - hashes search engine;  

For the first one - I'll use [Tensorflow](https://syslog.ravelin.com/go-tensorflow-74d1101fab3f)+OpenCV(?) for image processing.   

To create the search index, I'll use LSH (local sensetive hashing), with a distributed index (ideally).  
Finally, I want to avoid keep the index in-memory, so I need to use some on-disk key-value storage, with a some sort of LRU/MFU in-memory cache.  
Here is a high-level diagram sketch of a whole service:  
<p align="center"> <img src="https://github.com/gasparian/visual-search-go/blob/master/imgs/random - images-search.jpg" height=500 /> </p>  


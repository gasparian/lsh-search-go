# visual-search-go

### Proposal  

One of the fundamental topics in the image-retrieval field is perceptual hashing.  
The goal of this project is to build fast image search service based on p-hashes.  
There are a lot of approaches, but, obviously, one of the most convenient ways to create perceptual hash - is feature extraction using pretrained CNNs.  

Use cases:  
 - finding dublicates;  
 - finding original images (for example high-res, having only low-res);  
 - finding sources where searched image appears;  
 
The project consists of two major, separable parts:  
 - images hashing service;  
 - hashes search engine;  

For the first one - I'll use Go, since it's good for developing robust web-services and has opencv and [tensorflow bindings](https://syslog.ravelin.com/go-tensorflow-74d1101fab3f) to be able to work with images.  
To create and use a search index, I'll use distributed approximate nearest neighbors search engine [open-sourced by microsoft](https://github.com/microsoft/SPTAG) (production-ready solution).  
The actual hashes' state should be stored in some cheap and easy-to-use key-value storage, like AWS S3.  
Here is a high-level diagram of a whole service:  
<p align="center"> <img src="https://github.com/gasparian/visual-search-go/blob/master/imgs/random - images-search.jpg" height=500 /> </p>  


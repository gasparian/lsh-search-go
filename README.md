# lsh-search-service

### Proposal  

One of the most important things in machine learning is a problem of fast search in high-dimensional vector spaces.  
So the goal of this project is to build a simple approximate nearest neighbors (ANN for short) search service.  
We have two basic groups of algorithms to perform the ANN search:  
 - [local sensetive hashing](https://en.wikipedia.org/wiki/Locality-sensitive_hashing);  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - local search over proximity graphs, smth like "hierarchical navigatable small world graphs";  

I've decided to go with the LSH algorithm since it's pretty simple to implement and you can store datapoints according to generated hashes in a simple key-value db.  

### Local sensitive hashing reference   

LSH algorithm implies generation of random plane equation coefs.  

// TODO: how the algorithm works  

Here are visual examples of the planes generation for angular and non-angular distance metrics:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/non-biased.jpg" height=400/>  <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/biased.jpg" height=400/> </p>  

// TODO: complexety analysis  

### Building and running  

// TODO: 

### [Dev. kanban board](https://github.com/gasparian/lsh-search-service/projects/1?fullscreen=true)  

### Links:  
 - fashion mnist dataset for l2 dist. tests (60000/10000x784, 100 neighbors): http://ann-benchmarks.com/fashion-mnist-784-euclidean.hdf5  
 - last.fm dataset for cosine dist. tests (292385/50000x65, 100 neighbors): http://ann-benchmarks.com/lastfm-64-dot.hdf5  

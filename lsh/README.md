# Locality sensetive hashing  

### Proposal  

One of the most important things in machine learning is a problem of fast search in high-dimensional vector spaces.  
So the goal of this project is to build a simple approximate nearest neighbors (ANN for short) search service.  
We have two basic groups of algorithms to perform the ANN search:  
 - [local sensetive hashing](https://en.wikipedia.org/wiki/Locality-sensitive_hashing);  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - local search over proximity graphs, smth like "hierarchical navigatable small world graphs";  

I've decided to go with the LSH algorithm since it's pretty simple to implement and you can store datapoints according to generated hashes in a simple key-value storage.  

### Local sensitive hashing reference   

LSH algorithm implies generation of random plane equation coefs.  

// TODO: how the algorithm works  

Here are visual examples of the planes generation for angular and non-angular distance metrics:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/non-biased.jpg" height=400/>  <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/biased.jpg" height=400/> </p>  

// TODO: complexety analysis  

### Building and running  

// TODO: 

### [Dev. kanban board](https://github.com/gasparian/lsh-search-service/projects/1?fullscreen=true)  

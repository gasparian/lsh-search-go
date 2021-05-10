![lsh tests](https://github.com/gasparian/lsh-search-go/actions/workflows/test.yml/badge.svg?branch=master)
## lsh-search-go  

### Proposal  

One of the most important things in machine learning is a problem of fast search in high-dimensional vector spaces.  
We have two basic groups of algorithms to perform the ANN search:  
 - [local sensetive hashing](https://en.wikipedia.org/wiki/Locality-sensitive_hashing) - one of the space partitioning methods;  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - local search over proximity graphs, for example [hierarchical navigatable small world graphs](https://arxiv.org/pdf/1603.09320.pdf);  

I've decided to go with the LSH algorithm first, since it's pretty simple to understand and implement.  
So this repo contains library that has the functionality to create LSH index and perform search by the query vector.  
The storage and hashing parts are decoupled from each other.  
I use [pure-kv-go](https://github.com/gasparian/pure-kv-go) for storing index. You can use whatever you want - you just need to implement [store](https://github.com/gasparian/lsh-search-go/blob/master/store/store.go) interface first.  

### Local sensitive hashing reference   

LSH algorithm implies generation of random plane equation coefs.  

// TODO: how the algorithm works in a few words  

Here are visual examples of the planes generation for angular and non-angular distance metrics:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-go/blob/master/pics/non-biased.jpg" height=400/>  <img src="https://github.com/gasparian/lsh-search-go/blob/master/pics/biased.jpg" height=400/> </p>  

// TODO: complexety analysis  

### Building and running  

Install hdf5:  
```
make install-hdf5
```  

// TODO: 

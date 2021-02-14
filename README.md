# lsh-search-service

### Proposal  

One of the both most common and interesting topics in machine learning is a problem of search in high-dimensional vector spaces.  
So the goal of this project is to build the simple vector search service.  
We want to perform the search in logarithmic time on average, and we have two basic groups of algorithms to do this:  
 - [local sensetive hashing](https://en.wikipedia.org/wiki/Locality-sensitive_hashing);  
 - [graph-based approaches](https://en.wikipedia.org/wiki/Small-world_network) - local search over proximity graphs, smth like hierarchical navigatable small world graphs;  

I've decided to go first with the LSH since it's pretty convenient to serialize the Hasher, store hashes and perform search over these hashes with some already existed and well-developed relational/document/key-value database. Generally speaking, we just need to implement the hashing algorithm and communication with the db. As for database - I've chosen mongodb to store both the benchmark dataset and hashes. Basically, it can be any database that you are familiar with.  

### Local sensitive hashing reference   

LSH algorithm implies generation of random plane equation coefs. So, depending on similarity metric, we just need to define "bias" component (usually referred as "D") as zero (for "angular" metric) or non-zero (limited by the datapoints deviation).  
Here are visual examples of the planes generation for angular and non-angular distance metrics:  
<p align="center"> <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/non-biased.jpg" height=400/>  <img src="https://github.com/gasparian/lsh-search-service/blob/master/pics/biased.jpg" height=400/> </p>  

// TO DO: https://github.com/gasparian/lsh-search-service/projects/1#card-54376167

// TO DO: https://github.com/gasparian/lsh-search-service/projects/1#card-54376189

### Building and running  

To run the app, the only thing you need to be installed on your host machine - is docker engine.  
Also, since this solution depends on mongodb, you need to run mongodb and provide it's address in the `config.env`. And don't forget to change the db authentication method (see the note in `/db/db.go`).  

Everything runs inside a docker. Just launch it with:  
 - `./build_docker.sh && ./run_docker.sh` if you want to launch the main app;  
 - `cd ./db && ./launch.sh` if you want to launch the db (suitable for local tests);  

Also, for more convenient development, you can run the app locally, without docker. First, install deps:  
```
sudo apt-get install libhdf5-serial-dev
go mod init lsh-search-service
go mod tidy
```  
Then compile and run the needed `*_main.go` file, passing args from config:  
```
go build -o ./main ./main.go
export $(grep -v '^#' config.env | xargs) && ./main
```  

In order to run [benchmarks](https://github.com/erikbern/ann-benchmarks), first download the benchmark dataset:  
```
wget http://ann-benchmarks.com/deep-image-96-angular.hdf5 -P ./data
```   
Here is the list of objects inside the downloaded hdf5:  
 - `train` - train points;  
 - `test` - test points;  
 - `neighbors` - 100 nearest points for each point;  
 - `distances` - 100 distances (angular) to the nearest points;  

Then run the prepared script to load data from hdf5 to the mongodb:  
```
cd ./data
go mod tidy && build -o /usr/bin/run_prep_data run_prep_data.go
./run_data_prep
```  

Running the unit tests:  
```
go test -test.v ./{PACKAGE}/
```  
Or run this to perform all exsiting unit tests:  
```
go test -test.v ./...
```  

And then you're ready to run the benchmark itself and see the result in stdout:  
```
go build -o ./annbench_main ./annbench_main.go
./annbench_main
```  

### API Reference   
// TO DO: https://github.com/gasparian/lsh-search-service/projects/1#card-54376146

### [Dev. kanban board](https://github.com/gasparian/lsh-search-service/projects/1?fullscreen=true)

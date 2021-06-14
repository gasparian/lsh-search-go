## Test datasets  

### Ann benchmarks  
Benchmark data stored in `hdf5` format.  
Typical file structure is: `distances`, `neighbors`, `test`, `train`.  
Neighbors dtype: int32, other data - float32. Test sets consists of "query points" with corresponding neighbors ids from the train sets.  
  - [fashion mnist](https://github.com/zalandoresearch/fashion-mnist) dataset:
    - distance metric - euclidean;  
    - N neighbors - 100; 
    - dimensions - 784 (28x28);  
    - train size - 60000;  
    - test size - 10000;  
  - [NY times](https://archive.ics.uci.edu/ml/datasets/bag+of+words):  
    - distance metric - angular;  
    - N neighbors - 100; 
    - dimensions - 256;  
    - train size - 290000;  
    - test size - 10000;  
  - [SIFT](https://corpus-texmex.irisa.fr/):  
    - distance metric - euclidean;  
    - N neighbors - 100; 
    - dimensions - 128;  
    - train size - 1000000;  
    - test size - 10000;  
  - [GloVe 200](http://nlp.stanford.edu/projects/glove/):  
    - distance metric - angular;  
    - N neighbors - 100; 
    - dimensions - 200;  
    - train size - 1183514;  
    - test size - 10000;

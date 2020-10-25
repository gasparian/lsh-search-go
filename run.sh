#!/bin/bash
docker run --rm -it \
           -v $PWD/db-dump:/go/src/vector-search-go/db-dump \
	   -v $PWD/data:/go/src/vector-search-go/data \
           vector-search-go

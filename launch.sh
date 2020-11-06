#!/bin/bash
docker build -t vector-search-go:latest .
docker run --rm -it \
           -p 8080:8080 \
	       -v $PWD/data:/go/src/vector-search-go/data \
           -cpus 4 \
           -m 4096 \
           vector-search-go

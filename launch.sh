#!/bin/bash
set -e
docker build -t vector-search-go:latest .
docker run --rm -it \
           -p 8080:8080 \
           -v $PWD/db:/go/src/vector-search-go/db\
	       -v $PWD/data:/go/src/vector-search-go/data \
           -e APP_PATH=$1 \
           -cpus 4 \
           -m 4096 \
           vector-search-go
wait

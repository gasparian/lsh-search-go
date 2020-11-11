#!/bin/sh
docker build -t vector-search-go:latest .
docker run --rm -it \
           -p 8080:8080 \
	       -v $PWD/db:/go/src/vector-search-go/db \
           --cpus 4 \
           -m 4096m \
           --env-file config.env \
           vector-search-go:latest

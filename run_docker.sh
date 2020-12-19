#!/bin/sh
docker run --rm -it \
           -p 8080:8080 \
           --cpus 4 \
           -m 4096m \
           --env-file config.env \
           vector-search-go:latest
           # -v $PWD/annbench:/go/src/vector-search-go/annbench \
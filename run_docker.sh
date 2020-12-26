#!/bin/sh
docker run --rm -it \
           -p 8080:8080 \
           --cpus 4 \
           -m 4096m \
           --env-file config.env \
           lsh-search-service:latest
           # -v $PWD/annbench:/go/src/lsh-search-service/annbench \
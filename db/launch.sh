#!/bin/sh
if [ "$(docker inspect mvertes/alpine-mongo 1> /dev/null)" != "" ] 
then
    docker pull mvertes/alpine-mongo
    wait $!
fi
docker run --rm -it \
           --name mongo \
           -p 27017:27017 \
           -v $PWD/mongo:/data/db \
           --cpus 4 \
           -m 6000m \
           mvertes/alpine-mongo

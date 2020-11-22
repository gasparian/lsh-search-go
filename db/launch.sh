#!/bin/sh
if [ "$(docker inspect mongo 1> /dev/null)" != "" ] 
then
    docker pull mongo:4.0-xenial
    wait $!
fi
docker run --rm -it \
           --name mongo \
           -p 27017:27017 \
           -v $PWD/mongo:/data/db \
           --cpus 4 \
           -m 6000m \
           mongo:4.0-xenial \
           --wiredTigerCacheSizeGB 2.5

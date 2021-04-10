#!/bin/sh
path=$1
if [ -z "$path" ] 
then 
    path=./...
fi
go clean -testcache
go test -v -cover -race $path
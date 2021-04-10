#!/bin/sh
go fmt ./...
# go mod tidy -v
go build -o main

#!/bin/sh
go mod tidy
go build -o /usr/bin/app ./main.go
./app
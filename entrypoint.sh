#!/bin/bash
go mod tidy
go build -o /usr/bin/app ${APP_PATH} 
./app
#!/bin/bash
go mod tidy && go build -o /usr/bin/app ${APP_PATH} 
go build -o /usr/bin/run_prep_data ./data/run_prep_data.go
./app
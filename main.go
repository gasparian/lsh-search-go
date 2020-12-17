package main

import (
	"net/http"
	"vector-search-go/app"
)

func main() {
	logger := app.GetNewLoggers()
	config, err := app.ParseEnv()
	if err != nil {
		logger.Err.Fatal(err.Error())
	}
	annServer, err := app.NewANNServer(logger, config)
	if err != nil {
		logger.Err.Fatal(err.Error())
	}
	defer annServer.MongoClient.Disconnect()

	http.HandleFunc("/", app.HealthCheck)
	http.HandleFunc("/build-index", annServer.BuildIndexerHandler)
	http.HandleFunc("/check-build", annServer.CheckBuildHandler)
	http.HandleFunc("/get-nn", annServer.GetNeighborsHandler)
	http.HandleFunc("/pop-hash", annServer.PopHashRecordHandler)
	http.HandleFunc("/put-hash", annServer.PutHashRecordHandler)
	logger.Err.Fatal(http.ListenAndServe(":8080", nil))
}

package main

import (
	"log"

	"net/http"
	"vector-search-go/app"
)

func main() {
	searchIndexHandler, err := app.BuildIndex()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer searchIndexHandler.MongoClient.Disconnect()

	http.HandleFunc("/", app.HealthCheck)
	http.HandleFunc("/get", searchIndexHandler.GetNeighbors)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

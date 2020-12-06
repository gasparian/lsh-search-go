package main

import (
	"log"

	"net/http"
	"vector-search-go/app"
)

func main() {
	searchIndexServer, err := app.NewSearchIndexServer()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer searchIndexServer.MongoClient.Disconnect()

	http.HandleFunc("/", app.HealthCheck)
	http.HandleFunc("/get", searchIndexServer.GetNeighbors)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

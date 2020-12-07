package main

import (
	"log"

	"net/http"
	"vector-search-go/app"
)

func main() {
	annServer, err := app.NewANNServer()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer annServer.MongoClient.Disconnect()

	http.HandleFunc("/", app.HealthCheck)
	http.HandleFunc("/get", annServer.GetNeighborsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

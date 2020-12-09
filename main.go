package main

import (
	"log"

	"net/http"
	"vector-search-go/app"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	annServer, err := app.NewANNServer()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer annServer.MongoClient.Disconnect()

	http.HandleFunc("/", app.HealthCheck)
	http.HandleFunc("/get-hash", annServer.GetNeighborsHandler)
	http.HandleFunc("/build-index", annServer.BuildIndexerHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

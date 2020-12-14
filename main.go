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
	http.HandleFunc("/build-index", annServer.BuildIndexerHandler)
	http.HandleFunc("/get-nn", annServer.GetNeighborsHandler)
	http.HandleFunc("/pop-hash", annServer.PopHashRecordHandler)
	http.HandleFunc("/put-hash", annServer.PutHashRecordHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

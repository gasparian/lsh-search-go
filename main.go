package main

import (
	"log"

	"net/http"
	"vector-search-go/app"
)

func main() {
	http.HandleFunc("/", app.HealthCheck)
	http.HandleFunc("/build", app.BuildIndex)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

package main

import (
	"log"

	"net/http"
	"vector-search-go/app"
)

func main() {
	http.HandleFunc("/", app.HealthCheck)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

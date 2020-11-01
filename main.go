package main

import (
	"flag"
	"log"
	"net/http"
	"vector-search-go/app"
)

var (
	dbLocation = flag.String("db-location", "./db/index.db", "The path to the bolt db database")
	httpAddr   = flag.String("http-addr", "127.0.0.1:8080", "HTTP host and port")
	configFile = flag.String("config", "config.toml", "Config file for the application")
)

func parseFlags() {
	flag.Parse()

	if *dbLocation == "" {
		log.Fatalf("Must provide db-location")
	}
}

func main() {
	parseFlags()

	http.HandleFunc("/", app.HealthCheck)

	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

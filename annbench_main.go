package main

import (
	"os"
	"strconv"

	annb "lsh-search-engine/annbench"
	cm "lsh-search-engine/common"
	"lsh-search-engine/db"
)

var (
	dbLocation = os.Getenv("MONGO_ADDR")
	dbName     = os.Getenv("DB_NAME")
)

func main() {
	logger := cm.GetNewLogger()
	config := db.Config{
		DbLocation: dbLocation,
		DbName:     dbName,
	}
	// thrshs := []float64{0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.7}
	thrshs := []float64{0.1} // DEBUG
	result, err := annb.Validate(config, thrshs)
	if err != nil {
		logger.Err.Fatal(err)
	}
	logger.Info.Println(result)
}

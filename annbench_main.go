package main

import (
	"os"
	"strconv"

	annb "lsh-search-engine/annbench"
	cl "lsh-search-engine/client"
	cm "lsh-search-engine/common"
	"lsh-search-engine/db"
)

var (
	dbLocation   = os.Getenv("MONGO_ADDR")
	dbName       = os.Getenv("DB_NAME")
	dbtimeOut, _ = strconv.Atoi(os.Getenv("DB_CLIENT_TIMEOUT"))
)

func main() {
	logger := cm.GetNewLogger()
	mongodb, err := db.New(
		db.Config{
			DbLocation: dbLocation,
			DbName:     dbName,
		},
	)
	if err != nil {
		logger.Err.Fatal(err)
	}

	benchClient := annb.BenchClient{
		Client: cl.New(cl.Config{
			ServerAddress: "http://192.168.0.132",
			Timeout:       dbtimeOut,
		}),
		Db:     mongodb,
		Logger: logger,
	}
	defer benchClient.Db.Disconnect()
	// thrshs := []float64{0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.7}
	thrshs := []float64{0.1} // DEBUG
	result, err := benchClient.Validate(thrshs)
	if err != nil {
		logger.Err.Fatal(err)
	}
	logger.Info.Println(result)
}

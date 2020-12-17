package app

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	cm "vector-search-go/common"
	"vector-search-go/db"
)

var (
	helloMessage = getHelloMessage()
)

// HealthCheck just checks that server is up and running;
// also gives back list of available methods
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(helloMessage)
}

// BuildIndexerHandler updates the existing db documents with the
// new computed hashes based on dataset stats;
// TO DO: after the indexer object is ready - we must call every other worker to load fresh model
// TO DO: make it in async way (in a goroutine)
func (annServer *ANNServer) BuildIndexerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	database := annServer.MongoClient.GetDb(annServer.Config.DbName)
	coll := database.Collection(annServer.Config.DataCollectionName)
	db.CreateCollection(database, annServer.Config.HelperCollectionName)
	helperColl := database.Collection(annServer.Config.HelperCollectionName)

	// NOTE: check if the previous build has been done
	helperRecord, err := db.GetHelperRecord(database.Collection(annServer.Config.HelperCollectionName), false)
	if err != nil {
		annServer.Logger.Warn.Println("Building index: seems like helper record does not exist yet")
	}
	if !helperRecord.IsBuildDone {
		annServer.Logger.Err.Println("Building index: aborting - previous build is not done yet")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Aborting - previous build is not done yet"))
		return
	}

	// NOTE: Start build process
	err = db.UpdateField(
		helperColl,
		bson.D{
			{"indexer", bson.D{
				{"$exists", true},
			}}},
		bson.D{
			{"$set", bson.D{
				{"isBuildDone", false}},
			}})

	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	convMean, convStd, err := db.GetAggregatedStats(coll)
	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	annServer.Logger.Info.Println(convMean) // DEBUG - check for not being [0]
	annServer.Logger.Info.Println(convStd)  // DEBUG - check for not being [0]

	err = annServer.Index.Generate(convMean, convStd)
	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	annServer.Logger.Info.Println(annServer.Index.Instances[0]) // DEBUG - check for not being [0]

	lshSerialized, err := annServer.Index.Dump()
	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// NOTE: Getting old hash collection name
	oldHelperRecord, err := db.GetHelperRecord(helperColl, false)
	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// NOTE: Generating and saving new hash collection name
	newHashCollName, err := cm.GetRandomID()
	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// NOTE: create new collection for storing the newly generated hashes, while keeping the old one
	err = db.CreateCollection(database, newHashCollName)
	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// NOTE: fill the new collection with pointers to documents (_id) and fields with hashes
	newHashColl := database.Collection(newHashCollName)
	cursor, err := db.GetCursor(
		coll,
		db.FindQuery{
			Limit: 0,
			Query: bson.D{},
		},
	)
	for cursor.Next(context.Background()) {
		hashesBatch, err := annServer.hashDbRecordsBatch(cursor, annServer.Config.BatchSize)
		if err != nil {
			annServer.Logger.Err.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = db.SetRecords(newHashColl, hashesBatch)
		if err != nil {
			annServer.Logger.Err.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// NOTE: create indexes for the all new fields
	hashesColl := database.Collection(newHashCollName)
	err = db.CreateIndexesByFields(hashesColl, annServer.Index.HashFieldsNames, false)
	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// NOTE: drop old collection with hashes
	if oldHelperRecord.HashCollName != "" {
		err = db.DropCollection(database, oldHelperRecord.HashCollName)
		if err != nil {
			annServer.Logger.Err.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// NOTE: update helper with new indexer and status
	err = db.UpdateField(
		helperColl,
		bson.D{
			{"indexer", bson.D{
				{"$exists", true},
			}}},
		bson.D{
			{"$set", bson.D{
				{"isBuildDone", true},
				{"indexer", lshSerialized},
				{"hashCollName", newHashCollName},
			}}})

	if err != nil {
		annServer.Logger.Err.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CheckBuildHandler checks the build status in the db
func (annServer *ANNServer) CheckBuildHandler(w http.ResponseWriter, r *http.Request) {
	database := annServer.MongoClient.GetDb(annServer.Config.DbName)
	helperColl := database.Collection(annServer.Config.HelperCollectionName)
	helperRecord, err := db.GetHelperRecord(helperColl, false)
	if err != nil {
		annServer.Logger.Err.Println("Checking build status: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Smth went wrong (may be the index doesn't exist)"))
		return
	}
	var message string = "Building finished"
	if !helperRecord.IsBuildDone {
		message = "Building in process"
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(message))
}

// PopHashRecordHandler drops vector from the search index
// curl -v http://localhost:8080/check?id=kd8f9wfhsdfs9df
func (annServer *ANNServer) PopHashRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		params := r.URL.Query()
		// NOTE: id generated from mongodb ObjectID with Hex() method
		id, ok := params["id"]
		if !ok || len(id) == 0 {
			annServer.Logger.Err.Println("Pop hash record: object id must be specified")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if id[0] == "" {
			annServer.Logger.Err.Println("Pop hash record: object id must be specified")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err := annServer.popHashRecord(id[0])
		if err != nil {
			annServer.Logger.Err.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			annServer.Logger.Err.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var input RequestData
		err = json.Unmarshal(body, &input)
		if err != nil {
			annServer.Logger.Err.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = annServer.popHashRecord(input.ID)
		if err != nil {
			annServer.Logger.Err.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// PutHashRecordHandler puts new vector to the search index (also updates the initial db??)
// curl -v -X POST -H "Content-Type: application/json" -d '{"id": "sdf87sdfsdf9s8dfb", "vec": []}' http://localhost:8080/put
func (annServer *ANNServer) PutHashRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		params := r.URL.Query()
		// NOTE: id generated from mongodb ObjectID with Hex() method
		id, ok := params["id"]
		if !ok || len(id) == 0 {
			annServer.Logger.Err.Println("Put hash record: object id must be specified")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if id[0] == "" {
			annServer.Logger.Err.Println("Put hash record: object id must be specified")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err := annServer.putHashRecord(id[0])
		if err != nil {
			annServer.Logger.Err.Println("Put hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			annServer.Logger.Err.Println("Put hash record: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var input RequestData
		err = json.Unmarshal(body, &input)
		if err != nil {
			annServer.Logger.Err.Println("Put hash record: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = annServer.putHashRecord(input.ID)
		if err != nil {
			annServer.Logger.Err.Println("Put hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// GetNeighborsHandler makes query to the db and returns all neighbors in the MaxDist
func (annServer *ANNServer) GetNeighborsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			annServer.Logger.Err.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var input RequestData
		err = json.Unmarshal(body, &input)
		if err != nil {
			annServer.Logger.Err.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		result, err := annServer.getNeighbors(input)
		if err != nil {
			annServer.Logger.Err.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		jsonResp, err := json.Marshal(result)
		if err != nil {
			annServer.Logger.Err.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	cm "github.com/gasparian/lsh-search-service/common"
	"github.com/gasparian/lsh-search-service/db"
	hashing "github.com/gasparian/lsh-search-service/lsh"
	"net/http"
	"strconv"
	"time"
)

var (
	helloMessage = getHelloMessage()
)

// Config holds general constants
type Config struct {
	BatchSize      int
	MaxHashesQuery int
	MaxNN          int
}

// ServiceConfig holds all needed variables to run the app
type ServiceConfig struct {
	Hasher hashing.Config
	Db     db.Config
	App    Config
}

// ANNServer holds Hasher itself and the mongo Client
type ANNServer struct {
	Hasher        *hashing.Hasher
	Mongo         db.MongoDatastore
	Logger        *cm.Logger
	Config        ServiceConfig
	LastBuildTime int64
	HashCollName  string
}

// HealthCheck just checks that server is up and running;
// also gives back list of available methods
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(helloMessage)
}

// BuildHasherHandler updates the existing db documents with the
// new computed hashes based on dataset stats;
func (annServer *ANNServer) BuildHasherHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			annServer.Logger.Err.Println("Build hasher: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var input cm.DatasetStats
		err = json.Unmarshal(body, &input)
		if err != nil {
			annServer.Logger.Err.Println("Build hasher: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err != nil {
			annServer.Logger.Err.Println("Build hasher: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

		go func() {
			err := annServer.BuildIndex(input)
			if err != nil {
				annServer.UpdateBuildStatus(
					db.HelperRecord{
						IsBuildDone:   false,
						BuildError:    err.Error(),
						LastBuildTime: time.Now().UnixNano(),
					},
				)
			}
		}()
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// CheckBuildHandler checks the build status in the db
func (annServer *ANNServer) CheckBuildHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := cm.ResponseData{Results: cm.BuildStatusUnknown}
	helperRecord, err := annServer.GetHelperRecord(false)
	if err != nil {
		annServer.Logger.Err.Println("Checking build status: " + err.Error())
		resp.Results = cm.BuildStatusError
		resp.Message = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		if !helperRecord.IsBuildDone && len(helperRecord.BuildError) == 0 {
			resp.Results = cm.BuildStatusInProgress
		} else if helperRecord.IsBuildDone && len(helperRecord.BuildError) == 0 {
			resp.Results = cm.BuildStatusDone
		} else if len(helperRecord.BuildError) > 0 {
			resp.Message = fmt.Sprintf("Build error: %s", helperRecord.BuildError)
		}
		w.WriteHeader(http.StatusOK)
	}
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
}

// GetHashCollSizeHandler checks the hashCollection size, returns `0` if it doesnt exist
func (annServer *ANNServer) GetHashCollSizeHandler(w http.ResponseWriter, r *http.Request) {
	size, err := annServer.GetHashCollSize()
	if err != nil {
		annServer.Logger.Err.Println("Checking hash coll. size: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := cm.ResponseData{Results: size}
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
	w.WriteHeader(http.StatusOK)
}

// PopHashRecordHandler drops vector from the search index
// curl -v http://localhost:8080/check?id=kd8f9wfhsdfs9df
func (annServer *ANNServer) PopHashRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		params := r.URL.Query()
		ids, ok := params["id"]
		if !ok || len(ids) == 0 {
			annServer.Logger.Err.Println("Pop hash record: object id must be specified")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id, err := strconv.ParseUint(ids[0], 10, 64)
		if err != nil {
			annServer.Logger.Err.Println("Pop hash record: cannot convert id to uint64 type")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = annServer.popHashRecord(id)
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

// PutHashRecordHandler puts new vector to the search index
// curl -v -X POST -H "Content-Type: application/json" -d '[{"id":"as8d7dhus", "vec":[...]}]' http://localhost:8080/put
func (annServer *ANNServer) PutHashRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			annServer.Logger.Err.Println("Put hash record: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var input []cm.RequestData
		err = json.Unmarshal(body, &input)
		if err != nil || len(input) == 0 {
			annServer.Logger.Err.Println("Put hash record: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = annServer.putHashRecord(input)
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

// GetNeighborsHandler makes query to the db and returns all neighbors
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
		var input cm.RequestData
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

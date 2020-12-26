package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	cm "lsh-search-service/common"
	"lsh-search-service/db"
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

// BuildHasherHandler updates the existing db documents with the
// new computed hashes based on dataset stats;
func (annServer *ANNServer) BuildHasherHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	go func() {
		err := annServer.BuildIndex()
		if err != nil {
			annServer.Mongo.UpdateBuildStatus(
				db.HelperRecord{
					IsBuildDone:   false,
					BuildError:    err.Error(),
					LastBuildTime: time.Now().UnixNano(),
				},
			)
		}
	}()
	w.WriteHeader(http.StatusOK)
}

// CheckBuildHandler checks the build status in the db
func (annServer *ANNServer) CheckBuildHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var message string = "Build status unknown"
	helperRecord, err := annServer.Mongo.GetHelperRecord(false)
	if err != nil {
		annServer.Logger.Err.Println("Checking build status: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		message = "Smth went wrong (may be the helper doesn't exist)"
	} else {
		if !helperRecord.IsBuildDone && len(helperRecord.BuildError) == 0 {
			message = "Build in process"
		} else if helperRecord.IsBuildDone && len(helperRecord.BuildError) == 0 {
			message = "Build done"
		} else if len(helperRecord.BuildError) > 0 {
			message = fmt.Sprintf("Build error: %s", helperRecord.BuildError)
		}
		w.WriteHeader(http.StatusOK)
	}
	resp := cm.ResponseData{Message: message}
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
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
		if len(id[0]) == 0 {
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
		var input cm.RequestData
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

// PutHashRecordHandler puts new vector to the search index
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
		if len(id[0]) == 0 {
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
		var input cm.RequestData
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

package app

import (
	"encoding/json"
	"log"
	"net/http"
)

// HealthCheck just checks that server is running
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var raw map[string]interface{}
	err := json.Unmarshal(HelloMessage, &raw)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
	}
	out, _ := json.Marshal(raw)
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

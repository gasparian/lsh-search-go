package app

import (
	"net/http"
)

// HealthCheck just checks that server is running
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

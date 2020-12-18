package common

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Logger holds several logger instances with different prefixes
type Logger struct {
	Warn *log.Logger
	Info *log.Logger
	Err  *log.Logger
}

// GetNewLogger creates an instance of all needed loggers
func GetNewLogger() *Logger {
	return &Logger{
		Warn: log.New(os.Stderr, "[ Warn ] ", log.LstdFlags|log.Lshortfile),
		Info: log.New(os.Stderr, "[ Info ] ", log.LstdFlags|log.Lshortfile),
		Err:  log.New(os.Stderr, "[ Error ] ", log.LstdFlags|log.Lshortfile),
	}
}

// GetRandomID generates random alphanumeric string
func GetRandomID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%x", b)
	return s, nil
}

// Decorator wraps an http.Handler with additional functionality
type Decorator func(http.Handler) http.Handler

// Decorate handler with all specified decorators
func Decorate(h http.Handler, decorators ...Decorator) http.Handler {
	// apply decorator backwards so that they are executed in declared order
	for i := len(decorators) - 1; i >= 0; i-- {
		h = decorators[i](h)
	}
	return h
}

// Timer logs the time taken processing the request
func Timer(logger *Logger) Decorator {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			h.ServeHTTP(w, r)
			elapsed := time.Since(start)
			logger.Info.Printf("elapsed time: %v (%v)\n", elapsed, r.URL)
		})
	}
}

package common

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
)

// Logger holds several logger instances with different prefixes
type Logger struct {
	Warn *log.Logger
	Info *log.Logger
	Err  *log.Logger
}

// GetNewLogger creates an instance of all needed loggers
func GetNewLogger() Logger {
	return Logger{
		Warn: log.New(os.Stderr, "[ Warn ]", log.LstdFlags|log.Lshortfile),
		Info: log.New(os.Stderr, "[ Info ]", log.LstdFlags|log.Lshortfile),
		Err:  log.New(os.Stderr, "[ Error ]", log.LstdFlags|log.Lshortfile),
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

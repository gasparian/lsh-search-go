package common

import (
	"bytes"
	"os"
	"testing"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := GetNewLogger()
	logger.Warn.SetOutput(&buf)
	logger.Info.SetOutput(&buf)
	logger.Err.SetOutput(&buf)
	defer func() {
		logger.Warn.SetOutput(os.Stderr)
		logger.Info.SetOutput(os.Stderr)
		logger.Err.SetOutput(os.Stderr)
	}()
	logger.Warn.Println("Test Warn")
	logger.Info.Println("Test Info")
	logger.Err.Println("Test Err")
	if buf.Len() == 0 {
		t.Fatal("Loggers returned nothing")
	}
}

func TestRandomID(t *testing.T) {
	_, err := GetRandomID()
	if err != nil {
		t.Fatalf("Cannot generate random id %v", err)
	}
}

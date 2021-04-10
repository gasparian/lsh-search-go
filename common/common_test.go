package common_test

import (
	"bytes"
	"gonum.org/v1/gonum/blas/blas64"
	cm "github.com/gasparian/lsh-search-service/common"
	"os"
	"testing"
)

func TestNewVec(t *testing.T) {
	var v blas64.Vector
	v = cm.NewVec([]float64{0.0, 42.0})
	if blas64.Asum(v) != 42.0 {
		t.Fatal("Corrupted conversion to blas vector")
	}
	v = cm.NewVec(nil)
	if blas64.Asum(v) != 0.0 {
		t.Fatal("Corrupted conversion to blas vector: nil should return empty vector")
	}
}

func TestL2(t *testing.T) {
	v1 := cm.NewVec([]float64{0.0, 0.0})
	v2 := cm.NewVec([]float64{-4.0, 3.0})
	l2 := cm.L2(v1, v2)
	if l2 != 5.0 {
		t.Fatal("L2 distance is wrong")
	}
}

func TestCosineSim(t *testing.T) {
	v1 := cm.NewVec([]float64{0.0, 1.0})
	v2 := cm.NewVec([]float64{0.0, 1.0})
	v3 := cm.NewVec([]float64{1.0, 0.0})
	v4 := cm.NewVec([]float64{0.0, -1.0})
	sim1 := cm.CosineSim(v1, v2)
	if sim1 != 0.0 {
		t.Fatal("Cosine similarity must be 0.0 for equal vectors")
	}
	sim2 := cm.CosineSim(v1, v3)
	if sim2 != 1.0 {
		t.Fatal("Cosine similarity must be 1.0 for orthogonal vectors")
	}
	sim3 := cm.CosineSim(v1, v4)
	if sim3 != 2.0 {
		t.Fatal("Cosine similarity must be 2.0 for multidirectional vectors")
	}
}

func TestIsZeroVec(t *testing.T) {
	v1 := cm.NewVec([]float64{0.0, 0.0})
	v2 := cm.NewVec([]float64{0.0, 1.0})
	if !cm.IsZeroVector(v1) {
		t.Fatal("Provided vector should be zero vector")
	}
	if cm.IsZeroVector(v2) {
		t.Fatal("Provided vector should be non-zero vector")
	}
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := cm.GetNewLogger()
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
	_, err := cm.GetRandomID()
	if err != nil {
		t.Fatalf("Cannot generate random id %v", err)
	}
}

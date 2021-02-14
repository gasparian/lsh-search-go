package lsh_test

import (
	cm "lsh-search-service/common"
	hashing "lsh-search-service/lsh"
	"testing"
)

func TestGetHash(t *testing.T) {
	hasherInstance := hashing.HasherInstance{
		Planes: []hashing.Plane{
			hashing.Plane{
				Coefs: cm.NewVec([]float64{1.0, 1.0, 1.0}),
				D:     5,
			},
		},
	}
	inpVec := cm.NewVec([]float64{5.0, 1.0, 1.0})
	meanVec := cm.NewVec([]float64{0.0, 0.0, 0.0})
	hash := hasherInstance.GetHash(inpVec, meanVec)
	if hash != 1 {
		t.Fatal("Wrong hash value, must be 1")
	}
	inpVec = cm.NewVec([]float64{1.0, 1.0, 1.0})
	hash = hasherInstance.GetHash(inpVec, meanVec)
	if hash != 0 {
		t.Fatal("Wrong hash value, must be 0")
	}
}

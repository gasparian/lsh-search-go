package kv

import (
	"errors"
	"reflect"
	"testing"
)

var (
	vectorsAreNotEqualErr   = errors.New("Vectors are not equal")
	cantFindVecKey          = errors.New("Can not find vector uid")
	wrongKeyErr             = errors.New("Returned wrong vector uid")
	iteratorNotClosedErr    = errors.New("Iterator not closed, but it should")
	vectorShouldNotExistErr = errors.New("Vector should not exist in a store")
)

func TestKvStore(t *testing.T) {
	store := NewKVStore()
	vecIds := map[string]bool{
		"0": true,
		"1": true,
	}
	vec := []float64{1, 2}

	t.Run("SetVector", func(t *testing.T) {
		for k := range vecIds {
			err := store.SetVector(k, vec)
			if err != nil {
				t.Fatal(err)
			}
		}
		vecReturned, err := store.GetVector("0")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(vec, vecReturned) {
			t.Error(vectorsAreNotEqualErr)
		}
	})

	t.Run("SetHash", func(t *testing.T) {
		for k := range vecIds {
			err := store.SetHash(0, 0, k)
			if err != nil {
				t.Fatal(err)
			}
		}

		it, err := store.GetHashIterator(0, 0)
		if err != nil {
			t.Fatal(err)
		}

		for range vecIds {
			id, ok := it.Next()
			if !ok {
				t.Error(cantFindVecKey)
			}
			if !vecIds[id] {
				t.Error(wrongKeyErr)
			}
		}
		_, ok := it.Next()
		if ok {
			t.Error(iteratorNotClosedErr)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		store.Clear()
		_, err := store.GetVector("0")
		if err == nil {
			t.Error(vectorShouldNotExistErr)
		}
	})
}

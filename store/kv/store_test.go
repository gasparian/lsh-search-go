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

	t.Run("SetVector", func(t *testing.T) {
		vec := []float64{1, 2}
		err := store.SetVector("0", vec)
		if err != nil {
			t.Fatal(err)
		}
		vecReturned, err := store.GetVector("0")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(vec, vecReturned) {
			t.Error(vectorsAreNotEqualErr)
		}
		err = store.SetVector("1", vec)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("SetHash", func(t *testing.T) {
		err := store.SetHash(0, 0, "0")
		if err != nil {
			t.Fatal(err)
		}
		store.SetHash(0, 0, "1")
		it, err := store.GetHashIterator(0, 0)
		if err != nil {
			t.Fatal(err)
		}
		id, ok := it.Next()
		if !ok {
			t.Error(cantFindVecKey)
		}
		if id != "0" {
			t.Error(wrongKeyErr)
		}
		id, _ = it.Next()
		if id != "1" {
			t.Error(wrongKeyErr)
		}
		_, ok = it.Next()
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

package kv

import (
	"errors"
	"github.com/gasparian/lsh-search-go/store"
	guuid "github.com/google/uuid"
	"sync"
)

type KVStore struct {
	mx sync.RWMutex
	m  map[string]map[string]interface{}
}

func NewKVStore() *KVStore {
	return &KVStore{
		m: make(map[string]map[string]interface{}),
	}
}

type KeysIterator struct {
	idx    int
	vecIds []string
}

func (it *KeysIterator) Next() (string, bool) {
	if it.idx > len(it.vecIds)-1 {
		return "", false
	}
	vecId := it.vecIds[it.idx]
	it.idx++
	return vecId, true
}

func getBucketName(perm int, hash uint64) string {
	return string(perm) + "_" + string(hash)
}

func (s *KVStore) SetVector(id string, vec []float64) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	if _, ok := s.m["vec"]; !ok {
		s.m["vec"] = make(map[string]interface{})
	}
	s.m["vec"][id] = vec
	return nil
}

func (s *KVStore) GetVector(id string) ([]float64, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()
	vecTmp := s.m["vec"][id]
	vec := vecTmp.([]float64)
	return vec, nil
}

func (s *KVStore) SetHash(permutation int, hash uint64, vecId string) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	bucketName := getBucketName(permutation, hash)
	if _, ok := s.m[bucketName]; !ok {
		s.m[bucketName] = make(map[string]interface{})
	}
	uid := guuid.NewString()
	s.m[bucketName][uid] = vecId
	return nil
}

func (s *KVStore) GetHashIterator(permutation int, hash uint64) (store.Iterator, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()

	bucketName := getBucketName(permutation, hash)
	val, ok := s.m[bucketName]
	if !ok {
		return nil, errors.New("Bucket not found")
	}
	i := 0
	vecIds := make([]string, len(val))
	for _, v := range val {
		vecIds[i] = v.(string)
		i++
	}
	it := &KeysIterator{
		idx:    0,
		vecIds: vecIds,
	}
	return it, nil
}

func (s *KVStore) Clear() error {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.m = make(map[string]map[string]interface{})
	return nil
}

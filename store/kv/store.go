package kv

import (
	"errors"
	"fmt"
	"github.com/gasparian/lsh-search-go/store"
	guuid "github.com/google/uuid"
	"sync"
)

var (
	bucketNotFoundErr = errors.New("Bucket not found")
	keyNotFoundErr    = errors.New("Key not found")
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
	vecIds chan string
}

func (it *KeysIterator) Next() (string, bool) {
	vecId, opened := <-it.vecIds
	if !opened {
		return "", false
	}
	return vecId, true
}

func getBucketName(perm int, hash uint64) string {
	return fmt.Sprintf("%v_%v", perm, hash)
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
	vecTmp, ok := s.m["vec"][id]
	if !ok {
		return nil, keyNotFoundErr
	}
	vec := vecTmp.([]float64)
	return vec, nil
}

func (s *KVStore) SetHash(bucketName, vecId string) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	if _, ok := s.m[bucketName]; !ok {
		s.m[bucketName] = make(map[string]interface{})
	}
	uid := guuid.NewString()
	s.m[bucketName][uid] = vecId
	return nil
}

func (s *KVStore) GetHashIterator(bucketName string) (store.Iterator, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()

	bucket, ok := s.m[bucketName]
	if !ok {
		return nil, bucketNotFoundErr
	}
	hashCh := make(chan string)
	go func() {
		for _, v := range bucket {
			hashCh <- v.(string)
		}
		close(hashCh)
	}()
	it := &KeysIterator{
		vecIds: hashCh,
	}
	return it, nil
}

func (s *KVStore) Clear() error {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.m = make(map[string]map[string]interface{})
	return nil
}

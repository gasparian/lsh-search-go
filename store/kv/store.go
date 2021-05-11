package kv

import (
	"github.com/gasparian/lsh-search-go/store"
	pkv "github.com/gasparian/pure-kv-go/client"
	guuid "github.com/google/uuid"
	"sync"
)

type KeysIterator struct {
	client     *pkv.Client
	bucketName string
}

func (it *KeysIterator) Next() (string, error) {
	_, vecId, err := it.client.Next(it.bucketName)
	if err != nil || vecId == nil {
		it.client.Close()
	}
	return vecId, err
}

type Config struct {
	Address string
	Timeout int
}

type PureKvStore struct {
	mx     sync.RWMutex
	config Config
	client *pkv.Client
}

func New(config Config) *PureKvStore {
	return &PureKvStore{
		config: config,
		client: pkv.New(config.Address, config.Timeout),
	}
}

func (p *PureKvStore) Start() error {
	err := p.client.Open()
	if err != nil {
		return err
	}
	err = p.client.Create("vecs")
	return nil
}

func (p *PureKvStore) Close() {
	p.client.Close()
}

func (p *PureKvStore) Clear() {
	p.client.DestroyAll()
}

func (p *PureKvStore) SetVector(id string, vec []float64) error {
	err := p.client.Set("vecs", id, vec)
	if err != nil {
		return err
	}
	return nil
}

func (p *PureKvStore) GetVector(id string) ([]float64, bool) {
	val := make([]float64, 0)
	tmpVal, ok := p.client.Get("vecs", id)
	if ok {
		val = tmpVal.([]float64)
	}
	return val, ok
}

func getBucketName(perm int, hash uint64) string {
	return string(permutation) + "_" + string(hash)
}

func (p *PureKvStore) SetHash(permutation int, hash uint64, vecId string) error {
	bucketName := getBucketName(permutation, hash)
	err := p.client.Create(bucketName)
	if err != nil {
		return err
	}
	uid := guuid.NewString()
	err = p.client.Set(bucketName, uid, vecId)
	if err != nil {
		return err
	}
	return nil
}

func (p *PureKvStore) GetHashIterator(permutation int, hash uint64) (store.Iterator, error) {
	bucketName := getBucketName(permutation, hash)
	err := p.client.MakeIterator(bucketName)
	if err != nil {
		return nil, err
	}
	it := &KeysIterator{
		client:     p.client,
		bucketName: bucketName,
	}
	return it, nil
}

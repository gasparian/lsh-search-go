package db

import (
	pkv "github.com/gasparian/pure-kv-go/client"
	"os"
	"strconv"
)

// Record __
// TODO: implement it inside the pure-kv-go code
type Record interface {
	Encode() []byte
	Decode(inp []byte)
}

// VectorRecord used to store the vectors to search in the mongodb
type VectorRecord struct {
	Key          string
	NeighborsIds []uint64
	FeatureVec   []float64
}

func (v *VectorRecord) Encode() []byte {
	return []byte{'a'}
}

func (v *VectorRecord) Decode([]byte) {
	return
}

// HashRecord stores generated hash and a key of the original vector
type HashRecord struct {
	Key       string
	Hash      uint64
	VectorKey string
}

func (v *HashRecord) Encode() []byte {
	return []byte{'a'}
}

func (v *HashRecord) Decode([]byte) {
	return
}

// HasherState holds the Hasher model and supplementary data
type HasherState struct {
	VectorsBucket    string
	Hasher           []byte
	IsBuildDone      bool
	BuildError       string
	HashCollName     string
	LastBuildTime    int64
	BuildElapsedTime int64
}

func (v *HasherState) Encode() []byte {
	return []byte{'a'}
}

func (v *HasherState) Decode([]byte) {
	return
}

// Db __
type Db struct {
	Client             *pkv.Client
	SampleSize         int
	CreateIndexTimeout int
}

// NewDb __
func NewDb(address string, timeout int) (*Db, error) {
	cli, err := pkv.InitPureKvClient(address, uint(timeout))
	if err != nil {
		return nil, err
	}
	return &Db{
		Client: cli,
	}, nil
}

// TODO: need to implement:
//
// Calculate mean, std of random sample of records (e.g. from bucket with Train data)
// Populate collection with set of vectors (e.g. Train/Test with "original" vectors)

// In the app code:
//
// CreateCollection --> client.Create(bucketName string) err
// GetCollection --> client.Get(bucketName, key string) val, ok
// DropCollection --> client.Del(bucketName, key string) err / client.Destroy(bucketName string) err
// GetCollSize --> client.Size(bucketName string) size, err

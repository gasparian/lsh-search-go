package storage

import (
	"bytes"
	srv "github.com/gasparian/pure-kv-go/server"
	"os"
	"testing"
	"time"
)

const (
	path = "/tmp/lsh-storage-test"
)

func prepareServer(t *testing.T) func() error {
	srv := srv.InitServer(
		6668, // port
		2,    // persistence timeout sec.
		32,   // number of shards for concurrent map
		path, // db path
	)
	go srv.Run()

	return srv.Close
}

func TestClient(t *testing.T) {
	defer os.RemoveAll(path)
	closeServer := prepareServer(t)
	time.Sleep(1 * time.Second) // just wait for server to be started

	s := New("0.0.0.0:6668", 500)
	err := s.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	bucketName := "test"
	key := "key1"
	valSet := []byte{'a'}

	t.Run("Set", func(t *testing.T) {
		err := s.Client.Create(bucketName)
		if err != nil {
			t.Error(err)
		}
		err = s.Client.Set(bucketName, key, valSet)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		val, ok := s.Client.Get(bucketName, key)
		if !ok {
			t.Error("Can't found the value")
		}
		if bytes.Compare(val, valSet) != 0 {
			t.Error("Returned value is not equal to the original one")
		}
	})

	t.Run("CloseSrv", func(t *testing.T) {
		err := closeServer()
		if err != nil {
			t.Error(err)
		}
	})
}

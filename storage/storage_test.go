package storage

import (
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
	defer prepareServer(t)()
	time.Sleep(1 * time.Second) // just wait for server to be started

	s := New("0.0.0.0:6668", 500)
	err := s.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	bucketName := "test"
	keys := []string{
		"key1", "key2",
	}
	valSet := []byte{'a'}

	t.Run("Add kv pair", func(t *testing.T) {
		err := s.Client.Create(bucketName)
		if err != nil {
			t.Error(err)
		}
		err = s.Client.Set(bucketName, keys[0], valSet)
		if err != nil {
			t.Error(err)
		}
	})
}

package storage

import (
	pkv "github.com/gasparian/pure-kv-go/client"
)

// Storage holds pure-kv rpc client, address and client timeout value
type Storage struct {
	Client  *pkv.Client
	address string
	timeout uint
}

// New creates new Storage object with the specified params
func New(address string, timeout int) *Storage {
	return &Storage{
		address: address,
		timeout: uint(timeout),
	}
}

// Open instantiates rpc client
func (s *Storage) Open() error {
	var err error
	s.Client, err = pkv.InitPureKvClient(s.address, s.timeout)
	if err != nil {
		return err
	}
	return nil
}

// Close shutowns rpc client
func (s *Storage) Close() {
	s.Client.Close()
}

// TODO: add funcs for bytes conversion
func Serialize() {

}

func Deserialize() {

}

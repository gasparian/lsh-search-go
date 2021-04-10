package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	cm "lsh-search-service/common"
)

// Config holds necessary constants for initiating the ANNClient
type Config struct {
	ServerAddress string
	Timeout       int
}

type methods struct {
	HealthCheck     string
	CheckBuild      string
	BuildIndex      string
	GetHashCollSize string
	GetNN           string
	PopHash         string
	PutHash         string
}

// ANNClient holds data needed to perform custom http requests
type ANNClient struct {
	ServerAddress string
	Client        http.Client
	Methods       methods
}

// New creates new instance of ANNClient
func New(config Config) ANNClient {
	return ANNClient{
		ServerAddress: config.ServerAddress,
		Client:        http.Client{Timeout: time.Duration(config.Timeout)},
		Methods: methods{
			HealthCheck:     config.ServerAddress + "/",
			CheckBuild:      config.ServerAddress + "/check-build",
			BuildIndex:      config.ServerAddress + "/build-index",
			GetHashCollSize: config.ServerAddress + "/get-index-size",
			GetNN:           config.ServerAddress + "/get-nn",
			PopHash:         config.ServerAddress + "/pop-hash?id=",
			PutHash:         config.ServerAddress + "/put-hash?id=",
		},
	}
}

// MakeRequest performs the http request with specified body
func (client *ANNClient) MakeRequest(method, url string, body io.Reader, target interface{}) error {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-type", "application/json")

	resp, err := client.Client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
		return errors.New("Response error")
	}

	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}

// CheckBuildStatus returns the actual status of the latest index build
func (client *ANNClient) CheckBuildStatus() (*cm.ResponseData, error) {
	target := &cm.ResponseData{}
	err := client.MakeRequest("GET", client.Methods.CheckBuild, nil, target)
	if err != nil {
		return nil, err
	}
	return target, nil
}

// GetHashCollSize returns number of documents in the hash collection
func (client *ANNClient) GetHashCollSize() (int64, error) {
	target := &cm.ResponseData{}
	err := client.MakeRequest("GET", client.Methods.GetHashCollSize, nil, target)
	if err != nil {
		return 0, err
	}
	size, ok := target.Results.(int64)
	if !ok {
		return 0, errors.New("GetHashCollSize: can't cast response to the int64 type")
	}
	return size, nil
}

// BuildHasher initiates hasher building process on server
func (client *ANNClient) BuildHasher(mean, std []float64) error {
	request := &cm.DatasetStats{
		Mean: mean,
		Std:  std,
	}

	jsonRequest, err := json.Marshal(request)
	if err != nil {
		return err
	}
	err = client.MakeRequest("POST", client.Methods.BuildIndex, bytes.NewBuffer(jsonRequest), nil)
	if err != nil {
		return err
	}
	return nil
}

// PopHash drops specified hash from the search index
func (client *ANNClient) PopHash(id uint64) error {
	stringID := strconv.FormatUint(id, 10)
	err := client.MakeRequest("GET", client.Methods.PopHash+stringID, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// PutHashes adds specified hash to the search index
func (client *ANNClient) PutHashes(requestData []cm.RequestData) error {
	jsonRequest, err := json.Marshal(requestData)
	if err != nil {
		return err
	}
	err = client.MakeRequest("POST", client.Methods.PutHash, bytes.NewBuffer(jsonRequest), nil)
	if err != nil {
		return err
	}
	return nil
}

// GetNeighbors gets the nearest neighbors for the query point (by ID or feature vector)
func (client *ANNClient) GetNeighbors(vec []float64) ([]uint64, error) {
	request := &cm.RequestData{
		Vec: vec,
	}
	jsonRequest, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	target := &cm.ResponseData{}
	err = client.MakeRequest("POST", client.Methods.CheckBuild, bytes.NewBuffer(jsonRequest), target)
	if err != nil {
		return nil, err
	}
	neighbors, ok := target.Results.([]uint64)
	if !ok {
		return nil, errors.New("GetNeighbors: can't cast result to the []uint64 type")
	}
	return neighbors, nil
}

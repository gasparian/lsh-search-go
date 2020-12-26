package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	cm "lsh-search-service/common"
)

// New creates new instance of ANNClient
func New(config Config) ANNClient {
	return ANNClient{
		ServerAddress: config.ServerAddress,
		Client:        http.Client{Timeout: time.Duration(config.Timeout)},
		Methods: methods{
			HealthCheck: config.ServerAddress + "/",
			CheckBuild:  config.ServerAddress + "/check-build",
			BuildIndex:  config.ServerAddress + "/build-index",
			GetNN:       config.ServerAddress + "/get-nn",
			PopHash:     config.ServerAddress + "/pop-hash?id=",
			PutHash:     config.ServerAddress + "/put-hash?id=",
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

// BuildHasher returns the actual status of the latest index build
func (client *ANNClient) BuildHasher() error {
	err := client.MakeRequest("GET", client.Methods.BuildIndex, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// PopHash drops specified hash from the search index (GET)
func (client *ANNClient) PopHash(id string) error {
	err := client.MakeRequest("GET", client.Methods.PopHash+id, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// PutHash adds specified hash to the search index
func (client *ANNClient) PutHash(id string) error {
	err := client.MakeRequest("GET", client.Methods.PutHash+id, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// GetNeighbors gets the nearest neighbors for the query point (by ID or feature vector)
func (client *ANNClient) GetNeighbors(vec []float64) (*cm.ResponseData, error) {
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
	return target, nil
}

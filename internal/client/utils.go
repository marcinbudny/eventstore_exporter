package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	log "github.com/sirupsen/logrus"
)

func (client *EventStoreStatsClient) esHTTPGet(ctx context.Context, path string, acceptNotFound bool) (result []byte, err error) {
	url := client.config.EventStoreURL + path

	log.WithField("url", url).Debug("GET request to EventStore")

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if client.config.EventStoreUser != "" && client.config.EventStorePassword != "" {
		req.SetBasicAuth(client.config.EventStoreUser, client.config.EventStorePassword)
	}
	req.Header.Add("Accept", "application/json")
	response, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == 404 && acceptNotFound {
		return nil, nil
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP call to %s resulted in status code %d", url, response.StatusCode)
	}

	buf, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func esHTTPGetAndParse[TResponse any](ctx context.Context, client *EventStoreStatsClient, path string, acceptNotFound bool) (TResponse, error) {
	var response TResponse

	jsonBytes, err := client.esHTTPGet(ctx, path, acceptNotFound)
	if err != nil || jsonBytes == nil {
		return response, err
	}

	err = json.Unmarshal(jsonBytes, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func readSingleEvent(ctx context.Context, grpcClient *esdb.Client, stream string, options esdb.ReadStreamOptions) (*esdb.ResolvedEvent, error) {
	read, err := grpcClient.ReadStream(ctx, stream, options, 1)
	if err != nil {
		return nil, err
	}

	defer read.Close()
	event, err := read.Recv()
	if err != nil {
		return nil, err
	}

	return event, nil
}

func readSingleEventFromAll(ctx context.Context, grpcClient *esdb.Client, options esdb.ReadAllOptions) (*esdb.ResolvedEvent, error) {
	read, err := grpcClient.ReadAll(ctx, options, 1)
	if err != nil {
		return nil, err
	}

	defer read.Close()
	event, err := read.Recv()
	if err != nil {
		return nil, err
	}

	return event, nil
}

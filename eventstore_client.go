package main

import (
	"net/http"
	"io/ioutil"
)

var (
	client http.Client
)

func initializeClient() {
	client = http.Client{
		Timeout:   timeout,
	}
}

type getResult struct {
	result []byte
	err    error
}

type stats struct {
	serverStats		[]byte
	gossipStats		[]byte
	projectionStats	[]byte
}

func getStats() (*stats, error) {
	serverStatsChan := get("/stats")
	gossipStatsChan := get("/gossip")
	projectionStatsChan := get("/projections/any")

	serverStatsResult := <-serverStatsChan
	gossipStatsResult := <-gossipStatsChan
	projectionStatsResult := <-projectionStatsChan

	if serverStatsResult.err != nil {
		return nil, serverStatsResult.err
	}
	if gossipStatsResult.err != nil {
		return nil, serverStatsResult.err
	}
	if projectionStatsResult.err != nil {
		return nil, serverStatsResult.err
	}

	return &stats{
		serverStatsResult.result,
		gossipStatsResult.result,
		projectionStatsResult.result,
	}, nil
}

func get(path string) (<-chan getResult) {
	url := eventStoreURL + path

	result := make(chan getResult)

	go func() {
		log.WithField("url", url).Debug("GET request to EventStore")

		response, err := client.Get(url)
		if err != nil {
			result <- getResult { nil, err }
			return
		}
		defer response.Body.Close()

		buf, err := ioutil.ReadAll(response.Body)
		if err != nil {
			result <- getResult { nil, err }
			return
		}

		result <- getResult { buf, nil }
	}()

	return result
}
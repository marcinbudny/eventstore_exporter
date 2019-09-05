package main

import (
	"io/ioutil"
	"net/http"
)

var (
	client http.Client
)

func initializeClient() {
	client = http.Client{
		Timeout: timeout,
	}
}

type getResult struct {
	result []byte
	err    error
}

type stats struct {
	serverStats        []byte
	gossipStats        []byte
	projectionStats    []byte
	info               []byte
	subscriptionsStats []byte
}

func getStats() (*stats, error) {
	serverStatsChan := get("/stats")
	projectionStatsChan := get("/projections/all-non-transient")
	infoChan := get("/info")
	subscriptionsStatsChan := get("/subscriptions")

	serverStatsResult := <-serverStatsChan
	if serverStatsResult.err != nil {
		return nil, serverStatsResult.err
	}

	projectionStatsResult := <-projectionStatsChan
	if projectionStatsResult.err != nil {
		return nil, projectionStatsResult.err
	}

	infoResult := <-infoChan
	if infoResult.err != nil {
		return nil, infoResult.err
	}

	subscriptionsStatsResult := <-subscriptionsStatsChan
	if subscriptionsStatsResult.err != nil {
		return nil, subscriptionsStatsResult.err
	}

	gossipStatsResult := getResult{}
	if isInClusterMode() {
		gossipStatsChan := get("/gossip")

		gossipStatsResult = <-gossipStatsChan
		if gossipStatsResult.err != nil {
			return nil, gossipStatsResult.err
		}
	}

	return &stats{
		serverStatsResult.result,
		gossipStatsResult.result,
		projectionStatsResult.result,
		infoResult.result,
		subscriptionsStatsResult.result,
	}, nil
}

func get(path string) <-chan getResult {
	url := eventStoreURL + path

	result := make(chan getResult)

	go func() {
		log.WithField("url", url).Debug("GET request to EventStore")

		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		req.SetBasicAuth(eventStoreUser, eventStorePassword)
		response, err := client.Do(req)
		if err != nil {
			result <- getResult{nil, err}
			return
		}
		defer response.Body.Close()

		buf, err := ioutil.ReadAll(response.Body)
		if err != nil {
			result <- getResult{nil, err}
			return
		}

		result <- getResult{buf, nil}
	}()

	return result
}

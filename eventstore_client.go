package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	client http.Client
)

func initializeClient() {
	if insecureSkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = http.Client{
			Timeout:   timeout,
			Transport: tr,
		}
	} else {
		client = http.Client{
			Timeout: timeout,
		}
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
	serverStatsChan := get("/stats", false)
	projectionStatsChan := get("/projections/all-non-transient", true)
	infoChan := get("/info", false)
	subscriptionsStatsChan := get("/subscriptions", false)

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
		gossipStatsChan := get("/gossip", false)

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

func get(path string, acceptNotFound bool) <-chan getResult {
	url := eventStoreURL + path

	result := make(chan getResult)

	go func() {
		log.WithField("url", url).Debug("GET request to EventStore")

		req, err := http.NewRequest("GET", url, nil)
		if eventStoreUser != "" && eventStorePassword != "" {
			req.SetBasicAuth(eventStoreUser, eventStorePassword)
		}
		req.Header.Add("Accept", "application/json")
		response, err := client.Do(req)
		if err != nil {
			result <- getResult{nil, err}
			return
		}
		defer response.Body.Close()

		if response.StatusCode == 404 && acceptNotFound {
			result <- getResult{nil, nil}
		}

		if response.StatusCode >= 400 {
			result <- getResult{nil, fmt.Errorf("HTTP call to %s resulted in status code %d", url, response.StatusCode)}
		}

		buf, err := ioutil.ReadAll(response.Body)
		if err != nil {
			result <- getResult{nil, err}
			return
		}

		result <- getResult{buf, nil}
	}()

	return result
}

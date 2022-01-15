package client

import (
	"fmt"
	"io/ioutil"
	"net/http"

	jp "github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

func getFloat(json []byte, keys ...string) float64 {
	value, _ := jp.GetFloat(json, keys...)
	return value
}

func getString(json []byte, keys ...string) string {
	value, _ := jp.GetString(json, keys...)
	return value
}

func getBoolean(json []byte, keys ...string) bool {
	value, _ := jp.GetBoolean(json, keys...)
	return value
}

func getInt(json []byte, keys ...string) int64 {
	value, _ := jp.GetInt(json, keys...)
	return value
}

func (client *EventStoreStatsClient) get(path string, acceptNotFound bool) (result []byte, err error) {
	url := client.config.EventStoreURL + path

	log.WithField("url", url).Debug("GET request to EventStore")

	req, _ := http.NewRequest("GET", url, nil)
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

	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

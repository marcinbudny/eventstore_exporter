package client

import (
	"github.com/hashicorp/go-version"
)

type EventStoreVersion string

type getEsVersionResult struct {
	esVersion EventStoreVersion
	err       error
}

func (client *EventStoreStatsClient) getEsVersion() <-chan getEsVersionResult {
	result := make(chan getEsVersionResult, 1)

	go func() {
		if infoJson, err := client.esHttpGet("/info", false); err == nil {
			result <- getEsVersionResult{
				esVersion: EventStoreVersion(getString(infoJson, "esVersion")),
			}
		} else {
			result <- getEsVersionResult{err: err}
		}
	}()

	return result
}

func (esVersion EventStoreVersion) IsAtLeastVersion(minVersion string) bool {
	ver, err := version.NewVersion(string(esVersion))
	if err != nil {
		return false
	}
	minSupportedVersion, err := version.NewVersion(minVersion)
	if err != nil {
		return false
	}

	return ver.GreaterThanOrEqual(minSupportedVersion)
}

func (esVersion EventStoreVersion) IsVersionLowerThan(maxVersion string) bool {
	ver, err := version.NewVersion(string(esVersion))
	if err != nil {
		return false
	}
	minSupportedVersion, err := version.NewVersion(maxVersion)
	if err != nil {
		return false
	}

	return ver.LessThan(minSupportedVersion)
}

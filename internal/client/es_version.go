package client

import (
	"context"

	"github.com/hashicorp/go-version"
)

type EventStoreVersion string

type getEsVersionResult struct {
	esVersion EventStoreVersion
	err       error
}

func (client *EventStoreStatsClient) getEsVersion(ctx context.Context) (EventStoreVersion, error) {
	if infoJson, err := client.esHttpGet(ctx, "/info", false); err == nil {
		return EventStoreVersion(getString(infoJson, "esVersion")), nil
	} else {
		return "", err
	}
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

package client

import "github.com/hashicorp/go-version"

type EventStoreVersion string

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

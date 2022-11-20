package client

import (
	"fmt"

	jp "github.com/buger/jsonparser"
)

type ClusterMemberType int

const (
	Leader ClusterMemberType = iota
	Follower
	ReadOnlyReplica
	Clone
	Unknown
)

type MemberStats struct {
	MemberName string
	IsAlive    bool
}

type ClusterStats struct {
	CurrentNodeMemberType ClusterMemberType
	Members               []MemberStats
}

func (client *EventStoreStatsClient) getClusterStats() (stats *ClusterStats, err error) {
	if client.config.IsInClusterMode() {

		infoJson, err := client.esHttpGet("/info", false)
		if err != nil {
			return nil, err
		}

		gossipJson, err := client.esHttpGet("/gossip", false)
		if err != nil {
			return nil, err
		}

		return &ClusterStats{
			CurrentNodeMemberType: getClusterMemberType(infoJson),
			Members:               getMemberStats(gossipJson),
		}, nil

	} else {
		return &ClusterStats{}, nil
	}
}

func getClusterMemberType(infoJson []byte) ClusterMemberType {
	memberTypeString := getString(infoJson, "state")
	switch memberTypeString {
	case "leader":
		return Leader
	case "follower":
		return Follower
	case "readonlyreplica":
		return ReadOnlyReplica
	case "clone":
		return Clone
	default:
		return Unknown
	}
}

func getMemberStats(gossipJson []byte) []MemberStats {
    var members []MemberStats

	_, _ = jp.ArrayEach(gossipJson, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		ip := getString(jsonValue, "httpEndPointIp")
		port := getInt(jsonValue, "httpEndPointPort")

		members = append(members, MemberStats{
			MemberName: fmt.Sprintf("%s:%d", ip, port),
			IsAlive:    getBoolean(jsonValue, "isAlive"),
		})
	}, "members")

	return members
}

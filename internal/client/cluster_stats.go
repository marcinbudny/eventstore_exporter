package client

import (
	"fmt"

	jp "github.com/buger/jsonparser"
)

type getClusterStatsResult struct {
	cluster *ClusterStats
	err     error
}

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

func (esClient *EventStoreStatsClient) getClusterStats() <-chan getClusterStatsResult {
	stats := make(chan getClusterStatsResult, 1)

	go func() {
		if esClient.config.IsInClusterMode() {

			infoJson, err := esClient.get("/info", false)
			if err != nil {
				stats <- getClusterStatsResult{err: err}
				return
			}

			gossipJson, err := esClient.get("/gossip", false)
			if err != nil {
				stats <- getClusterStatsResult{err: err}
				return
			}

			stats <- getClusterStatsResult{
				cluster: &ClusterStats{
					CurrentNodeMemberType: getClusterMemberType(infoJson),
					Members:               getMemberStats(gossipJson),
				},
			}
		} else {
			stats <- getClusterStatsResult{}
		}
	}()

	return stats
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
	members := []MemberStats{}

	jp.ArrayEach(gossipJson, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		ip := getString(jsonValue, "httpEndPointIp")
		port := getInt(jsonValue, "httpEndPointPort")

		members = append(members, MemberStats{
			MemberName: fmt.Sprintf("%s:%d", ip, port),
			IsAlive:    getBoolean(jsonValue, "isAlive"),
		})
	}, "members")

	return members
}

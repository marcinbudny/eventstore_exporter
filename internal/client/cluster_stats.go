package client

import (
	"context"
)

type gossipEnvelope struct {
	Members []MemberStats `json:"members"`
}

const (
	MemberStateLeader          string = "leader"
	MemberStateFollower        string = "follower"
	MemberStateReadOnlyReplica string = "readonlyreplica"
	MemberStateClone           string = "clone"
)

type MemberStats struct {
	HTTPEndpointIP   string `json:"httpEndPointIp"`
	HTTPEndpointPort int    `json:"httpEndPointPort"`
	IsAlive          bool
}

func (client *EventStoreStatsClient) getClusterStats(ctx context.Context) (stats []MemberStats, err error) {
	gossip, err := esHTTPGetAndParse[gossipEnvelope](ctx, client, "/gossip", false)
	if err != nil {
		return nil, err
	}

	return gossip.Members, nil
}

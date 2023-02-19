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
	HttpEndpointIp   string `json:"httpEndPointIp"`
	HttpEndpointPort int    `json:"httpEndPointPort"`
	IsAlive          bool
}

func (client *EventStoreStatsClient) getClusterStats(ctx context.Context) (stats []MemberStats, err error) {
	if gossip, err := esHttpGetAndParse[gossipEnvelope](ctx, client, "/gossip", false); err != nil {
		return nil, err
	} else {
		return gossip.Members, nil
	}
}

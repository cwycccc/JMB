package signer

import (
	"context"
	"sync"
	"time"
)

type NodeStatus struct {
	PartyID   int     `json:"party_id"`
	Status    string  `json:"status"`
	KeyLoaded bool    `json:"key_loaded"`
	LatencyMS float64 `json:"latency_ms"`
}
type ClusterStatus struct {
	Mode                     string       `json:"mode"`
	KeyID                    string       `json:"key_id"`
	Threshold                int          `json:"threshold"`
	Total                    int          `json:"total_parties"`
	Nodes                    []NodeStatus `json:"nodes"`
	RequiresReinitialization bool         `json:"requires_reinitialization"`
}
type Monitor interface {
	ClusterStatus(context.Context) ClusterStatus
}

func (s *DistributedSigner) ClusterStatus(ctx context.Context) ClusterStatus {
	result := ClusterStatus{Mode: "distributed", KeyID: s.config.KeyID(), Threshold: s.config.Threshold(), Total: s.config.TotalParties(), RequiresReinitialization: s.blocked.Load()}
	result.Nodes = make([]NodeStatus, result.Total)
	var wg sync.WaitGroup
	for id := range result.Nodes {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			start := time.Now()
			h, e := s.client.Health(ctx, id)
			n := NodeStatus{PartyID: id, Status: "UNAVAILABLE", LatencyMS: float64(time.Since(start)) / float64(time.Millisecond)}
			if e == nil {
				n.Status = "IDENTITY_MISMATCH"
				if h.PartyID == id && h.KeyID == result.KeyID && h.Threshold == result.Threshold && h.Total == result.Total && h.KeyLoaded && h.Status == "CRYPTO_READY" {
					n.Status = "CRYPTO_READY"
					n.KeyLoaded = true
				}
			}
			result.Nodes[id] = n
		}(id)
	}
	wg.Wait()
	return result
}

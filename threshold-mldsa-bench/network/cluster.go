package network

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"threshold-mldsa/cryptoengine"
)

// ClusterConfig is PUBLIC startup information. Credentials and private shares
// must never be added to this type or to the launcher's JSON summary.
type ClusterConfig struct {
	Version      uint16                        `json:"version"`
	Status       string                        `json:"status"`
	KeyID        string                        `json:"key_id"`
	PublicConfig cryptoengine.PublicDescriptor `json:"public_config"`
	Nodes        []ClusterNode                 `json:"nodes"`
}

type ClusterNode struct {
	PartyID int    `json:"party_id"`
	PID     int    `json:"pid,omitempty"`
	Port    int    `json:"port"`
	Address string `json:"address"`
}

func ReadClusterConfig(reader io.Reader) (ClusterConfig, error) {
	var config ClusterConfig
	data, err := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return config, errors.New("invalid public cluster configuration size")
	}
	// Windows PowerShell's Set-Content -Encoding utf8 may emit a BOM.
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&config) != nil || d.Decode(new(any)) != io.EOF {
		return config, errors.New("invalid public cluster configuration JSON")
	}
	_, _, err = config.Validate()
	return config, err
}

func (c ClusterConfig) Validate() (*cryptoengine.PublicConfig, map[int]string, error) {
	invalid := errors.New("invalid public cluster configuration")
	if c.Version != 1 || c.Status != "CRYPTO_READY" {
		return nil, nil, invalid
	}
	public, err := cryptoengine.ImportPublicConfig(c.PublicConfig, c.KeyID)
	if err != nil || len(c.Nodes) != public.TotalParties() {
		return nil, nil, invalid
	}
	endpoints := make(map[int]string, len(c.Nodes))
	for _, node := range c.Nodes {
		if node.PartyID < 0 || node.PartyID >= public.TotalParties() {
			return nil, nil, invalid
		}
		if _, exists := endpoints[node.PartyID]; exists {
			return nil, nil, invalid
		}
		if node.Port < 1 || node.Port > 65535 || node.Address != fmt.Sprintf("http://127.0.0.1:%d", node.Port) {
			return nil, nil, invalid
		}
		endpoints[node.PartyID] = node.Address
	}
	return public, endpoints, nil
}

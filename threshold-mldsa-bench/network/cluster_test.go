package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"threshold-mldsa/cryptoengine"
)

func TestPublicClusterRoundTripAndValidation(t *testing.T) {
	config, shares, err := cryptoengine.TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, share := range shares {
		share.Destroy()
	}
	cluster := ClusterConfig{Version: 1, Status: "CRYPTO_READY", KeyID: config.KeyID(), PublicConfig: config.PublicDescriptor()}
	for id := 0; id < 5; id++ {
		cluster.Nodes = append(cluster.Nodes, ClusterNode{PartyID: id, Port: 9200 + id, Address: fmt.Sprintf("http://127.0.0.1:%d", 9200+id)})
	}
	data, err := json.Marshal(cluster)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadClusterConfig(bytes.NewReader(append([]byte{0xef, 0xbb, 0xbf}, data...))); err != nil {
		t.Fatal("Windows UTF-8 public config did not round trip")
	}
	cluster.PublicConfig.PublicKey[0] ^= 1
	if _, _, err := cluster.Validate(); err == nil {
		t.Fatal("changed public key accepted")
	}
	cluster.PublicConfig = config.PublicDescriptor()
	cluster.Nodes[4].PartyID = 0
	if _, _, err := cluster.Validate(); err == nil {
		t.Fatal("duplicate node accepted")
	}
	var unknown map[string]any
	if json.Unmarshal(data, &unknown) != nil {
		t.Fatal("decode fixture")
	}
	unknown["coordinator_token"] = "must not be in public configuration"
	data, err = json.Marshal(unknown)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadClusterConfig(bytes.NewReader(data)); err == nil {
		t.Fatal("credential field accepted in public configuration")
	}
}

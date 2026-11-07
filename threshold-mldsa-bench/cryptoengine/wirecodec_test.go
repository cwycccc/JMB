package cryptoengine

import (
	"bytes"
	"encoding/json"
	"testing"
)

func wireFixture(t *testing.T) (*PublicConfig, WireContext) {
	t.Helper()
	config, shares, err := TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, share := range shares {
		share.Destroy()
	}
	return config, WireContext{Version: WireVersion, SessionID: "wire-test", Attempt: 4,
		KeyID: config.KeyID(), Participants: []int{0, 2, 4}, MessageHash: MessageHash([]byte("transaction"))}
}

func TestWireRoundTrips(t *testing.T) {
	config, ctx := wireFixture(t)
	for round := 1; round <= 3; round++ {
		size := 32
		if round == 2 {
			size = config.params.CommitmentSize()
		}
		if round == 3 {
			size = config.params.ResponseSize()
		}
		var messages []PublicRoundMessage
		for _, id := range ctx.Participants {
			messages = append(messages, PublicRoundMessage{ctx, id, round, bytes.Repeat([]byte{byte(id)}, size)})
		}
		data, err := EncodeRoundMessages(config, ctx, round, messages)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeRoundMessages(config, ctx, round, data)
		if err != nil {
			t.Fatal(err)
		}
		for i := range messages {
			if !bytes.Equal(messages[i].Payload, decoded[i]) {
				t.Fatal("round trip changed public bytes")
			}
		}
	}
}

func TestWireRejectsBindingOrderAndLengthChanges(t *testing.T) {
	config, ctx := wireFixture(t)
	mutations := []struct {
		name   string
		change func([]PublicRoundMessage)
	}{
		{"version", func(m []PublicRoundMessage) { m[0].Version++ }},
		{"session", func(m []PublicRoundMessage) { m[0].SessionID = "other" }},
		{"attempt", func(m []PublicRoundMessage) { m[0].Attempt++ }},
		{"key", func(m []PublicRoundMessage) { m[0].KeyID = "other" }},
		{"hash", func(m []PublicRoundMessage) { m[0].MessageHash = MessageHash([]byte("other")) }},
		{"sender", func(m []PublicRoundMessage) { m[0].PartyID = 1 }},
		{"order", func(m []PublicRoundMessage) { m[0], m[1] = m[1], m[0] }},
		{"duplicate", func(m []PublicRoundMessage) { m[1] = m[0] }},
		{"length", func(m []PublicRoundMessage) { m[0].Payload = []byte{0} }},
		{"round", func(m []PublicRoundMessage) { m[0].Round = 2 }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			var messages []PublicRoundMessage
			for _, id := range ctx.Participants {
				messages = append(messages, PublicRoundMessage{ctx, id, 1, make([]byte, 32)})
			}
			mutation.change(messages)
			data, err := json.Marshal(messages)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeRoundMessages(config, ctx, 1, data); err == nil {
				t.Fatal("modified wire batch accepted")
			}
		})
	}
}

func TestPublicConfigPinsKeyID(t *testing.T) {
	config, _ := wireFixture(t)
	d := config.PublicDescriptor()
	imported, err := ImportPublicConfig(d, config.KeyID())
	if err != nil || imported.KeyID() != config.KeyID() {
		t.Fatal("public config did not round trip")
	}
	d.PublicKey[0] ^= 1
	if _, err := ImportPublicConfig(d, config.KeyID()); err == nil {
		t.Fatal("changed public key accepted")
	}
}

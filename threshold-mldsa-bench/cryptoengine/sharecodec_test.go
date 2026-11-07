package cryptoengine

import (
	"bytes"
	"encoding/json"
	"testing"
)

func exportTestEnvelope(t *testing.T, id int) (*PublicConfig, []byte) {
	t.Helper()
	config, shares, err := TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, share := range shares {
			share.Destroy()
		}
	})
	data, err := ExportPartyShare(config, shares[id])
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { clear(data) })
	return config, data
}

func TestShareCodecRoundTripSigns(t *testing.T) {
	config, shares, err := TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, share := range shares {
			share.Destroy()
		}
	}()
	parties := make([]Party, 5)
	var importedConfig *PublicConfig
	for id := range shares {
		data, err := ExportPartyShare(config, shares[id])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := NewParty(config, shares[id]); err == nil {
			t.Fatal("exported handle was reusable")
		}
		if _, err := ExportPartyShare(config, shares[id]); err == nil {
			t.Fatal("share was exported twice")
		}
		importedConfig, parties[id], err = ImportPartyShareForKey(data, id, config.KeyID())
		clear(data)
		if err != nil {
			t.Fatalf("import party %d: %v", id, err)
		}
		if parties[id].ID() != id || importedConfig.KeyID() != config.KeyID() {
			t.Fatal("import identity changed")
		}
		if importedConfig.setup == config.setup {
			t.Fatal("import reused process-local setup pointer")
		}
	}
	coordinator, err := NewCoordinator(importedConfig)
	if err != nil {
		t.Fatal(err)
	}
	system := testSystem{parties: parties, coordinator: coordinator}
	for _, participants := range [][]int{{0, 1, 2}, {0, 2, 4}} {
		sessionID := "codec-012"
		if participants[1] == 2 {
			sessionID = "codec-024"
		}
		signature, err := system.sign([]byte(messageA), participants, sessionID)
		if err != nil {
			t.Fatal(err)
		}
		if !coordinator.VerifyMessage([]byte(messageA), signature) {
			t.Fatal("restored shares could not sign")
		}
		if coordinator.VerifyMessage([]byte(messageB), signature) {
			t.Fatal("modified message verified")
		}
	}
}

func TestShareCodecRejectsMismatchedSetup(t *testing.T) {
	configA, dataA := exportTestEnvelope(t, 0)
	configB, dataB := exportTestEnvelope(t, 0)
	if configA.KeyID() == configB.KeyID() {
		t.Fatal("fresh setup reused KeyID")
	}
	if _, _, err := ImportPartyShareForKey(dataB, 0, configA.KeyID()); err == nil {
		t.Fatal("another valid initialization batch passed the pinned KeyID")
	}
	var envelopeA, envelopeB ShareEnvelope
	if err := json.Unmarshal(dataA, &envelopeA); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(dataB, &envelopeB); err != nil {
		t.Fatal(err)
	}
	defer clear(envelopeA.SecretShare)
	defer clear(envelopeB.SecretShare)
	envelopeA.SecretShare = append(envelopeA.SecretShare[:0], envelopeB.SecretShare...)
	mixed, err := json.Marshal(envelopeA)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(mixed)
	if _, _, err := ImportPartyShare(mixed, 0); err == nil {
		t.Fatal("mixed secret share and public configuration were accepted")
	}
	if _, _, err := ImportPartyShare(dataA, 2); err == nil {
		t.Fatal("party 2 loaded party 0's share")
	}
}

func TestShareCodecRejectsWrongExportConfiguration(t *testing.T) {
	configA, sharesA, err := TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, share := range sharesA {
			share.Destroy()
		}
	}()
	configB, sharesB, err := TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, share := range sharesB {
			share.Destroy()
		}
	}()
	if _, err := ExportPartyShare(configB, sharesA[0]); err == nil {
		t.Fatal("export accepted a share from a different setup")
	}
	data, err := ExportPartyShare(configA, sharesA[0])
	if err != nil {
		t.Fatal("failed export consumed the correct share handle")
	}
	clear(data)
}

func TestShareCodecRejectsMalformedEnvelope(t *testing.T) {
	_, valid := exportTestEnvelope(t, 0)
	tests := []struct {
		name   string
		mutate func(*ShareEnvelope)
	}{
		{"version", func(e *ShareEnvelope) { e.Version++ }},
		{"threshold", func(e *ShareEnvelope) { e.Threshold = 1 }},
		{"total", func(e *ShareEnvelope) { e.Total = 300 }},
		{"key id", func(e *ShareEnvelope) { e.KeyID = "wrong" }},
		{"public key", func(e *ShareEnvelope) { e.PublicKey[50] ^= 1 }},
		{"short public key", func(e *ShareEnvelope) { e.PublicKey = e.PublicKey[:10] }},
		{"short private share", func(e *ShareEnvelope) { e.SecretShare = e.SecretShare[:10] }},
		{"extra private data", func(e *ShareEnvelope) { e.SecretShare = append(e.SecretShare, 0) }},
		{"inner party id", func(e *ShareEnvelope) { e.SecretShare[0] = 2 }},
		{"rho", func(e *ShareEnvelope) { e.SecretShare[1] ^= 1 }},
		{"public key hash", func(e *ShareEnvelope) { e.SecretShare[65] ^= 1 }},
		{"coefficient", func(e *ShareEnvelope) { e.SecretShare[thresholdShareHeaderSize+1] |= 7 }},
		{"duplicate share index", func(e *ShareEnvelope) {
			e.SecretShare[thresholdShareHeaderSize+thresholdShareRecordSize] = e.SecretShare[thresholdShareHeaderSize]
		}},
		{"other party share index", func(e *ShareEnvelope) { e.SecretShare[thresholdShareHeaderSize] = 14 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var envelope ShareEnvelope
			if err := json.Unmarshal(valid, &envelope); err != nil {
				t.Fatal(err)
			}
			defer clear(envelope.SecretShare)
			test.mutate(&envelope)
			// Rebind mutations of the inner layout to exercise the independent
			// range/mask/header validator, rather than only the commitment check.
			if test.name == "coefficient" || test.name == "duplicate share index" || test.name == "other party share index" {
				envelope.ShareCommitments[0] = shareCommitment(0, envelope.SecretShare)
				envelope.KeyID = configurationKeyID(envelope.Threshold, envelope.Total, envelope.PublicKey, envelope.ShareCommitments)
			}
			data, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			defer clear(data)
			if _, _, err := ImportPartyShare(data, 0); err == nil {
				t.Fatal("malformed envelope accepted")
			}
		})
	}
	for _, data := range [][]byte{nil, []byte("{}"), append(append([]byte(nil), valid...), []byte(" {}")...), bytes.Repeat([]byte{'x'}, MaxShareEnvelopeSize+1)} {
		if _, _, err := ImportPartyShare(data, 0); err == nil {
			t.Fatal("invalid envelope framing accepted")
		}
		clear(data)
	}
}

func TestShareCodecAllSupportedThresholds(t *testing.T) {
	for total := 2; total <= 6; total++ {
		for threshold := 2; threshold <= total; threshold++ {
			config, shares, err := TrustedKeySetup(threshold, total)
			if err != nil {
				t.Fatal(err)
			}
			for id, share := range shares {
				data, err := ExportPartyShare(config, share)
				if err != nil {
					t.Fatal(err)
				}
				_, _, err = ImportPartyShareForKey(data, id, config.KeyID())
				clear(data)
				if err != nil {
					t.Fatalf("%d-of-%d party %d: %v", threshold, total, id, err)
				}
			}
		}
	}
}

func FuzzImportPartyShare(f *testing.F) {
	f.Add([]byte("{}"), 0)
	f.Add([]byte(`{"version":1,"party_id":0,"total":5}`), 2)
	f.Fuzz(func(t *testing.T, data []byte, id int) {
		_, _, _ = ImportPartyShare(data, id)
	})
}

package signernode

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
)

// Initialization wraps the unchanged C2.2 envelope and the transport credential
// exclusively on the private stdin pipe. It is never an HTTP protocol message.
type Initialization struct {
	Version          uint16          `json:"version"`
	Share            json.RawMessage `json:"share"`
	CoordinatorToken string          `json:"coordinator_token"`
}

func EncodeInitialization(share []byte, token string) ([]byte, error) {
	if len(share) > cryptoengine.MaxShareEnvelopeSize || network.ValidateToken(token) != nil {
		return nil, errors.New("invalid initialization")
	}
	return json.Marshal(Initialization{Version: 1, Share: share, CoordinatorToken: token})
}

func ReadInitialization(reader io.Reader, partyID int, keyID string) (*cryptoengine.PublicConfig, cryptoengine.Party, string, error) {
	const limit = cryptoengine.MaxShareEnvelopeSize + 1024
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	defer clear(data)
	if err != nil || len(data) > limit {
		return nil, nil, "", errors.New("invalid initialization pipe size")
	}
	var init Initialization
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&init) != nil || decoder.Decode(new(any)) != io.EOF || init.Version != 1 || network.ValidateToken(init.CoordinatorToken) != nil {
		clear(init.Share)
		return nil, nil, "", errors.New("invalid pipe initialization")
	}
	defer clear(init.Share)
	config, party, err := cryptoengine.ImportPartyShareForKey(init.Share, partyID, keyID)
	if err != nil {
		return nil, nil, "", errors.New("share initialization rejected")
	}
	return config, party, init.CoordinatorToken, nil
}

// NewServer rejects all non-loopback bind addresses.
func NewServer(address string, handler http.Handler) (*http.Server, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, errors.New("node address must be a numeric loopback address")
	}
	return &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second,
		MaxHeaderBytes: 16 << 10}, nil
}

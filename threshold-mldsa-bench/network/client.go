package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"threshold-mldsa/cryptoengine"
)

type HTTPError struct{ Status int }

func (e *HTTPError) Error() string { return fmt.Sprintf("node HTTP status %d", e.Status) }

type Client struct {
	endpoints map[int]string
	token     string
	http      *http.Client
}

func NewClient(endpoints map[int]string, token string, timeout time.Duration) (*Client, error) {
	if ValidateToken(token) != nil || timeout <= 0 {
		return nil, errors.New("invalid coordinator client configuration")
	}
	copyEndpoints := make(map[int]string, len(endpoints))
	for id, endpoint := range endpoints {
		u, err := url.Parse(endpoint)
		if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return nil, errors.New("endpoint must be a plain localhost HTTP origin")
		}
		host, _, err := net.SplitHostPort(u.Host)
		ip := net.ParseIP(host)
		if err != nil || ip == nil || !ip.IsLoopback() || id < 0 || id >= 6 {
			return nil, errors.New("endpoint must use a numeric loopback address and valid Party ID")
		}
		copyEndpoints[id] = strings.TrimRight(endpoint, "/")
	}
	return &Client{endpoints: copyEndpoints, token: token, http: &http.Client{
		Timeout: timeout,
		// New connections and non-replayable POST bodies prohibit transparent
		// transport retries of cryptographic rounds. Redirects are rejected.
		Transport:     &http.Transport{Proxy: nil, DisableKeepAlives: true, DialContext: dialMeasured},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) call(ctx context.Context, id int, method, path string, body, result any) error {
	endpoint, exists := c.endpoints[id]
	if !exists {
		return errors.New("missing Party endpoint")
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil || len(data) > MaxHTTPBodySize {
			return errors.New("invalid protocol request body")
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint+path, reader)
	if err != nil {
		return errors.New("invalid HTTP request")
	}
	req.GetBody = nil
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("party %d request outcome uncertain: %w", id, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &HTTPError{Status: response.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxHTTPBodySize+1))
	if err != nil || len(data) > MaxHTTPBodySize {
		return errors.New("node response size or read error")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(result) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("invalid node response")
	}
	return nil
}

func (c *Client) Health(ctx context.Context, id int) (Health, error) {
	var result Health
	err := c.call(ctx, id, http.MethodGet, "/health", nil, &result)
	return result, err
}

func (c *Client) PublicConfig(ctx context.Context, id int, expectedKeyID string) (*cryptoengine.PublicConfig, error) {
	var descriptor cryptoengine.PublicDescriptor
	if err := c.call(ctx, id, http.MethodGet, "/internal/config", nil, &descriptor); err != nil {
		return nil, err
	}
	return cryptoengine.ImportPublicConfig(descriptor, expectedKeyID)
}

func (c *Client) Round(ctx context.Context, id, round int, req RoundRequest) (RoundResponse, error) {
	var result RoundResponse
	if round < 1 || round > 3 {
		return result, errors.New("invalid round")
	}
	if meter, ok := ctx.Value(trafficKey{}).(*TrafficMeter); ok {
		ctx = context.WithValue(ctx, connectionKey{}, connectionMetric{meter: meter, round: round})
	}
	err := c.call(ctx, id, http.MethodPost, fmt.Sprintf("/internal/round%d", round), req, &result)
	if err != nil {
		return result, err
	}
	if !result.WireContext.Equal(req.WireContext) || result.PartyID != id || result.Round != round {
		return result, errors.New("node response binding mismatch")
	}
	if meter, ok := ctx.Value(trafficKey{}).(*TrafficMeter); ok {
		meter.add(round, 0, 0, int64(len(result.Payload)))
	}
	return result, nil
}

func (c *Client) State(ctx context.Context, id int, req RoundRequest) (StateResponse, error) {
	return c.state(ctx, id, "status", req)
}

func (c *Client) Abort(ctx context.Context, id int, req RoundRequest) (StateResponse, error) {
	return c.state(ctx, id, "abort", req)
}

func (c *Client) state(ctx context.Context, id int, operation string, req RoundRequest) (StateResponse, error) {
	var result StateResponse
	req.Payload, req.Message = nil, nil
	if err := c.call(ctx, id, http.MethodPost, "/internal/"+operation, req, &result); err != nil {
		return result, err
	}
	if !result.WireContext.Equal(req.WireContext) || result.PartyID != id {
		return result, errors.New("state response binding mismatch")
	}
	switch result.State {
	case "UNKNOWN", "ROUND1_DONE", "ROUND2_DONE", "COMPLETED", "FAILED", "ABORTED":
	default:
		return result, errors.New("invalid session state")
	}
	if operation == "abort" && result.State != "ABORTED" {
		return result, errors.New("abort not confirmed")
	}
	return result, nil
}

package network

import (
	"context"
	"net"
	"sync"
)

// Traffic counts actual HTTP/1.1 stream bytes on new per-request connections.
// It excludes TCP/IP/link headers, health/config/status/abort requests and the
// browser-to-backend connection. No header or payload contents are retained.
type Traffic struct {
	Rounds [3]RoundTraffic `json:"rounds"`
}
type RoundTraffic struct {
	Sent              int64 `json:"http_sent_bytes"`
	Received          int64 `json:"http_received_bytes"`
	ProtocolResponses int64 `json:"protocol_response_bytes"`
}
type TrafficMeter struct {
	mu    sync.Mutex
	value Traffic
}
type trafficKey struct{}
type connectionKey struct{}
type connectionMetric struct {
	meter *TrafficMeter
	round int
}

func MeasureTraffic(ctx context.Context) (context.Context, *TrafficMeter) {
	m := &TrafficMeter{}
	return context.WithValue(ctx, trafficKey{}, m), m
}
func (m *TrafficMeter) Snapshot() Traffic { m.mu.Lock(); defer m.mu.Unlock(); return m.value }
func (m *TrafficMeter) add(round int, sent, received, protocol int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := &m.value.Rounds[round-1]
	r.Sent += sent
	r.Received += received
	r.ProtocolResponses += protocol
}

type measuredConn struct {
	net.Conn
	metric connectionMetric
}

func (c *measuredConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	c.metric.meter.add(c.metric.round, 0, int64(n), 0)
	return n, e
}
func (c *measuredConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	c.metric.meter.add(c.metric.round, int64(n), 0, 0)
	return n, e
}
func dialMeasured(ctx context.Context, network, address string) (net.Conn, error) {
	c, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	if m, ok := ctx.Value(connectionKey{}).(connectionMetric); ok {
		return &measuredConn{Conn: c, metric: m}, nil
	}
	return c, nil
}

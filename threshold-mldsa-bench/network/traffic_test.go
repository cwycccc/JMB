package network

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"threshold-mldsa/cryptoengine"
)

// A raw HTTP peer gives exact, independent stream byte counts, including
// headers. Counting a JSON marshal or estimating Base64 overhead would fail it.
func TestTrafficMatchesHTTPStream(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	wire := cryptoengine.WireContext{Version: 1, SessionID: "test", KeyID: "key", Participants: []int{0, 2, 4}, MessageHash: cryptoengine.MessageHash(nil)}
	req := RoundRequest{WireContext: wire}
	response, _ := json.Marshal(RoundResponse{WireContext: wire, PartyID: 0, Round: 1, Payload: make([]byte, 32)})
	type count struct {
		sent, received int64
		err            error
	}
	done := make(chan count, 1)
	go func() {
		c, e := l.Accept()
		if e != nil {
			done <- count{err: e}
			return
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		reader := bufio.NewReader(c)
		var received int64
		var head strings.Builder
		for {
			line, e := reader.ReadString('\n')
			if e != nil {
				done <- count{err: e}
				return
			}
			received += int64(len(line))
			head.WriteString(line)
			if line == "\r\n" {
				break
			}
		}
		r, e := http.ReadRequest(bufio.NewReader(strings.NewReader(head.String())))
		if e != nil {
			done <- count{err: e}
			return
		}
		n, e := io.CopyN(io.Discard, reader, r.ContentLength)
		received += n
		if e != nil {
			done <- count{err: e}
			return
		}
		message := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(response), response)
		nn, e := io.WriteString(c, message)
		done <- count{sent: int64(nn), received: received, err: e}
	}()
	client, e := NewClient(map[int]string{0: "http://" + l.Addr().String()}, strings.Repeat("a", 64), 5*time.Second)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	ctx, meter := MeasureTraffic(context.Background())
	if _, e := client.Round(ctx, 0, 1, req); e != nil {
		t.Fatal(e)
	}
	expected := <-done
	if expected.err != nil {
		t.Fatal(expected.err)
	}
	r := meter.Snapshot().Rounds[0]
	if r.Sent != expected.received || r.Received != expected.sent || r.ProtocolResponses != 32 {
		t.Fatalf("got %+v, expected request=%d response=%d payload=32", r, expected.received, expected.sent)
	}
	if r.Sent <= int64(len(response)) {
		t.Fatal("headers not counted")
	}
	if meter.Snapshot().Rounds[1] != (RoundTraffic{}) {
		t.Fatal("round isolation failed")
	}
}

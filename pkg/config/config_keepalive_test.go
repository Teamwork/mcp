package config

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// keepaliveTestInterval is short enough to keep the tests quick, and long
// enough that the first tick lands well after an in-process round trip.
const keepaliveTestInterval = 200 * time.Millisecond

// newKeepaliveTestServer starts a server carrying keepalivePingGate and a fast
// keepalive, and returns the raw client end of the connection. The client side
// is hand-rolled rather than an mcp.Client because the SDK client picks its own
// protocol version, and these tests need to pin both sides of the boundary.
func newKeepaliveTestServer(t *testing.T) mcp.Connection {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, &mcp.ServerOptions{
		KeepAlive:                 keepaliveTestInterval,
		KeepAliveFailureThreshold: keepaliveFailureThreshold,
	})
	server.AddSendingMiddleware(keepalivePingGate())

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	session, err := server.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	conn, err := clientTransport.Connect(t.Context())
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func writeMessage(t *testing.T, conn mcp.Connection, msg jsonrpc.Message) {
	t.Helper()
	if err := conn.Write(t.Context(), msg); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}
}

func jsonrpcRequest(t *testing.T, id int64, method string, params any) *jsonrpc.Request {
	t.Helper()
	encoded, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to encode params: %v", err)
	}
	jsonrpcID, err := jsonrpc.MakeID(float64(id))
	if err != nil {
		t.Fatalf("failed to build request id: %v", err)
	}
	return &jsonrpc.Request{ID: jsonrpcID, Method: method, Params: encoded}
}

// readUntilResponse reads messages until the response to id arrives, reporting
// whether a ping was seen on the way. The keepalive is allowed to ping before a
// version is known, so a ping ahead of the first response says nothing about the
// gate; only what follows does.
func readUntilResponse(t *testing.T, conn mcp.Connection, id int64) (pinged bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	for {
		msg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("failed to read message: %v", err)
		}
		switch msg := msg.(type) {
		case *jsonrpc.Request:
			if msg.Method == "ping" {
				pinged = true
			}
		case *jsonrpc.Response:
			if wanted, err := jsonrpc.MakeID(float64(id)); err == nil && msg.ID == wanted {
				if msg.Error != nil {
					t.Fatalf("request %d failed: %v", id, msg.Error)
				}
				return pinged
			}
		}
	}
}

// TestKeepalivePingGateLegacySessionKeepsThePing pins the untouched half: a
// session on a protocol version that still has "ping" must keep receiving it,
// otherwise the gate has traded one broken keepalive for another.
func TestKeepalivePingGateLegacySessionKeepsThePing(t *testing.T) {
	conn := newKeepaliveTestServer(t)

	writeMessage(t, conn, jsonrpcRequest(t, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "probe", "version": "0"},
	}))
	readUntilResponse(t, conn, 1)
	writeMessage(t, conn, &jsonrpc.Request{Method: "notifications/initialized", Params: json.RawMessage(`{}`)})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for {
		msg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("expected a keepalive ping, got: %v", err)
		}
		if req, ok := msg.(*jsonrpc.Request); ok && req.Method == "ping" {
			return
		}
	}
}

// TestKeepalivePingGateModernSessionGetsNoPing is the regression: "ping" was
// removed in 2026-07-28, so a session on that version must never see one. The
// keepalive is version-blind, so without the gate a ping goes out on the first
// tick and the peer has to reject it.
func TestKeepalivePingGateModernSessionGetsNoPing(t *testing.T) {
	conn := newKeepaliveTestServer(t)

	// A sessionless request carries the version in _meta instead of a handshake,
	// which is the only way to reach 2026-07-28: "initialize" is deprecated there,
	// so the SDK caps that path at 2025-11-25.
	modernParams := func() map[string]any {
		return map[string]any{
			"_meta": map[string]any{
				"io.modelcontextprotocol/protocolVersion":    protocolVersionWithoutPing,
				"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "probe", "version": "0"},
				"io.modelcontextprotocol/clientCapabilities": map[string]any{},
			},
		}
	}

	writeMessage(t, conn, jsonrpcRequest(t, 1, "tools/list", modernParams()))
	readUntilResponse(t, conn, 1)

	// Give the keepalive several ticks to misfire, then confirm the session is
	// still usable: the gate must stop the ping, not the connection.
	time.Sleep(3 * keepaliveTestInterval)
	writeMessage(t, conn, jsonrpcRequest(t, 2, "tools/list", modernParams()))
	if pinged := readUntilResponse(t, conn, 2); pinged {
		t.Error("server sent a keepalive ping on a session whose protocol version removed it")
	}
}

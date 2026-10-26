package agentcore

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labtether/labtether-linux/internal/agentcore/files"
	"github.com/labtether/labtether-linux/pkg/agentidentity"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

func TestPendingEnrollmentKeepsOnlyControlChannelActive(t *testing.T) {
	t.Setenv(envAllowInsecureTransport, "true")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity := &deviceIdentity{
		KeyAlgorithm: agentidentity.KeyAlgorithmEd25519, PublicKey: publicKey, PrivateKey: privateKey,
		PublicKeyBase64: base64.StdEncoding.EncodeToString(publicKey),
		Fingerprint:     agentidentity.FingerprintFromPublicKey(publicKey),
	}
	connections := make(chan *websocket.Conn, 2)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, upgradeErr := upgrader.Upgrade(w, r, nil)
		if upgradeErr != nil {
			t.Error(upgradeErr)
			return
		}
		connections <- conn
	}))
	defer server.Close()
	transport := newWSTransport("ws"+strings.TrimPrefix(server.URL, "http"), "", "pending-node", "linux", "test", nil, "", identity)
	defer transport.Close()
	if err := transport.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	serverConn := <-connections
	defer serverConn.Close()
	if transport.Connected() || !transport.EnrollmentPending() {
		t.Fatal("pending enrollment was advertised as ready")
	}
	if state, _, _ := transport.ConnectionState(); state != "enrollment_pending" {
		t.Fatalf("state=%q", state)
	}
	publisher := newWSHeartbeatPublisher(transport, pendingEnrollmentFallback{t}, RuntimeConfig{}, nil, nil)
	if err := publisher.Publish(context.Background(), TelemetrySample{}); err != nil {
		t.Fatal(err)
	}
	if err := transport.Send(agentmgr.Message{Type: agentmgr.MsgHeartbeat}); !errors.Is(err, errEnrollmentPending) {
		t.Fatalf("pending heartbeat send error=%v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	fileRoot := t.TempDir()
	go func() {
		defer close(done)
		receiveLoop(ctx, transport, RuntimeConfig{}, nil,
			newTerminalManager(), newDesktopManager(nil), nil, &files.Manager{BaseDir: fileRoot},
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}()
	defer func() {
		cancel()
		transport.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("receive loop did not stop")
		}
	}()
	// If an operational ping is wrongly dispatched, its pong precedes the
	// enrollment proof and the real Hub rejects the pending socket.
	if err := serverConn.WriteJSON(agentmgr.Message{Type: agentmgr.MsgPing}); err != nil {
		t.Fatal(err)
	}
	challenge, err := json.Marshal(agentmgr.EnrollmentChallengeData{ConnectionID: "pending-connection", Nonce: "nonce"})
	if err != nil {
		t.Fatal(err)
	}
	if err := serverConn.WriteJSON(agentmgr.Message{Type: agentmgr.MsgEnrollmentChallenge, Data: challenge}); err != nil {
		t.Fatal(err)
	}
	_ = serverConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var proof agentmgr.Message
	if err := serverConn.ReadJSON(&proof); err != nil {
		t.Fatal(err)
	}
	if proof.Type != agentmgr.MsgEnrollmentProof {
		t.Fatalf("pending channel emitted %q before proof", proof.Type)
	}
	var payload agentmgr.EnrollmentProofData
	if err := json.Unmarshal(proof.Data, &payload); err != nil {
		t.Fatal(err)
	}
	signature, err := base64.StdEncoding.DecodeString(payload.Signature)
	if err != nil || !ed25519.Verify(publicKey, agentidentity.BuildEnrollmentProofPayload("pending-connection", "nonce", identity.Fingerprint), signature) {
		t.Fatal("enrollment proof signature did not verify")
	}
}

type pendingEnrollmentFallback struct{ t *testing.T }

func (p pendingEnrollmentFallback) Publish(context.Context, TelemetrySample) error {
	p.t.Error("pending enrollment attempted unauthenticated HTTP fallback")
	return nil
}

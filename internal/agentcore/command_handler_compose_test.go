package agentcore

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/labtether/labtether-linux/internal/agentcore/docker"
	"github.com/labtether/labtether-linux/internal/agentcore/files"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReceiveLoopDispatchesComposeWireVersions(t *testing.T) {
	collector := docker.NewDockerCollector("https://docker.example:2376?unsupported=1", nil, "host", time.Minute)
	_, server, stop := startReceiveLoopForComposeTest(t, collector)
	defer stop()
	for _, test := range []struct{ name, kind, want string }{
		{"v2", agentmgr.MsgDockerComposeActionV2, "Compose endpoint changed or missing; refresh and resubmit the action"},
		{"legacy", agentmgr.MsgDockerComposeAction, "Docker endpoint cannot be represented by Compose"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := json.Marshal(agentmgr.DockerComposeActionData{RequestID: test.name, StackName: "project", Action: "restart"})
			if err != nil {
				t.Fatal(err)
			}
			if err := server.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
				t.Fatal(err)
			}
			if err := server.WriteJSON(agentmgr.Message{Type: test.kind, Data: request}); err != nil {
				t.Fatal(err)
			}
			if err := server.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
				t.Fatal(err)
			}
			var message agentmgr.Message
			if err := server.ReadJSON(&message); err != nil {
				t.Fatal(err)
			}
			var result agentmgr.DockerComposeResultData
			if message.Type != agentmgr.MsgDockerComposeResult || message.ID != test.name || json.Unmarshal(message.Data, &result) != nil || result.RequestID != test.name || result.Success || result.Output != "" || result.Error != test.want {
				t.Fatalf("Compose receiver contract: type=%q id=%q result=%+v", message.Type, message.ID, result)
			}
		})
	}
}

func startReceiveLoopForComposeTest(t *testing.T, collector *docker.DockerCollector) (*wsTransport, *websocket.Conn, func()) {
	t.Helper()
	sockets := make(chan *websocket.Conn, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		sockets <- socket
	}))
	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	var socket *websocket.Conn
	select {
	case socket = <-sockets:
	case <-time.After(2 * time.Second):
		client.Close()
		server.Close()
		t.Fatal("missing fixture websocket")
	}
	transport := &wsTransport{token: "authenticated-test-token", conn: client, connected: true}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	fileRoot := t.TempDir()
	go func() {
		defer close(done)
		receiveLoop(ctx, transport, RuntimeConfig{}, nil,
			newTerminalManager(), newDesktopManager(nil), nil, &files.Manager{BaseDir: fileRoot},
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, collector, nil, nil, nil)
	}()
	return transport, socket, func() {
		cancel()
		transport.Close()
		socket.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Compose receiver did not stop")
		}
		server.Close()
	}
}

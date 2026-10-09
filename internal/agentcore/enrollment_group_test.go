package agentcore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

func TestEnrollmentUsesAndRestoresHubCanonicalGroup(t *testing.T) {
	cases := []struct {
		name          string
		responseGroup *string
		wantEnrolled  string
		wantRestart   string
		wantKnown     bool
	}{
		{name: "forced group", responseGroup: groupPointer("server-group"), wantEnrolled: "server-group", wantRestart: "server-group", wantKnown: true},
		{name: "explicit unplaced", responseGroup: groupPointer(""), wantEnrolled: "", wantRestart: "", wantKnown: true},
		{name: "older hub omitted group", wantEnrolled: "requested-group", wantRestart: "restart-group"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envAllowInsecureTransport, "true")
			t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request enrollRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if request.GroupID != "requested-group" {
					t.Errorf("requested group=%q", request.GroupID)
				}
				response := map[string]any{"agent_token": "issued-agent-token", "asset_id": "group-agent", "hub_ws_url": "ws://" + r.Host + "/ws/agent", "hub_api_url": "http://" + r.Host}
				if tc.responseGroup != nil {
					response["group_id"] = *tc.responseGroup
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			tokenFile := filepath.Join(t.TempDir(), "agent-token")
			cfg := RuntimeConfig{AssetID: "group-agent", GroupID: "requested-group", APIBaseURL: server.URL, EnrollmentToken: "one-use-token", TokenFilePath: tokenFile}
			if err := ResolveToken(context.Background(), &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.GroupID != tc.wantEnrolled || cfg.groupIDCanonical != tc.wantKnown {
				t.Fatalf("enrolled group=%q known=%t", cfg.GroupID, cfg.groupIDCanonical)
			}
			var state map[string]any
			raw, err := os.ReadFile(enrollmentStatePath(tokenFile))
			if err != nil {
				t.Fatalf("read enrollment state: %v", err)
			}
			if err := json.Unmarshal(raw, &state); err != nil {
				t.Fatalf("decode enrollment state: %v", err)
			}
			_, present := state["group_id"]
			if present != tc.wantKnown {
				t.Fatalf("persisted group presence=%t, want %t", present, tc.wantKnown)
			}
			restarted := RuntimeConfig{AssetID: "local-hostname", GroupID: "restart-group", APIToken: "issued-agent-token", APITokenFromFile: true, TokenFilePath: tokenFile}
			if err := ResolveToken(context.Background(), &restarted); err != nil {
				t.Fatal(err)
			}
			if restarted.GroupID != tc.wantRestart || restarted.groupIDCanonical != tc.wantKnown {
				t.Fatalf("restored group=%q known=%t", restarted.GroupID, restarted.groupIDCanonical)
			}
		})
	}
}

func TestRotatedGroupUpdatesWSHeartbeatAndLocalStatus(t *testing.T) {
	t.Setenv(envAllowInsecureTransport, "true")
	transport, messages, cleanup := newAgentcoreCapturedTransport(t)
	defer cleanup()
	transport.url = "ws://hub.example/ws/agent"
	transport.setInitialGroup("stale-group", false)
	canonical := "server-group"
	if _, err := transport.adoptCredential("rotated-token", "canonical-asset", transport.url, "http://hub.example", &canonical); err != nil {
		t.Fatal(err)
	}
	transport.connected = true
	publisher := newWSHeartbeatPublisher(transport, nil, RuntimeConfig{GroupID: "stale-group", Source: "agent"}, nil, nil)
	if err := publisher.Publish(context.Background(), TelemetrySample{AssetID: "local-hostname"}); err != nil {
		t.Fatal(err)
	}
	msg := waitForCapturedAgentMessage(t, messages, agentmgr.MsgHeartbeat, 2*time.Second)
	var heartbeat agentmgr.HeartbeatData
	if err := json.Unmarshal(msg.Data, &heartbeat); err != nil || heartbeat.AssetID != "canonical-asset" || heartbeat.GroupID != canonical {
		t.Fatalf("heartbeat identity: asset=%q group=%q err=%v", heartbeat.AssetID, heartbeat.GroupID, err)
	}
	runtime := NewRuntime(RuntimeConfig{AssetID: "local-hostname", GroupID: "stale-group"}, nil, publisher)
	runtime.transport = transport
	response := httptest.NewRecorder()
	runtime.statusHandler()(response, httptest.NewRequest(http.MethodGet, "/agent/status", nil))
	var local StatusResponse
	if err := json.Unmarshal(response.Body.Bytes(), &local); err != nil || local.AssetID != "canonical-asset" || local.GroupID != canonical {
		t.Fatalf("local status identity: asset=%q group=%q err=%v", local.AssetID, local.GroupID, err)
	}
	unplaced := ""
	if _, err := transport.adoptCredential("next-token", "canonical-asset", transport.url, "http://hub.example", &unplaced); err != nil {
		t.Fatal(err)
	}
	if got := transport.identitySnapshot(); !got.groupKnown || strings.TrimSpace(got.groupID) != "" {
		t.Fatalf("explicit unplaced group not adopted: group=%q known=%t", got.groupID, got.groupKnown)
	}
}

func groupPointer(group string) *string { return &group }

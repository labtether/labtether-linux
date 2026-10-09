package agentcore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"github.com/labtether/labtether-linux/pkg/assets"
)

func TestPendingApprovalActivatesCanonicalIdentityAndHTTPFallback(t *testing.T) {
	t.Setenv(envAllowInsecureTransport, "true")
	t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")
	requests := make(chan assets.HeartbeatRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets/heartbeat" || r.Header.Get("Authorization") != "Bearer approved-token" {
			t.Errorf("unexpected approved heartbeat: path=%q auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body assets.HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode heartbeat: %v", err)
		}
		requests <- body
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws/agent"
	tokenFile := filepath.Join(t.TempDir(), "agent-token")
	transport := newWSTransport(wsURL, "", "local-hostname", "linux", "test", nil, tokenFile, nil)
	transport.setInitialGroup("requested-group", false)
	publisher := newHeartbeatPublisher(RuntimeConfig{GroupID: "requested-group"}, nil, transport.identitySnapshot)
	if err := publisher.Publish(context.Background(), TelemetrySample{AssetID: "local-hostname"}); err == nil {
		t.Fatal("tokenless pending agent sent an HTTP heartbeat")
	}

	runtime := NewRuntime(RuntimeConfig{AssetID: "local-hostname", GroupID: "requested-group"},
		stubProvider{sample: TelemetrySample{AssetID: "local-hostname"}}, publisher)
	runtime.transport = transport
	approved, err := json.Marshal(agentmgr.EnrollmentApprovedData{Token: "approved-token", AssetID: "canonical-asset"})
	if err != nil {
		t.Fatal(err)
	}
	handleEnrollmentApproved(transport, agentmgr.Message{Data: approved}, RuntimeConfig{TokenFilePath: tokenFile})
	if got := transport.identitySnapshot(); got.assetID != "canonical-asset" || got.token != "approved-token" || got.apiBaseURL != server.URL {
		t.Fatalf("approved identity not active: asset=%q api=%q token-present=%v", got.assetID, got.apiBaseURL, got.token != "")
	}
	runtime.collectOnce(time.Now())
	if got := runtime.current().AssetID; got != "canonical-asset" {
		t.Fatalf("telemetry asset=%q, want canonical-asset", got)
	}
	if err := publisher.Publish(context.Background(), runtime.current()); err != nil {
		t.Fatalf("HTTP fallback after approval: %v", err)
	}
	select {
	case body := <-requests:
		if body.AssetID != "canonical-asset" || body.Name != "canonical-asset" || body.GroupID != "" {
			t.Fatalf("heartbeat used stale asset: %+v", body)
		}
	default:
		t.Fatal("approved heartbeat was not sent")
	}
	stateCfg := RuntimeConfig{TokenFilePath: tokenFile}
	stateCfg.GroupID = "stale-group"
	if err := restoreEnrollmentState(&stateCfg); err != nil || stateCfg.AssetID != "canonical-asset" || stateCfg.APIBaseURL != server.URL || stateCfg.GroupID != "" {
		t.Fatalf("approved identity not persisted: asset=%q api=%q err=%v", stateCfg.AssetID, stateCfg.APIBaseURL, err)
	}
	status := httptest.NewRecorder()
	runtime.statusHandler()(status, httptest.NewRequest(http.MethodGet, "/agent/status", nil))
	var local StatusResponse
	if err := json.Unmarshal(status.Body.Bytes(), &local); err != nil || local.GroupID != "" || local.AssetID != "canonical-asset" {
		t.Fatalf("approved local status identity: asset=%q group=%q err=%v", local.AssetID, local.GroupID, err)
	}
	for _, path := range []string{tokenFile, enrollmentStatePath(tokenFile)} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("private enrollment file %s: info=%v err=%v", path, info, err)
		}
	}
}

func TestRotatedCredentialUpdatesHTTPFallbackWithoutRestart(t *testing.T) {
	t.Setenv(envAllowInsecureTransport, "true")
	t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")
	var seen atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Add(1)
		if r.Header.Get("Authorization") != "Bearer rotated-token" {
			t.Errorf("fallback sent stale credential: %q", r.Header.Get("Authorization"))
		}
		var body assets.HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode heartbeat: %v", err)
		}
		if body.AssetID != "canonical-asset" {
			t.Errorf("fallback asset=%q", body.AssetID)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	transport := newWSTransport("ws://old-hub.invalid/ws/agent", "revoked-token", "old-host", "linux", "test", nil, "", nil)
	transport.apiBaseURL = "http://old-hub.invalid"
	publisher := newHeartbeatPublisher(RuntimeConfig{}, nil, transport.identitySnapshot)
	_, err := transport.adoptCredential("rotated-token", "canonical-asset", strings.Replace(server.URL, "http://", "ws://", 1)+"/ws/agent", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(context.Background(), TelemetrySample{AssetID: "old-host"}); err != nil {
		t.Fatalf("fallback after rotation: %v", err)
	}
	if got := seen.Load(); got != 1 {
		t.Fatalf("fallback request count=%d, want 1", got)
	}
}

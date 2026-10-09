package agentcore

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labtether/labtether-linux/pkg/assets"
	"github.com/labtether/labtether-linux/pkg/platforms"
)

func TestHeartbeatPublisherTrustsConfiguredCA(t *testing.T) {
	t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets/heartbeat" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected heartbeat request: path=%q auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	caPath := filepath.Join(t.TempDir(), "hub-ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(caPath, caPEM, 0600); err != nil {
		t.Fatal(err)
	}
	publisher := NewHeartbeatPublisher(RuntimeConfig{
		APIBaseURL: server.URL, APIToken: "test-token", TLSCAFile: caPath,
	}, nil)
	if err := publisher.Publish(context.Background(), TelemetrySample{AssetID: "qa-node"}); err != nil {
		t.Fatalf("HTTPS heartbeat with configured CA: %v", err)
	}
}

func TestResolveHeartbeatPlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		metadata map[string]string
		want     string
	}{
		{
			name:     "explicit platform",
			metadata: map[string]string{"platform": "windows"},
			want:     platforms.Windows,
		},
		{
			name:     "fallback to os",
			metadata: map[string]string{"os": "darwin"},
			want:     platforms.Darwin,
		},
		{
			name:     "fallback to os name alias",
			metadata: map[string]string{"os_name": "Ubuntu Linux"},
			want:     platforms.Linux,
		},
		{
			name:     "unknown value preserved",
			metadata: map[string]string{"platform": "openbsd"},
			want:     "openbsd",
		},
		{
			name:     "empty",
			metadata: map[string]string{},
			want:     "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveHeartbeatPlatform(tc.metadata); got != tc.want {
				t.Fatalf("resolveHeartbeatPlatform() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPublishNormalizesPlatformMetadata(t *testing.T) {
	t.Setenv(envAllowInsecureTransport, "true")
	t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")

	var captured assets.HeartbeatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets/heartbeat" {
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("failed to decode heartbeat payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	cfg := RuntimeConfig{
		Name:       "labtether-agent",
		APIBaseURL: server.URL,
		APIToken:   "test-token",
		Source:     "agent",
	}
	publisher := NewHeartbeatPublisher(cfg, map[string]string{
		"os_name": "Ubuntu 24.04 LTS",
	})

	if err := publisher.Publish(context.Background(), TelemetrySample{
		AssetID: "node-01",
	}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if captured.Platform != platforms.Linux {
		t.Fatalf("expected canonical platform %q, got %q", platforms.Linux, captured.Platform)
	}
	if captured.Metadata["platform"] != platforms.Linux {
		t.Fatalf("expected metadata platform %q, got %q", platforms.Linux, captured.Metadata["platform"])
	}
}

package agentcore

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentidentity"
	"github.com/labtether/labtether-linux/pkg/assets"
)

func verifyHubTokenProof(t *testing.T, req enrollRequest, version, subject string) {
	t.Helper()
	if req.DeviceKeyAlg != agentidentity.KeyAlgorithmEd25519 || req.DeviceProofVersion != version {
		t.Errorf("identity fields: algorithm=%q version=%q", req.DeviceKeyAlg, req.DeviceProofVersion)
	}
	publicKey, err := base64.StdEncoding.DecodeString(req.DevicePublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		t.Fatal("invalid device public key")
	}
	if req.DeviceFingerprint != agentidentity.FingerprintFromPublicKey(publicKey) {
		t.Error("device fingerprint does not match public key")
	}
	signature, err := base64.StdEncoding.DecodeString(req.DeviceSignature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		t.Fatal("invalid device signature")
	}
	tokenHash := sha256.Sum256([]byte(strings.TrimSpace(req.EnrollmentToken)))
	payload := []byte("labtether-token-enrollment-proof-" + version + "|" + subject + "|" +
		hex.EncodeToString(tokenHash[:]) + "|" + req.DeviceFingerprint)
	if !ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature) {
		t.Error("Hub token proof signature did not verify")
	}
}

func testEnrollmentConfig(t *testing.T) (RuntimeConfig, *deviceIdentity) {
	t.Helper()
	t.Setenv(envAllowInsecureTransport, "true")
	t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")
	dir := t.TempDir()
	cfg := RuntimeConfig{
		AssetID:                 "qa-linux-1",
		TokenFilePath:           filepath.Join(dir, "agent-token"),
		EnrollmentTokenFilePath: filepath.Join(dir, "enrollment-token"),
		DeviceKeyPath:           filepath.Join(dir, "device-key"),
		DevicePublicKeyPath:     filepath.Join(dir, "device-key.pub"),
		DeviceFingerprintPath:   filepath.Join(dir, "device-fingerprint"),
	}
	identity, err := ensureDeviceIdentity(cfg)
	if err != nil {
		t.Fatalf("create device identity: %v", err)
	}
	return cfg, identity
}

func TestLinuxTokenEnrollmentSendsHubProofAndRestoresAssetID(t *testing.T) {
	cfg, identity := testEnrollmentConfig(t)
	cfg.EnrollmentToken = "one-time-token"
	cfg.EnrollmentTokenFromFile = true
	if err := os.WriteFile(cfg.EnrollmentTokenFilePath, []byte(cfg.EnrollmentToken), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/enroll" || r.Method != http.MethodPost {
			t.Errorf("unexpected enrollment route: %s %s", r.Method, r.URL.Path)
		}
		var req enrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Hostname != cfg.AssetID {
			t.Errorf("hostname=%q, want %q", req.Hostname, cfg.AssetID)
		}
		verifyHubTokenProof(t, req, "v1", req.Hostname)
		_ = json.NewEncoder(w).Encode(enrollResponse{
			AgentToken: "issued-agent-token", AssetID: "qa-linux-1",
			HubWSURL: "ws://localhost/ws/agent", HubAPIURL: "http://localhost",
		})
	}))
	defer server.Close()
	cfg.APIBaseURL = server.URL
	if err := resolveTokenWithIdentity(context.Background(), &cfg, identity); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if cfg.APIToken != "issued-agent-token" || cfg.EnrollmentToken != "" {
		t.Fatalf("credential state: token=%q enrollment remaining=%v", cfg.APIToken, cfg.EnrollmentToken != "")
	}
	if _, err := os.Stat(cfg.EnrollmentTokenFilePath); !os.IsNotExist(err) {
		t.Fatalf("consumed token file still exists: %v", err)
	}
	stored, err := os.ReadFile(cfg.TokenFilePath)
	if err != nil || string(stored) != "issued-agent-token\n" {
		t.Fatalf("persisted agent token invalid: %v", err)
	}
	for _, path := range []string{cfg.TokenFilePath, enrollmentStatePath(cfg.TokenFilePath)} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("managed file %s mode=%v, want 0600", path, info.Mode())
		}
	}
	restarted := RuntimeConfig{AssetID: "untrusted-hostname", APIToken: "issued-agent-token", APITokenFromFile: true, TokenFilePath: cfg.TokenFilePath}
	if err := ResolveToken(context.Background(), &restarted); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restarted.AssetID != "qa-linux-1" || restarted.WSBaseURL != "ws://localhost/ws/agent" {
		t.Fatalf("restored identity: asset=%q ws=%q", restarted.AssetID, restarted.WSBaseURL)
	}
}

func TestLinuxReEnrollmentBypassesStaleTokenAndSignsCanonicalAsset(t *testing.T) {
	cfg, identity := testEnrollmentConfig(t)
	cfg.EnrollmentToken = "fresh-one-time-token"
	cfg.EnrollmentTokenFromFile = true
	if err := os.WriteFile(cfg.EnrollmentTokenFilePath, []byte(cfg.EnrollmentToken), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.TokenFilePath, []byte("rejected-agent-token\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cfg.TokenFilePath, 0644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/assets/heartbeat" {
			if got := r.Header.Get("Authorization"); got != "Bearer replacement-agent-token" {
				t.Errorf("recovery fallback sent stale token: %q", got)
			}
			var heartbeat assets.HeartbeatRequest
			if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
				t.Error(err)
			}
			if heartbeat.AssetID != "qa-linux-1" {
				t.Errorf("recovery fallback asset=%q", heartbeat.AssetID)
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		var req enrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Hostname != "qa-linux-1" {
			t.Errorf("recovery hostname=%q", req.Hostname)
		}
		verifyHubTokenProof(t, req, "v2", "qa-linux-1")
		_ = json.NewEncoder(w).Encode(enrollResponse{
			AgentToken: "replacement-agent-token", AssetID: "qa-linux-1",
			HubWSURL: "ws://" + r.Host + "/ws/agent", HubAPIURL: "http://" + r.Host,
		})
	}))
	defer server.Close()
	cfg.APIBaseURL = "http://old-hub.invalid"
	transport := newWSTransport(strings.Replace(server.URL, "http://", "ws://", 1)+"/ws/agent",
		"rejected-agent-token", "qa-linux-1", "linux", "test", nil, cfg.TokenFilePath, identity)
	transport.reEnrollFn = func() (string, error) { return "", nil }
	issued, err := reEnrollAgainstActiveHub(context.Background(), cfg, transport)
	if err != nil || issued != "replacement-agent-token" {
		t.Fatalf("re-enroll issued=%q err=%v", issued, err)
	}
	stored, err := os.ReadFile(cfg.TokenFilePath)
	if err != nil || string(stored) != "replacement-agent-token\n" {
		t.Fatalf("replacement token not stored: %v", err)
	}
	info, err := os.Stat(cfg.TokenFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("replacement token mode=%v, want 0600", info.Mode())
	}
	if _, err := os.Stat(cfg.EnrollmentTokenFilePath); !os.IsNotExist(err) {
		t.Fatalf("consumed token file still exists: %v", err)
	}
	if transport.reEnrollFn != nil {
		t.Fatal("consumed token left recovery callback armed")
	}
	publisher := newHeartbeatPublisher(cfg, nil, transport.identitySnapshot)
	if err := publisher.Publish(context.Background(), TelemetrySample{AssetID: "old-hostname"}); err != nil {
		t.Fatalf("HTTP fallback after signed re-enrollment: %v", err)
	}
}

func TestRuntimeUsesHubIssuedAssetIDAfterEnrollment(t *testing.T) {
	runtime := NewRuntime(RuntimeConfig{AssetID: "canonical-device"},
		stubProvider{sample: TelemetrySample{AssetID: "pre-enrollment-hostname"}}, noopHeartbeatPublisher{})
	runtime.collectOnce(time.Now())
	if got := runtime.current().AssetID; got != "canonical-device" {
		t.Fatalf("telemetry asset ID=%q, want Hub-issued ID", got)
	}
}

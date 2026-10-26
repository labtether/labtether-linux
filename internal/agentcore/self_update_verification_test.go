package agentcore

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestFetchReleaseMetadataBoundsAndAuth(t *testing.T) {
	t.Setenv(envAllowInsecureTransport, "true")
	t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")

	t.Run("custom endpoint does not receive Hub token", func(t *testing.T) {
		seenAuthorization := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenAuthorization = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(agentReleaseMetadata{Version: "v2.0.0"})
		}))
		defer server.Close()

		if _, err := fetchReleaseMetadata(RuntimeConfig{APIToken: "hub-secret"}, server.URL); err != nil {
			t.Fatalf("fetchReleaseMetadata returned error: %v", err)
		}
		if seenAuthorization != "" {
			t.Fatalf("custom endpoint received Hub authorization header %q", seenAuthorization)
		}
	})

	t.Run("canonical Hub origin may receive Hub token", func(t *testing.T) {
		seenAuthorization := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenAuthorization = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(agentReleaseMetadata{Version: "v2.0.0"})
		}))
		defer server.Close()

		cfg := RuntimeConfig{APIBaseURL: server.URL, APIToken: "hub-secret"}
		if _, err := fetchReleaseMetadata(cfg, server.URL+"/api/v1/agent/releases/latest"); err != nil {
			t.Fatalf("fetchReleaseMetadata returned error: %v", err)
		}
		if seenAuthorization != "Bearer hub-secret" {
			t.Fatalf("canonical endpoint authorization header = %q", seenAuthorization)
		}
	})

	t.Run("content length above limit is rejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(maxSelfUpdateMetadataSize+1))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		if _, err := fetchReleaseMetadata(RuntimeConfig{}, server.URL); err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("expected metadata size error, got %v", err)
		}
	})

	t.Run("chunked body above limit is rejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Transfer-Encoding", "chunked")
			_, _ = w.Write([]byte(strings.Repeat("x", maxSelfUpdateMetadataSize+1)))
		}))
		defer server.Close()

		if _, err := fetchReleaseMetadata(RuntimeConfig{}, server.URL); err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("expected metadata size error, got %v", err)
		}
	})

	t.Run("trailing JSON is rejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"version":"v2.0.0"}{"extra":true}`))
		}))
		defer server.Close()

		if _, err := fetchReleaseMetadata(RuntimeConfig{}, server.URL); err == nil || !strings.Contains(err.Error(), "trailing JSON") {
			t.Fatalf("expected trailing JSON error, got %v", err)
		}
	})

	t.Run("cross-origin redirect strips Hub token", func(t *testing.T) {
		seenAuthorization := ""
		destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenAuthorization = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(agentReleaseMetadata{Version: "v2.0.0"})
		}))
		defer destination.Close()
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", destination.URL)
			w.WriteHeader(http.StatusFound)
		}))
		defer origin.Close()

		cfg := RuntimeConfig{APIBaseURL: origin.URL, APIToken: "hub-secret"}
		if _, err := fetchReleaseMetadata(cfg, origin.URL+"/api/v1/agent/releases/latest"); err != nil {
			t.Fatalf("fetch redirected metadata: %v", err)
		}
		if seenAuthorization != "" {
			t.Fatalf("redirect destination received Hub authorization header %q", seenAuthorization)
		}
	})
}

func TestValidateReleaseTargetRejectsWrongPlatformAndRollback(t *testing.T) {
	valid := agentReleaseMetadata{Version: "v2.0.0", OS: runtime.GOOS, Arch: runtime.GOARCH}
	if err := validateReleaseTarget(RuntimeConfig{Version: "v1.9.0"}, valid); err != nil {
		t.Fatalf("newer matching release rejected: %v", err)
	}

	wrongOS := valid
	wrongOS.OS = "not-" + runtime.GOOS
	if err := validateReleaseTarget(RuntimeConfig{Version: "v1.9.0"}, wrongOS); err == nil {
		t.Fatal("expected wrong OS to be rejected")
	}

	wrongArch := valid
	wrongArch.Arch = "not-" + runtime.GOARCH
	if err := validateReleaseTarget(RuntimeConfig{Version: "v1.9.0"}, wrongArch); err == nil {
		t.Fatal("expected wrong architecture to be rejected")
	}

	rollback := valid
	rollback.Version = "v1.8.9"
	if err := validateReleaseTarget(RuntimeConfig{Version: "v1.9.0"}, rollback); err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("expected rollback to be rejected, got %v", err)
	}

	if err := validateReleaseTarget(RuntimeConfig{Version: "development"}, valid); err != nil {
		t.Fatalf("development build should accept a stable release: %v", err)
	}
	hugeRollback := valid
	hugeRollback.Version = "v999999999999999999999999999999.0.0"
	if err := validateReleaseTarget(RuntimeConfig{Version: "v1000000000000000000000000000000.0.0"}, hugeRollback); err == nil {
		t.Fatal("expected large numeric rollback to be rejected")
	}
	invalidRelease := valid
	invalidRelease.Version = "latest"
	if err := validateReleaseTarget(RuntimeConfig{Version: "v1.9.0"}, invalidRelease); err == nil {
		t.Fatal("expected non-semantic release version to be rejected")
	}
}

func TestUnsignedSelfUpdateRequiresExplicitDevelopmentOptIn(t *testing.T) {
	t.Setenv(envSelfUpdateTrustedPublicKey, "")
	t.Setenv(envSelfUpdateAcceptUnsigned, "false")
	release := agentReleaseMetadata{}
	if err := verifyReleaseMetadataSignature(release, "https://example.invalid/binary"); err == nil {
		t.Fatal("missing trust key silently accepted an unsigned update")
	}
	t.Setenv(envSelfUpdateAcceptUnsigned, "true")
	if err := verifyReleaseMetadataSignature(release, "https://example.invalid/binary"); err != nil {
		t.Fatalf("explicit development opt-in rejected: %v", err)
	}
	t.Setenv(envSelfUpdateTrustedPublicKey, "malformed")
	if err := verifyReleaseMetadataSignature(release, "https://example.invalid/binary"); err == nil {
		t.Fatal("unsigned opt-in bypassed an invalid configured trust key")
	}
}

func TestSelfUpdateRefusalsLeaveExecutableUntouched(t *testing.T) {
	t.Setenv(envAllowInsecureTransport, "false")
	t.Setenv("LABTETHER_OUTBOUND_ALLOW_LOOPBACK", "true")
	t.Setenv(envSelfUpdateAcceptUnsigned, "false")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"missing trust key", "missing signature", "invalid signature", "wrong OS", "wrong arch", "forced downgrade"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv(envSelfUpdateTrustedPublicKey, base64.StdEncoding.EncodeToString(publicKey))
			binary := []byte("unapplied candidate")
			var release agentReleaseMetadata
			binaryRequests := make(chan struct{}, 1)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/release" {
					_ = json.NewEncoder(w).Encode(release)
					return
				}
				binaryRequests <- struct{}{}
				_, _ = w.Write(binary)
			}))
			defer server.Close()
			release = agentReleaseMetadata{Version: "v2.0.0", OS: runtime.GOOS, Arch: runtime.GOARCH,
				SHA256: sha256Hex(binary), URL: server.URL + "/binary", SizeBytes: int64(len(binary))}
			switch kind {
			case "wrong OS":
				release.OS = "not-" + runtime.GOOS
			case "wrong arch":
				release.Arch = "not-" + runtime.GOARCH
			case "forced downgrade":
				release.Version = "v1.0.0"
			}
			release.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(selfUpdateSignaturePayload(release, release.URL))))
			switch kind {
			case "missing trust key":
				t.Setenv(envSelfUpdateTrustedPublicKey, "")
			case "missing signature":
				release.Signature = ""
			case "invalid signature":
				release.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
			}
			path := filepath.Join(t.TempDir(), "dummy-executable")
			original := []byte("current executable fixture")
			if err := os.WriteFile(path, original, 0o700); err != nil {
				t.Fatal(err)
			}
			previous := executablePathFn
			executablePathFn = func() (string, error) { return path, nil }
			t.Cleanup(func() { executablePathFn = previous })
			cfg := selfUpdateTLSConfig(t, server)
			cfg.AutoUpdateCheckURL = server.URL + "/release"
			cfg.Version = "v1.9.0"
			updated, _, err := checkAndApplySelfUpdateWithOptions(cfg, selfUpdateOptions{Force: true})
			if err == nil || updated {
				t.Fatalf("rejected update result = %v, %v", updated, err)
			}
			select {
			case <-binaryRequests:
				t.Fatal("rejected release downloaded a binary")
			default:
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(original) {
				t.Fatal("rejected release changed executable")
			}
		})
	}
}

func TestReleaseMetadataFieldBounds(t *testing.T) {
	valid := agentReleaseMetadata{Version: "v2.0.0", OS: runtime.GOOS, Arch: runtime.GOARCH,
		SHA256: strings.Repeat("a", 64), URL: "/binary", SizeBytes: 1}
	endpoint := "https://hub.example.invalid/release"
	if err := validateReleaseMetadata(endpoint, "https://hub.example.invalid/binary", valid); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*agentReleaseMetadata){
		"long version":    func(r *agentReleaseMetadata) { r.Version = strings.Repeat("v", maxSelfUpdateVersionLength+1) },
		"missing OS":      func(r *agentReleaseMetadata) { r.OS = "" },
		"long OS":         func(r *agentReleaseMetadata) { r.OS = strings.Repeat("x", maxSelfUpdatePlatformLength+1) },
		"missing arch":    func(r *agentReleaseMetadata) { r.Arch = "" },
		"long arch":       func(r *agentReleaseMetadata) { r.Arch = strings.Repeat("x", maxSelfUpdatePlatformLength+1) },
		"long URL":        func(r *agentReleaseMetadata) { r.URL = strings.Repeat("x", maxSelfUpdateURLLength+1) },
		"long signature":  func(r *agentReleaseMetadata) { r.Signature = strings.Repeat("x", maxSelfUpdateSignatureLength+1) },
		"bad digest":      func(r *agentReleaseMetadata) { r.SHA256 = "not-a-digest" },
		"negative size":   func(r *agentReleaseMetadata) { r.SizeBytes = -1 },
		"oversize binary": func(r *agentReleaseMetadata) { r.SizeBytes = maxSelfUpdateBinarySize + 1 },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			release := valid
			mutate(&release)
			if err := validateReleaseMetadata(endpoint, "https://hub.example.invalid/binary", release); err == nil {
				t.Fatal("invalid metadata accepted")
			}
		})
	}
}

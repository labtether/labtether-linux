package securityruntime

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Enrollment and NPM login send credentials as JSON bodies. HTTP 307 replays
// those bodies, so the policy owner must reject an unapproved redirect before
// that server sees any request, even when TLS is otherwise trusted.
func TestOutboundRequestRejectsCredentialBodyRedirect(t *testing.T) {
	t.Setenv(envOutboundAllowlistMode, "true")
	t.Setenv(envOutboundAllowedHosts, "127.0.0.1")
	t.Setenv(envOutboundAllowLoopback, "true")
	payload := []byte(`{"identity":"qa-user","secret":"qa-only-not-real"}`)
	var targetRequests atomic.Int32
	var replayed atomic.Bool
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequests.Add(1)
		body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		replayed.Store(bytes.Equal(body, payload))
		_, _ = w.Write([]byte("received"))
	}))
	defer target.Close()
	redirect := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := NewOutboundRequestWithContext(ctx, http.MethodPost, origin.URL+"/api/tokens", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Both fixtures present httptest's trusted example.com certificate. A
	// trusted TLS alias must not become a bypass for the URL host allowlist.
	client := origin.Client()
	transport := client.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	client.Transport = transport
	response, err := DoOutboundRequest(client, req)
	if response != nil {
		_ = response.Body.Close()
	}
	if targetRequests.Load() != 0 {
		t.Errorf("unapproved redirect received %d requests; credential body replayed=%t", targetRequests.Load(), replayed.Load())
	}
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("redirect was not rejected by host policy: %v", err)
	}
}

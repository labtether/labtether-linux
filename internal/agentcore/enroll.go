package agentcore

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentidentity"
	"github.com/labtether/labtether-linux/pkg/securityruntime"
)

// ResolveToken sets cfg.APIToken using the first available source:
// 1. Explicit LABTETHER_API_TOKEN env var (already loaded)
// 2. Persisted token file on disk
// 3. Enrollment with the hub using LABTETHER_ENROLLMENT_TOKEN
func ResolveToken(ctx context.Context, cfg *RuntimeConfig) error {
	return resolveTokenWithIdentity(ctx, cfg, nil)
}

func resolveTokenWithIdentity(ctx context.Context, cfg *RuntimeConfig, identity *deviceIdentity) error {
	// Priority 1: explicit API token
	if cfg.APIToken != "" {
		if cfg.APITokenFromFile {
			if err := restoreEnrollmentState(cfg); err != nil {
				log.Printf("agent: warning: could not restore enrollment state: %v", err)
			}
		}
		if cfg.APIBaseURL == "" {
			cfg.APIBaseURL = apiBaseURLFromWS(cfg.WSBaseURL)
		}
		return nil
	}

	// Priority 2: persisted token file
	if token, err := loadTokenFromFile(cfg.TokenFilePath); err == nil && token != "" {
		log.Printf("agent: loaded token from %s", cfg.TokenFilePath)
		cfg.APIToken = token
		cfg.APITokenFromFile = true
		if err := restoreEnrollmentState(cfg); err != nil {
			log.Printf("agent: warning: could not restore enrollment state: %v", err)
		}
		if cfg.APIBaseURL == "" {
			cfg.APIBaseURL = apiBaseURLFromWS(cfg.WSBaseURL)
		}
		return nil
	}

	// Priority 3: enrollment
	if cfg.EnrollmentToken == "" {
		return nil // no enrollment token set — agent will run without auth (legacy)
	}
	if cfg.WSBaseURL == "" && cfg.APIBaseURL == "" {
		return fmt.Errorf("enrollment requires LABTETHER_WS_URL or LABTETHER_API_BASE_URL")
	}

	log.Printf("agent: enrolling with hub...")
	resp, err := enrollWithHubWithIdentityProof(ctx, cfg, identity, "")
	if err != nil {
		return fmt.Errorf("enrollment failed: %w", err)
	}

	cfg.APIToken = resp.AgentToken
	if resp.AssetID != "" {
		cfg.AssetID = resp.AssetID
	}
	if normalized := normalizeWSBaseURL(resp.HubWSURL); normalized != "" {
		cfg.WSBaseURL = normalized
	}
	if normalized := normalizeAPIBaseURL(resp.HubAPIURL); normalized != "" {
		cfg.APIBaseURL = normalized
	}

	// Persist token to disk
	if err := saveTokenToFile(cfg.TokenFilePath, resp.AgentToken); err != nil {
		log.Printf("agent: warning: could not persist token to %s: %v", cfg.TokenFilePath, err)
		if cfg.TokenFilePath != "" {
			if removeErr := os.Remove(cfg.TokenFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
				log.Printf("agent: warning: could not remove stale token: %v", removeErr)
			}
		}
	} else {
		log.Printf("agent: token persisted to %s", cfg.TokenFilePath)
		if err := saveEnrollmentState(cfg.TokenFilePath, enrollmentState{
			AssetID: cfg.AssetID, HubWSURL: cfg.WSBaseURL, HubAPIURL: cfg.APIBaseURL,
		}); err != nil {
			log.Printf("agent: warning: could not persist enrollment state: %v", err)
		}
	}

	// Preserve an operator-supplied CA for a TLS proxy in front of the Hub.
	// The Hub's returned internal CA may not trust that public endpoint.
	caPath := filepath.Join(filepath.Dir(cfg.TokenFilePath), "ca.crt")
	if resp.CACertPEM != "" && cfg.TokenFilePath != "" &&
		(cfg.TLSCAFile == "" || filepath.Clean(cfg.TLSCAFile) == filepath.Clean(caPath)) {
		if err := validateCACertPEM(resp.CACertPEM); err != nil {
			log.Printf("agent: warning: hub returned invalid CA certificate: %v (ignoring)", err)
		} else {
			if err := os.WriteFile(caPath, []byte(resp.CACertPEM), 0644); err != nil { // #nosec G306 -- CA certificate is public trust material, not a private secret.
				log.Printf("agent: warning: could not save hub CA to %s: %v", caPath, err)
			} else {
				log.Printf("agent: saved hub CA certificate to %s", caPath)
				cfg.TLSCAFile = caPath
			}
		}
	}

	log.Printf("agent: enrolled successfully as %s", cfg.AssetID)
	return discardConsumedEnrollmentToken(cfg)
}

type enrollRequest struct {
	EnrollmentToken    string `json:"enrollment_token"`
	Hostname           string `json:"hostname"`
	Platform           string `json:"platform"`
	GroupID            string `json:"group_id,omitempty"`
	DeviceKeyAlg       string `json:"device_key_algorithm,omitempty"`
	DevicePublicKey    string `json:"device_public_key,omitempty"`
	DeviceFingerprint  string `json:"device_fingerprint,omitempty"`
	DeviceSignature    string `json:"device_signature,omitempty"`
	DeviceProofVersion string `json:"device_proof_version,omitempty"`
}

type enrollResponse struct {
	AgentToken string `json:"agent_token"`
	AssetID    string `json:"asset_id"`
	HubWSURL   string `json:"hub_ws_url"`
	HubAPIURL  string `json:"hub_api_url"`
	CACertPEM  string `json:"ca_cert_pem,omitempty"`
}

func enrollWithHub(ctx context.Context, cfg *RuntimeConfig) (*enrollResponse, error) {
	return enrollWithHubWithIdentityProof(ctx, cfg, nil, "")
}

func enrollWithHubWithIdentityProof(ctx context.Context, cfg *RuntimeConfig, identity *deviceIdentity, continuityAssetID string) (*enrollResponse, error) {
	// Build enroll URL from WSBaseURL or APIBaseURL
	var enrollURL string
	if cfg.APIBaseURL != "" {
		enrollURL = strings.TrimRight(normalizeAPIBaseURL(cfg.APIBaseURL), "/") + "/api/v1/enroll"
	} else if cfg.WSBaseURL != "" {
		parsedWS, err := url.Parse(normalizeWSBaseURL(cfg.WSBaseURL))
		if err != nil || strings.TrimSpace(parsedWS.Host) == "" {
			return nil, fmt.Errorf("invalid websocket url for enrollment")
		}
		switch strings.ToLower(strings.TrimSpace(parsedWS.Scheme)) {
		case "wss":
			parsedWS.Scheme = "https"
		case "ws":
			if allowInsecureTransportOptIn() {
				parsedWS.Scheme = "http"
			} else {
				parsedWS.Scheme = "https"
			}
		default:
			return nil, fmt.Errorf("unsupported websocket scheme for enrollment: %s", parsedWS.Scheme)
		}
		parsedWS.Path = ""
		parsedWS.RawPath = ""
		parsedWS.RawQuery = ""
		parsedWS.Fragment = ""
		enrollURL = strings.TrimRight(parsedWS.String(), "/") + "/api/v1/enroll"
	}

	hostname, _ := os.Hostname()
	if validEnrollmentAssetID(cfg.AssetID) {
		hostname = cfg.AssetID
	}
	if continuityAssetID != "" {
		hostname = continuityAssetID
	}
	if hostname == "" {
		hostname = "labtether-agent"
	}

	reqBody := enrollRequest{
		EnrollmentToken: cfg.EnrollmentToken,
		Hostname:        hostname,
		Platform:        runtime.GOOS,
		GroupID:         cfg.GroupID,
	}
	if identity != nil {
		reqBody.DeviceKeyAlg = identity.KeyAlgorithm
		reqBody.DevicePublicKey = identity.PublicKeyBase64
		reqBody.DeviceFingerprint = identity.Fingerprint
		var payload []byte
		if continuityAssetID != "" {
			reqBody.DeviceProofVersion = "v2"
			payload = agentidentity.BuildTokenEnrollmentProofPayloadV2(continuityAssetID, cfg.EnrollmentToken, identity.Fingerprint)
		} else {
			reqBody.DeviceProofVersion = "v1"
			payload = agentidentity.BuildTokenEnrollmentProofPayload(hostname, cfg.EnrollmentToken, identity.Fingerprint)
		}
		reqBody.DeviceSignature = base64.StdEncoding.EncodeToString(ed25519.Sign(identity.PrivateKey, payload))
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := securityruntime.NewOutboundRequestWithContext(ctx, http.MethodPost, enrollURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	if tlsCfg := buildTLSConfig(cfg); tlsCfg != nil {
		client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
	}
	resp, err := securityruntime.DoOutboundRequest(client, httpReq)
	if err != nil {
		return nil, fmt.Errorf("enroll HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("enroll returned status %d", resp.StatusCode)
	}

	var result enrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("enroll response decode failed: %w", err)
	}
	if result.AgentToken == "" {
		return nil, fmt.Errorf("enroll response missing agent_token")
	}
	return &result, nil
}

func loadTokenFromFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path) // #nosec G304 -- Enrollment token path is controlled runtime configuration or default state.
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func saveTokenToFile(path, token string) error {
	if path == "" {
		return nil
	}
	return writeSecretFileAtomic(path, []byte(token+"\n"))
}

// validateCACertPEM parses a PEM-encoded certificate and verifies it has the
// CA basic constraint set. This prevents a compromised hub from injecting a
// non-CA certificate that could be used to intercept traffic.
func validateCACertPEM(pemData string) error {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil || block.Type != "CERTIFICATE" {
		return fmt.Errorf("PEM data does not contain a CERTIFICATE block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}
	if !cert.IsCA {
		return fmt.Errorf("certificate is not a CA (BasicConstraints.IsCA=false)")
	}
	return nil
}

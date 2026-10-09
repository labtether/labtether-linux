package agentcore

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

// reEnrollAgainstActiveHub bypasses the rejected on-disk agent credential.
// The signed v2 proof binds a fresh one-time token to this exact Hub asset ID.
func reEnrollAgainstActiveHub(ctx context.Context, cfg RuntimeConfig, transport *wsTransport) (string, error) {
	if transport == nil || transport.deviceIdentity == nil {
		return "", fmt.Errorf("re-enrollment device identity is unavailable")
	}
	if strings.TrimSpace(cfg.EnrollmentToken) == "" {
		return "", fmt.Errorf("re-enrollment token is unavailable")
	}
	assetID := canonicalEnrollmentAssetID(transport.AssetID())
	current := transport.identitySnapshot()
	if assetID == "" || strings.TrimSpace(current.wsBaseURL) == "" {
		return "", fmt.Errorf("re-enrollment connection identity is unavailable")
	}

	cfgCopy := cfg
	cfgCopy.APIToken = ""
	cfgCopy.GroupID = current.groupID
	cfgCopy.APIBaseURL = current.apiBaseURL
	cfgCopy.WSBaseURL = current.wsBaseURL
	resp, err := enrollWithHubWithIdentityProof(ctx, &cfgCopy, transport.deviceIdentity, assetID)
	if err != nil {
		return "", err
	}
	if resp.AssetID != assetID {
		return "", fmt.Errorf("re-enrollment returned a different asset id")
	}
	nextAPI := resp.HubAPIURL
	if resp.HubWSURL == "" && nextAPI == "" {
		nextAPI = current.apiBaseURL
	}
	adopted, err := transport.adoptCredential(resp.AgentToken, resp.AssetID, resp.HubWSURL, nextAPI, resp.GroupID)
	if err != nil {
		return "", fmt.Errorf("invalid replacement credential: %w", err)
	}
	if err := saveTokenToFile(cfg.TokenFilePath, resp.AgentToken); err != nil {
		log.Printf("agentws: replacement token is memory-only: %v", err)
		if cfg.TokenFilePath != "" {
			if removeErr := os.Remove(cfg.TokenFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
				log.Printf("agentws: could not remove stale token: %v", removeErr)
			}
		}
	} else if err := saveEnrollmentState(cfg.TokenFilePath, enrollmentState{
		AssetID: adopted.assetID, GroupID: canonicalGroupIDPointer(adopted.groupID, adopted.groupKnown),
		HubWSURL: adopted.wsBaseURL, HubAPIURL: adopted.apiBaseURL,
	}); err != nil {
		log.Printf("agentws: could not persist re-enrollment state: %v", err)
	}
	if err := discardConsumedEnrollmentToken(&cfgCopy); err != nil {
		log.Printf("agentws: consumed re-enrollment token cleanup failed: %v", err)
	}
	transport.mu.Lock()
	transport.reEnrollFn = nil
	transport.mu.Unlock()
	return resp.AgentToken, nil
}

func discardConsumedEnrollmentToken(cfg *RuntimeConfig) error {
	if cfg == nil {
		return nil
	}
	defer func() {
		cfg.EnrollmentToken = ""
		cfg.EnrollmentTokenFromFile = false
		_ = os.Unsetenv("LABTETHER_ENROLLMENT_TOKEN")
	}()
	if !cfg.EnrollmentTokenFromFile || cfg.EnrollmentTokenFilePath == "" {
		return nil
	}
	if cfg.EnrollmentTokenFilePath == cfg.TokenFilePath {
		return fmt.Errorf("enrollment token file and agent token file must differ")
	}
	if err := os.Remove(cfg.EnrollmentTokenFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove consumed enrollment token file: %w", err)
	}
	return nil
}

// This matches Hub's NormalizeHostnameForAssetID for a known asset ID.
func canonicalEnrollmentAssetID(assetID string) string {
	const maxAssetIDBytes = 64
	canonical := strings.ToLower(strings.TrimSpace(assetID))
	if len(canonical) > maxAssetIDBytes {
		canonical = canonical[:maxAssetIDBytes]
	}
	var normalized strings.Builder
	for _, ch := range canonical {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9', ch == '-' || ch == '_' || ch == '.':
			normalized.WriteRune(ch)
		default:
			normalized.WriteByte('-')
		}
	}
	return strings.Trim(normalized.String(), "-")
}

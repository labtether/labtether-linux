package agentcore

import (
	"fmt"
	"strings"
)

// transportIdentity is copied under the transport lock. A heartbeat or dial
// uses one token, asset ID, and Hub origin from the same credential generation.
type transportIdentity struct {
	token      string
	assetID    string
	wsBaseURL  string
	apiBaseURL string
	generation uint64
}

func (t *wsTransport) identitySnapshot() transportIdentity {
	t.mu.Lock()
	defer t.mu.Unlock()
	apiBaseURL := t.apiBaseURL
	if apiBaseURL == "" {
		apiBaseURL = apiBaseURLFromWS(t.url)
	}
	return transportIdentity{
		token: t.token, assetID: t.assetID, wsBaseURL: t.url,
		apiBaseURL: apiBaseURL, generation: t.identityGeneration,
	}
}

// adoptCredential installs a Hub-issued credential and its routing identity
// together, including the API origin used by HTTP fallback.
func (t *wsTransport) adoptCredential(token, assetID, wsBaseURL, apiBaseURL string) (transportIdentity, error) {
	token = strings.TrimSpace(token)
	assetID = strings.TrimSpace(assetID)
	if err := validateIssuedAgentToken(token); err != nil {
		return transportIdentity{}, err
	}
	if !validEnrollmentAssetID(assetID) {
		return transportIdentity{}, fmt.Errorf("canonical asset id is invalid")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if wsBaseURL == "" {
		wsBaseURL = t.url
	}
	wsBaseURL = normalizeWSBaseURL(wsBaseURL)
	if apiBaseURL == "" {
		apiBaseURL = apiBaseURLFromWS(wsBaseURL)
	}
	apiBaseURL = normalizeAPIBaseURL(apiBaseURL)
	if err := validateEnrollmentState(enrollmentState{
		Version: enrollmentStateVersion, AssetID: assetID,
		HubWSURL: wsBaseURL, HubAPIURL: apiBaseURL,
	}); err != nil {
		return transportIdentity{}, err
	}
	if wsBaseURL == "" || apiBaseURL == "" {
		return transportIdentity{}, fmt.Errorf("enrollment hub origin is unavailable")
	}
	t.token, t.assetID = token, assetID
	t.url, t.apiBaseURL = wsBaseURL, apiBaseURL
	t.identityGeneration++
	t.consecutiveAuthFailures = 0
	t.lastError = ""
	result := transportIdentity{
		token: token, assetID: assetID, wsBaseURL: wsBaseURL,
		apiBaseURL: apiBaseURL, generation: t.identityGeneration,
	}
	return result, nil
}

func validateIssuedAgentToken(token string) error {
	if token == "" || len(token) > 4096 {
		return fmt.Errorf("agent credential length is invalid")
	}
	for _, ch := range token {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '-', ch == '.', ch == '_', ch == '~', ch == '+', ch == '/', ch == '=':
		default:
			return fmt.Errorf("agent credential contains invalid characters")
		}
	}
	return nil
}

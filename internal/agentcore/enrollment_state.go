package agentcore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const enrollmentStateFileName = "enrollment-state.json"
const enrollmentStateVersion = 1
const maxEnrollmentStateBytes int64 = 64 * 1024

type enrollmentState struct {
	Version   int    `json:"version"`
	AssetID   string `json:"asset_id"`
	HubWSURL  string `json:"hub_ws_url,omitempty"`
	HubAPIURL string `json:"hub_api_url,omitempty"`
}

func enrollmentStatePath(tokenFilePath string) string {
	if strings.TrimSpace(tokenFilePath) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(tokenFilePath), enrollmentStateFileName)
}

func saveEnrollmentState(tokenFilePath string, state enrollmentState) error {
	path := enrollmentStatePath(tokenFilePath)
	if path == "" {
		return nil
	}
	state.AssetID = strings.TrimSpace(state.AssetID)
	state.Version = enrollmentStateVersion
	state.HubWSURL = normalizeWSBaseURL(state.HubWSURL)
	state.HubAPIURL = normalizeAPIBaseURL(state.HubAPIURL)
	if state.HubAPIURL == "" {
		state.HubAPIURL = apiBaseURLFromWS(state.HubWSURL)
	}
	if err := validateEnrollmentState(state); err != nil {
		return err
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal enrollment state: %w", err)
	}
	if err := writeSecretFileAtomic(path, raw); err != nil {
		return fmt.Errorf("commit enrollment state: %w", err)
	}
	return nil
}

func restoreEnrollmentState(cfg *RuntimeConfig) error {
	path := enrollmentStatePath(cfg.TokenFilePath)
	if path == "" {
		return nil
	}
	file, err := os.Open(path) // #nosec G304 -- Path is derived from configured agent token storage.
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open enrollment state: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat enrollment state: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxEnrollmentStateBytes {
		return fmt.Errorf("invalid enrollment state file")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxEnrollmentStateBytes+1))
	if err != nil || int64(len(raw)) > maxEnrollmentStateBytes {
		return fmt.Errorf("read enrollment state: invalid or too large")
	}
	var state enrollmentState
	if err := json.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("decode enrollment state: %w", err)
	}
	if err := validateEnrollmentState(state); err != nil {
		return err
	}
	// A single endpoint override moves both transports to the chosen origin.
	// A deliberately split WS/API setup must provide both endpoints explicitly.
	cfg.AssetID = strings.TrimSpace(state.AssetID)
	configuredWS := normalizeWSBaseURL(cfg.WSBaseURL)
	configuredAPI := normalizeAPIBaseURL(cfg.APIBaseURL)
	savedWS := normalizeWSBaseURL(state.HubWSURL)
	savedAPI := normalizeAPIBaseURL(state.HubAPIURL)
	switch {
	case configuredWS != "" && configuredAPI != "":
		cfg.WSBaseURL, cfg.APIBaseURL = configuredWS, configuredAPI
	case configuredWS != "":
		cfg.WSBaseURL, cfg.APIBaseURL = configuredWS, apiBaseURLFromWS(configuredWS)
	case configuredAPI != "":
		cfg.WSBaseURL, cfg.APIBaseURL = wsBaseURLFromAPI(configuredAPI, savedWS), configuredAPI
	default:
		cfg.WSBaseURL, cfg.APIBaseURL = savedWS, savedAPI
		if cfg.APIBaseURL == "" {
			cfg.APIBaseURL = apiBaseURLFromWS(cfg.WSBaseURL)
		}
	}
	return nil
}

func validateEnrollmentState(state enrollmentState) error {
	if state.Version != enrollmentStateVersion {
		return fmt.Errorf("invalid enrollment state version")
	}
	if !validEnrollmentAssetID(strings.TrimSpace(state.AssetID)) {
		return fmt.Errorf("invalid enrollment asset id")
	}
	for _, endpoint := range []struct {
		raw       string
		websocket bool
	}{
		{state.HubWSURL, true}, {state.HubAPIURL, false},
	} {
		if endpoint.raw == "" {
			continue
		}
		parsed, err := url.Parse(endpoint.raw)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("invalid enrollment hub URL")
		}
		scheme := strings.ToLower(parsed.Scheme)
		if endpoint.websocket && scheme != "wss" && !(scheme == "ws" && allowInsecureTransportOptIn()) {
			return fmt.Errorf("invalid enrollment websocket URL")
		}
		if !endpoint.websocket && scheme != "https" && !(scheme == "http" && allowInsecureTransportOptIn()) {
			return fmt.Errorf("invalid enrollment API URL")
		}
	}
	return nil
}

func validEnrollmentAssetID(assetID string) bool {
	if assetID == "" || len(assetID) > 128 {
		return false
	}
	for _, ch := range assetID {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') &&
			(ch < '0' || ch > '9') && ch != '-' && ch != '_' && ch != '.' {
			return false
		}
	}
	return true
}

func apiBaseURLFromWS(raw string) string {
	parsed, err := url.Parse(normalizeWSBaseURL(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	switch parsed.Scheme {
	case "wss":
		return "https://" + parsed.Host
	case "ws":
		if allowInsecureTransportOptIn() {
			return "http://" + parsed.Host
		}
	}
	return ""
}

func wsBaseURLFromAPI(rawAPI, savedWS string) string {
	api, err := url.Parse(normalizeAPIBaseURL(rawAPI))
	if err != nil || api.Host == "" || api.User != nil {
		return ""
	}
	ws := &url.URL{Host: api.Host, Path: "/ws/agent"}
	if saved, err := url.Parse(normalizeWSBaseURL(savedWS)); err == nil && saved.Path != "" {
		ws.Path, ws.RawPath = saved.Path, saved.RawPath
	}
	switch api.Scheme {
	case "https":
		ws.Scheme = "wss"
	case "http":
		if !allowInsecureTransportOptIn() {
			return ""
		}
		ws.Scheme = "ws"
	default:
		return ""
	}
	return ws.String()
}

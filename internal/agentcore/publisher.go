package agentcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labtether/labtether-linux/pkg/assets"
	"github.com/labtether/labtether-linux/pkg/metricschema"
	"github.com/labtether/labtether-linux/pkg/platforms"
	"github.com/labtether/labtether-linux/pkg/securityruntime"
)

func NewHeartbeatPublisher(cfg RuntimeConfig, staticMetadata map[string]string) HeartbeatPublisher {
	return newHeartbeatPublisher(cfg, staticMetadata, nil)
}

func newHeartbeatPublisher(cfg RuntimeConfig, staticMetadata map[string]string, identitySource func() transportIdentity) HeartbeatPublisher {
	cfg.APIBaseURL = normalizeAPIBaseURL(cfg.APIBaseURL)
	if identitySource == nil && (cfg.APIBaseURL == "" || cfg.APIToken == "") {
		log.Printf("%s heartbeat disabled: LABTETHER_API_BASE_URL/LABTETHER_API_TOKEN not configured", cfg.Name)
		return noopHeartbeatPublisher{}
	}
	transport := &http.Transport{
		MaxIdleConns:        2,
		MaxIdleConnsPerHost: 1,
		IdleConnTimeout:     90 * time.Second,
	}
	if tlsCfg := buildTLSConfig(&cfg); tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	scheme := "https"
	if endpoint, err := url.Parse(cfg.APIBaseURL); err == nil && endpoint.Scheme != "" {
		scheme = endpoint.Scheme
	}

	// One owned secure pool lets fallback heartbeats reuse their TLS connection;
	// URL, dial and redirect policy are still checked on every request.
	return &apiHeartbeatPublisher{
		client: securityruntime.NewReusableOutboundHTTPClient(&http.Client{
			Timeout:   6 * time.Second,
			Transport: transport,
		}, scheme),
		cfg:            cfg,
		meta:           cloneStringMap(staticMetadata),
		identitySource: identitySource,
	}
}

type apiHeartbeatPublisher struct {
	client         *http.Client
	cfg            RuntimeConfig
	meta           map[string]string
	identitySource func() transportIdentity
}

func (p *apiHeartbeatPublisher) Publish(ctx context.Context, sample TelemetrySample) error {
	identity := transportIdentity{
		token: p.cfg.APIToken, assetID: sample.AssetID, groupID: p.cfg.GroupID,
		apiBaseURL: p.cfg.APIBaseURL,
	}
	if p.identitySource != nil {
		identity = p.identitySource()
	}
	if identity.token == "" || identity.apiBaseURL == "" || identity.assetID == "" {
		return fmt.Errorf("heartbeat credentials or hub origin unavailable")
	}
	groupID := identity.groupID
	if groupID == "" && !identity.groupKnown {
		groupID = p.cfg.GroupID
	}
	metadata := heartbeatMetadata(p.meta, sample)
	metadata[metricschema.HeartbeatKeyCPUPercent] = fmt.Sprintf("%.2f", sample.CPUPercent)
	metadata[metricschema.HeartbeatKeyCPUUsedPercent] = fmt.Sprintf("%.2f", sample.CPUPercent)
	metadata[metricschema.HeartbeatKeyMemoryPercent] = fmt.Sprintf("%.2f", sample.MemoryPercent)
	metadata[metricschema.HeartbeatKeyMemoryUsedPercent] = fmt.Sprintf("%.2f", sample.MemoryPercent)
	metadata[metricschema.HeartbeatKeyDiskPercent] = fmt.Sprintf("%.2f", sample.DiskPercent)
	metadata[metricschema.HeartbeatKeyDiskUsedPercent] = fmt.Sprintf("%.2f", sample.DiskPercent)
	metadata[metricschema.HeartbeatKeyNetworkRXBytesPerSec] = fmt.Sprintf("%.2f", sample.NetRXBytesPerSec)
	metadata[metricschema.HeartbeatKeyNetworkTXBytesPerSec] = fmt.Sprintf("%.2f", sample.NetTXBytesPerSec)
	if sample.TempCelsius != nil {
		metadata[metricschema.HeartbeatKeyTempCelsius] = fmt.Sprintf("%.2f", *sample.TempCelsius)
		metadata[metricschema.HeartbeatKeyTemperatureCelsius] = fmt.Sprintf("%.2f", *sample.TempCelsius)
	}
	resolvedPlatform := resolveHeartbeatPlatform(metadata)
	if resolvedPlatform != "" {
		metadata["platform"] = resolvedPlatform
	}

	payload := assets.HeartbeatRequest{
		AssetID:  identity.assetID,
		Type:     "host",
		Name:     identity.assetID,
		Source:   p.cfg.Source,
		GroupID:  groupID,
		Status:   "online",
		Platform: resolvedPlatform,
		Metadata: metadata,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	endpoint := strings.TrimRight(identity.apiBaseURL, "/") + "/assets/heartbeat"
	req, err := securityruntime.NewOutboundRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+identity.token)

	resp, err := securityruntime.DoOutboundRequest(p.client, req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()
	// Read normal small acknowledgements so Go 1.26 can reuse the connection.
	// A large or stalled response remains bounded by this cap and client timeout.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("heartbeat rejected with status %d", resp.StatusCode)
	}
	return nil
}

type noopHeartbeatPublisher struct{}

func (noopHeartbeatPublisher) Publish(context.Context, TelemetrySample) error {
	return nil
}

func cloneStringMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func resolveHeartbeatPlatform(metadata map[string]string) string {
	return platforms.Resolve(
		metadata["platform"],
		metadata["os"],
		metadata["os_name"],
		metadata["os_pretty_name"],
	)
}

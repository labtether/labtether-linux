package webservice

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	dockerpkg "github.com/labtether/labtether-linux/internal/agentcore/docker"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

type changingAssetTransport struct {
	*recordingCollectorTransport
	assetID string
}

func (t *changingAssetTransport) AssetID() string { return t.assetID }

func TestDockerServicesUseApprovedAssetAfterIdentityChange(t *testing.T) {
	transport := &changingAssetTransport{newRecordingCollectorTransport(true), "old-host"}
	collector := &WebServiceCollector{transport: transport, assetID: "old-host", hostIP: "127.0.0.1"}
	containers := []dockerpkg.DockerContainer{{
		ID: "container-1", Names: []string{"/grafana"}, Image: "grafana/grafana", State: "running",
		Ports: []dockerpkg.DockerPort{{PrivatePort: 3000, PublicPort: 3000, Type: "tcp"}},
	}}
	before := collector.buildServicesFromContainers(containers)
	transport.assetID = "canonical-host"
	after := collector.buildServicesFromContainers(containers)
	if len(before) != 1 || len(after) != 1 {
		t.Fatalf("service counts before=%d after=%d", len(before), len(after))
	}
	if before[0].HostAssetID != "old-host" || after[0].HostAssetID != "canonical-host" || before[0].ID == after[0].ID {
		t.Fatalf("service identity did not change with Hub asset: before=%+v after=%+v", before[0], after[0])
	}
}

func TestApprovedAssetDoesNotReplayOldServicesAfterDiscoveryFailure(t *testing.T) {
	transport := &changingAssetTransport{newRecordingCollectorTransport(true), "canonical-host"}
	collector := NewWebServiceCollector(transport, "old-host", "127.0.0.1", 0, nil,
		WebServiceDiscoveryConfig{ProxyEnabled: true})
	collector.proxyProviders = []ProxyProvider{fakeProxyProvider{
		name: "failed-proxy", detected: true, err: errors.New("unavailable"),
	}}
	collector.lastServices = []agentmgr.DiscoveredWebService{{
		ID: "old-service", HostAssetID: "old-host", Source: "proxy", Name: "old",
	}}
	collector.RunCycle(context.Background())
	msg := waitForCollectorMessage(t, transport.recordingCollectorTransport, time.Second)
	var report agentmgr.WebServiceReportData
	if err := json.Unmarshal(msg.Data, &report); err != nil {
		t.Fatal(err)
	}
	if report.HostAssetID != "canonical-host" || len(report.Services) != 0 {
		t.Fatalf("replayed stale service after approval: host=%q services=%+v", report.HostAssetID, report.Services)
	}
}

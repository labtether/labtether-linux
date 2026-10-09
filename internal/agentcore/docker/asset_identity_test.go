package docker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

type changingAssetTransport struct {
	*recordingCollectorTransport
	assetID string
}

func (t *changingAssetTransport) AssetID() string { return t.assetID }

func TestDockerEventsUseApprovedAssetAfterIdentityChange(t *testing.T) {
	transport := &changingAssetTransport{newRecordingCollectorTransport(true), "old-host"}
	collector := &DockerCollector{transport: transport, assetID: "old-host"}
	collector.forwardEvent(DockerEvent{Type: "container", Action: "start"})
	transport.assetID = "canonical-host"
	collector.forwardEvent(DockerEvent{Type: "container", Action: "stop"})
	for _, want := range []string{"old-host", "canonical-host"} {
		msg := waitForCollectorMessage(t, transport.recordingCollectorTransport, time.Second)
		if msg.Type != agentmgr.MsgDockerEvents {
			t.Fatalf("Docker message type=%q", msg.Type)
		}
		var event agentmgr.DockerEventData
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			t.Fatal(err)
		}
		if event.HostID != want {
			t.Fatalf("Docker host=%q, want %q", event.HostID, want)
		}
	}
}

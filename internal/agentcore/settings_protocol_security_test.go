package agentcore

import (
	"encoding/json"
	"testing"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

type recordingSettingsSender struct{ messages []agentmgr.Message }

func (s *recordingSettingsSender) Send(msg agentmgr.Message) error {
	s.messages = append(s.messages, msg)
	return nil
}

// The TURN password is a local secret. The settings state sent on every
// connect must not carry it to the Hub.
func TestAgentSettingsStateOmitsTURNPassword(t *testing.T) {
	runtime := NewRuntime(RuntimeConfig{
		WebRTCTURNURL:  "turn:turn.example.test:3478",
		WebRTCTURNPass: "must-not-echo",
	}, nil, nil)
	sender := &recordingSettingsSender{}

	sendAgentSettingsState(sender, runtime, "rev-1")

	if len(sender.messages) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sender.messages))
	}
	var state agentmgr.AgentSettingsStateData
	if err := json.Unmarshal(sender.messages[0].Data, &state); err != nil {
		t.Fatalf("decode agent.settings.state: %v", err)
	}
	if _, ok := state.Values[SettingKeyWebRTCTURNPass]; ok {
		t.Fatal("TURN password was sent to the Hub")
	}
	if state.Values[SettingKeyWebRTCTURNURL] != "turn:turn.example.test:3478" {
		t.Fatalf("TURN URL=%q, want it kept", state.Values[SettingKeyWebRTCTURNURL])
	}
}

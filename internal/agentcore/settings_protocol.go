package agentcore

import (
	"encoding/json"
	"log"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

// redactSensitiveAgentSettingValues keeps secrets such as the TURN password
// on the agent. The Hub sees every other effective value.
func redactSensitiveAgentSettingValues(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		if key == SettingKeyWebRTCTURNPass {
			continue
		}
		out[key] = value
	}
	return out
}

func sendAgentSettingsState(transport agentMessageSender, runtime *Runtime, revision string) {
	if transport == nil || runtime == nil {
		return
	}

	if connected, ok := transport.(interface{ Connected() bool }); ok && !connected.Connected() {
		return
	}

	fingerprint := ""
	if runtime.deviceIdentity != nil {
		fingerprint = runtime.deviceIdentity.Fingerprint
	}

	payload := agentmgr.AgentSettingsStateData{
		Revision:             revision,
		Values:               redactSensitiveAgentSettingValues(runtime.ReportedAgentSettings()),
		Fingerprint:          fingerprint,
		AllowRemoteOverrides: runtime.allowRemoteOverrides(),
		ReportedAt:           time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("agentws: failed to marshal agent.settings.state: %v", err)
		return
	}
	if err := transport.Send(agentmgr.Message{
		Type: agentmgr.MsgAgentSettingsState,
		Data: data,
	}); err != nil {
		log.Printf("agentws: failed to send agent.settings.state: %v", err)
	}
}

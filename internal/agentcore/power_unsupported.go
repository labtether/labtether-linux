package agentcore

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

// This build has no reboot or shutdown backend. It answers power.action with
// the same power.result shape the canonical agent sends, status "unsupported",
// so the Hub reports a clear answer instead of waiting for a reply.
const (
	msgPowerAction = "power.action"
	msgPowerResult = "power.result"
)

type powerUnsupportedResult struct {
	RequestID string `json:"request_id"`
	AssetID   string `json:"asset_id"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
}

func handlePowerActionUnsupported(sender agentMessageSender, msg agentmgr.Message, assetID string) {
	requestID := strings.TrimSpace(msg.ID)
	if sender == nil || requestID == "" || len(requestID) > 128 {
		return
	}
	var request struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		return
	}
	if request.Action != "reboot" && request.Action != "shutdown" {
		return
	}
	data, err := json.Marshal(powerUnsupportedResult{
		RequestID: requestID,
		AssetID:   strings.TrimSpace(assetID),
		Action:    request.Action,
		Status:    "unsupported",
		Code:      "unsupported_platform",
		Message:   "power actions are not supported by this agent",
	})
	if err != nil {
		return
	}
	if err := sender.Send(agentmgr.Message{Type: msgPowerResult, ID: requestID, Data: data}); err != nil {
		log.Printf("agentws: failed to send power result: %v", err)
	}
}

package agentcore

import (
	"errors"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

var errEnrollmentPending = errors.New("agent enrollment is pending")

// A tokenless socket stays readable for enrollment control but must not be
// advertised as ready for telemetry or ordinary Hub operations.
func (t *wsTransport) socketOpen() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected
}

func (t *wsTransport) EnrollmentPending() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected && t.pendingEnrollment
}

func inboundMessageAllowed(enrollmentPending bool, messageType string) bool {
	if !enrollmentPending {
		return true
	}
	switch messageType {
	case agentmgr.MsgEnrollmentChallenge, agentmgr.MsgEnrollmentApproved, agentmgr.MsgEnrollmentRejected:
		return true
	default:
		return false
	}
}

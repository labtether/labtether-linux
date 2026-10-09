package agentcore

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/labtether/labtether-linux/pkg/agentidentity"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

// handleEnrollmentChallenge signs the hub-issued enrollment challenge with the
// agent device private key and sends enrollment.proof back to the hub.
func handleEnrollmentChallenge(transport *wsTransport, msg agentmgr.Message, cfg RuntimeConfig) {
	var data agentmgr.EnrollmentChallengeData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("agentws: invalid enrollment.challenge payload: %v", err)
		return
	}

	nonce := strings.TrimSpace(data.Nonce)
	connectionID := strings.TrimSpace(data.ConnectionID)
	if nonce == "" || connectionID == "" {
		log.Printf("agentws: enrollment.challenge missing nonce/connection_id")
		return
	}

	identity := transport.deviceIdentity
	if identity == nil {
		loaded, err := loadDeviceIdentity(cfg)
		if err != nil {
			log.Printf("agentws: could not load device identity for enrollment proof: %v", err)
			return
		}
		identity = loaded
		transport.deviceIdentity = loaded
	}

	signingPayload := agentidentity.BuildEnrollmentProofPayload(connectionID, nonce, identity.Fingerprint)
	signature := ed25519.Sign(identity.PrivateKey, signingPayload)

	proof := agentmgr.EnrollmentProofData{
		ConnectionID: connectionID,
		Nonce:        nonce,
		KeyAlgorithm: identity.KeyAlgorithm,
		PublicKey:    identity.PublicKeyBase64,
		Fingerprint:  identity.Fingerprint,
		Signature:    base64.StdEncoding.EncodeToString(signature),
	}
	rawProof, err := json.Marshal(proof)
	if err != nil {
		log.Printf("agentws: failed to marshal enrollment proof: %v", err)
		return
	}

	if err := transport.Send(agentmgr.Message{
		Type: agentmgr.MsgEnrollmentProof,
		Data: rawProof,
	}); err != nil {
		log.Printf("agentws: failed to send enrollment proof: %v", err)
		return
	}

	log.Printf("agentws: enrollment proof sent for connection_id=%s", connectionID)
}

// handleEnrollmentApproved processes an enrollment.approved message from the hub.
// It saves the token to disk, updates the transport credentials, and closes the
// current (unauthenticated) connection so the reconnect loop re-dials with the token.
func handleEnrollmentApproved(transport *wsTransport, msg agentmgr.Message, cfg RuntimeConfig) {
	var data agentmgr.EnrollmentApprovedData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("agentws: invalid enrollment.approved payload: %v", err)
		return
	}
	data.Token = strings.TrimSpace(data.Token)
	data.AssetID = strings.TrimSpace(data.AssetID)
	if err := validateIssuedAgentToken(data.Token); err != nil || !validEnrollmentAssetID(data.AssetID) {
		log.Printf("agentws: enrollment.approved received invalid credential or canonical asset id")
		return
	}

	log.Printf("agentws: enrollment APPROVED! asset_id=%s", data.AssetID)
	current := transport.identitySnapshot()
	unplacedGroupID := ""
	adopted, err := transport.adoptCredential(data.Token, data.AssetID, current.wsBaseURL, current.apiBaseURL, &unplacedGroupID)
	if err != nil {
		log.Printf("agentws: enrollment.approved identity update failed: %v", err)
		return
	}
	// Close the unauthenticated socket before any heartbeat can use the new ID.
	// The reconnect loop uses the approved identity already held in memory.
	transport.markDisconnected()

	// Persist token to disk so it survives restarts.
	if cfg.TokenFilePath != "" {
		if err := saveTokenToFile(cfg.TokenFilePath, data.Token); err != nil {
			log.Printf("agentws: approved token is memory-only: %v", err)
			if removeErr := os.Remove(cfg.TokenFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
				log.Printf("agentws: warning: could not remove stale token: %v", removeErr)
			}
		} else {
			log.Printf("agentws: token saved to %s", cfg.TokenFilePath)
			if err := saveEnrollmentState(cfg.TokenFilePath, enrollmentState{
				AssetID: adopted.assetID, GroupID: canonicalGroupIDPointer(adopted.groupID, adopted.groupKnown),
				HubWSURL: adopted.wsBaseURL, HubAPIURL: adopted.apiBaseURL,
			}); err != nil {
				log.Printf("agentws: warning: failed to save enrollment state: %v", err)
			}
		}
	}

}

// handleEnrollmentRejected processes an enrollment.rejected message from the hub.
// The agent keeps the connection open and will retry; the admin may still approve later.
func handleEnrollmentRejected(msg agentmgr.Message) {
	var data agentmgr.EnrollmentRejectedData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("agentws: invalid enrollment.rejected payload: %v", err)
		return
	}
	log.Printf("agentws: enrollment REJECTED: %s", data.Reason)
	// Do not disconnect - the admin might approve a pending request later via a different
	// approval flow, or the operator may re-enroll with a different token.
}

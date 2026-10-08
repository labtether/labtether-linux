package agentmgr

import (
	"encoding/json"
)

// Wire protocol message types.
const (
	MsgHeartbeat            = "heartbeat"
	MsgTelemetry            = "telemetry"
	MsgCommandRequest       = "command.request"
	MsgCommandResult        = "command.result"
	MsgPing                 = "ping"
	MsgPong                 = "pong"
	MsgLogStream            = "log.stream"
	MsgLogBatch             = "log.batch"
	MsgJournalQuery         = "journal.query"   // Hub → Agent: historical journalctl query
	MsgJournalEntries       = "journal.entries" // Agent → Hub: historical journalctl query response
	MsgConfigUpdate         = "config.update"
	MsgConfigApplied        = "config.applied"
	MsgAgentSettingsApply   = "agent.settings.apply"
	MsgAgentSettingsApplied = "agent.settings.applied"
	MsgAgentSettingsState   = "agent.settings.state"
	MsgUpdateRequest        = "update.request"
	MsgUpdateProgress       = "update.progress"
	MsgUpdateResult         = "update.result"

	// Terminal session messages (interactive PTY over agent WebSocket).
	MsgTerminalProbe    = "terminal.probe"     // Hub → Agent: probe tmux availability
	MsgTerminalProbed   = "terminal.probed"    // Agent → Hub: tmux probe result
	MsgTerminalStart    = "terminal.start"     // Hub → Agent: start a PTY shell
	MsgTerminalStarted  = "terminal.started"   // Agent → Hub: shell ready
	MsgTerminalData     = "terminal.data"      // Bidirectional: stdin/stdout chunks (base64)
	MsgTerminalResize   = "terminal.resize"    // Hub → Agent: change PTY window size
	MsgTerminalTmuxKill = "terminal.tmux.kill" // Hub → Agent: end a saved tmux session by stable name
	MsgTerminalClose    = "terminal.close"     // Hub → Agent: end terminal session
	MsgTerminalClosed   = "terminal.closed"    // Agent → Hub: session ended

	// SSH key auto-provisioning messages.
	MsgSSHKeyInstall   = "ssh_key.install"   // Hub → Agent: install public key
	MsgSSHKeyInstalled = "ssh_key.installed" // Agent → Hub: key installed confirmation
	MsgSSHKeyRemove    = "ssh_key.remove"    // Hub → Agent: remove public key
	MsgSSHKeyRemoved   = "ssh_key.removed"   // Agent → Hub: key removed confirmation

	// Desktop (VNC) session messages.
	MsgDesktopStart        = "desktop.start"         // Hub → Agent: start VNC + bridge
	MsgDesktopStarted      = "desktop.started"       // Agent → Hub: VNC ready
	MsgDesktopData         = "desktop.data"          // Bidirectional: base64 VNC bytes
	MsgDesktopClose        = "desktop.close"         // Hub → Agent: end desktop session
	MsgDesktopClosed       = "desktop.closed"        // Agent → Hub: session ended
	MsgDesktopListDisplays = "desktop.list-displays" // Hub → Agent: enumerate displays
	MsgDesktopDisplays     = "desktop.displays"      // Agent → Hub: display enumeration response
	MsgDesktopDiagnose     = "desktop.diagnose"      // Hub → Agent: request desktop stack diagnostic
	MsgDesktopDiagnosed    = "desktop.diagnosed"     // Agent → Hub: desktop stack diagnostic response

	// WebRTC streaming messages.
	MsgWebRTCCapabilities = "webrtc.capabilities" // Agent → Hub: available encoders, audio sources
	MsgWebRTCOffer        = "webrtc.offer"        // Hub → Agent: SDP offer from browser
	MsgWebRTCAnswer       = "webrtc.answer"       // Agent → Hub: SDP answer
	MsgWebRTCICE          = "webrtc.ice"          // Bidirectional: ICE candidate
	MsgWebRTCStart        = "webrtc.start"        // Hub → Agent: start WebRTC session
	MsgWebRTCStarted      = "webrtc.started"      // Agent → Hub: WebRTC session ready
	MsgWebRTCStop         = "webrtc.stop"         // Hub → Agent: stop WebRTC session
	MsgWebRTCStopped      = "webrtc.stopped"      // Agent → Hub: WebRTC session ended
	MsgWebRTCInput        = "webrtc.input"        // Hub → Agent: keyboard/mouse input (fallback)

	// Clipboard sync messages.
	MsgClipboardGet    = "clipboard.get"     // Hub → Agent: read remote clipboard
	MsgClipboardData   = "clipboard.data"    // Agent → Hub: clipboard contents
	MsgClipboardSet    = "clipboard.set"     // Hub → Agent: write to remote clipboard
	MsgClipboardSetAck = "clipboard.set_ack" // Agent → Hub: write confirmation

	// Desktop audio sideband messages (VNC sessions).
	MsgDesktopAudioStart = "desktop.audio.start" // Hub → Agent: start audio capture
	MsgDesktopAudioStop  = "desktop.audio.stop"  // Hub → Agent: stop audio capture
	MsgDesktopAudioData  = "desktop.audio.data"  // Agent → Hub: Opus audio frame
	MsgDesktopAudioState = "desktop.audio.state" // Agent → Hub: state change

	// Wake-on-LAN messages.
	MsgWoLSend   = "wol.send"   // Hub → Agent: send a WoL packet on local network
	MsgWoLResult = "wol.result" // Agent → Hub: WoL send result

	// File transfer messages.
	MsgFileList         = "file.list"          // Hub → Agent: list directory
	MsgFileListed       = "file.listed"        // Agent → Hub: directory listing result
	MsgFileRead         = "file.read"          // Hub → Agent: read file
	MsgFileData         = "file.data"          // Agent → Hub: file data chunk (base64)
	MsgFileWrite        = "file.write"         // Hub → Agent: write file chunk (base64)
	MsgFileWritten      = "file.written"       // Agent → Hub: write confirmation
	MsgFileMkdir        = "file.mkdir"         // Hub → Agent: create directory
	MsgFileDelete       = "file.delete"        // Hub → Agent: delete file/directory
	MsgFileRename       = "file.rename"        // Hub → Agent: rename/move file or directory
	MsgFileCopy         = "file.copy"          // Hub → Agent: copy file/directory
	MsgFileResult       = "file.result"        // Agent → Hub: operation result (mkdir, delete, rename, copy)
	MsgFileSearch       = "file.search"        // Hub → Agent: recursive filename search
	MsgFileSearchResult = "file.search_result" // Agent → Hub: filename search results

	// Process list messages.
	MsgProcessList       = "process.list"        // Hub → Agent: request process list
	MsgProcessListed     = "process.listed"      // Agent → Hub: process list response
	MsgProcessKill       = "process.kill"        // Hub → Agent: send signal to a process
	MsgProcessKillResult = "process.kill_result" // Agent → Hub: signal delivery result

	// Service management messages.
	MsgServiceList   = "service.list"   // Hub → Agent: request service list
	MsgServiceListed = "service.listed" // Agent → Hub: service list response
	MsgServiceAction = "service.action" // Hub → Agent: perform service action
	MsgServiceResult = "service.result" // Agent → Hub: service action result

	// Disk/mount info messages.
	MsgDiskList   = "disk.list"   // Hub → Agent: request mount/disk info
	MsgDiskListed = "disk.listed" // Agent → Hub: mount/disk info response

	// Network interface messages.
	MsgNetworkList   = "network.list"   // Hub → Agent: request network interface info
	MsgNetworkListed = "network.listed" // Agent → Hub: network interface response
	MsgNetworkAction = "network.action" // Hub → Agent: perform network action (apply/rollback)
	MsgNetworkResult = "network.result" // Agent → Hub: network action result

	// Package inventory messages.
	MsgPackageList   = "package.list"   // Hub → Agent: request installed packages
	MsgPackageListed = "package.listed" // Agent → Hub: installed packages response
	MsgPackageAction = "package.action" // Hub → Agent: run package action (install/remove/upgrade)
	MsgPackageResult = "package.result" // Agent → Hub: package action result

	// Cron/timer visibility messages.
	MsgCronList   = "cron.list"   // Hub → Agent: request cron/timer entries
	MsgCronListed = "cron.listed" // Agent → Hub: cron/timer entries response

	// User session messages.
	MsgUsersList   = "users.list"   // Hub → Agent: request active user sessions
	MsgUsersListed = "users.listed" // Agent → Hub: active user sessions response

	// Alert notification messages.
	MsgAlertNotify = "alert.notify" // Hub → Agent: push alert for local display

	// Enrollment approval messages.
	MsgEnrollmentChallenge = "enrollment.challenge" // Hub → Agent: prove device-key possession
	MsgEnrollmentProof     = "enrollment.proof"     // Agent → Hub: signed challenge proof
	MsgEnrollmentApproved  = "enrollment.approved"  // Hub → Agent: enrollment approved, token attached
	MsgEnrollmentRejected  = "enrollment.rejected"  // Hub → Agent: enrollment rejected

	// Docker container management messages.
	MsgDockerDiscovery      = "docker.discovery"       // Agent → Hub: full inventory
	MsgDockerDiscoveryDelta = "docker.discovery.delta" // Agent → Hub: incremental inventory update
	MsgDockerStats          = "docker.stats"           // Agent → Hub: container stats
	MsgDockerEvents         = "docker.events"          // Agent → Hub: daemon events
	MsgDockerAction         = "docker.action"          // Hub → Agent: lifecycle command
	MsgDockerActionResult   = "docker.action.result"   // Agent → Hub: action result
	MsgDockerLogsStart      = "docker.logs.start"      // Hub → Agent: start log stream
	MsgDockerLogsStop       = "docker.logs.stop"       // Hub → Agent: stop log stream
	MsgDockerLogsStream     = "docker.logs.stream"     // Agent → Hub: log lines
	MsgDockerExecStart      = "docker.exec.start"      // Hub → Agent: start exec session
	MsgDockerExecStarted    = "docker.exec.started"    // Agent → Hub: exec ready
	MsgDockerExecData       = "docker.exec.data"       // Agent → Hub: exec output
	MsgDockerExecInput      = "docker.exec.input"      // Hub → Agent: exec stdin
	MsgDockerExecResize     = "docker.exec.resize"     // Hub → Agent: resize exec PTY
	MsgDockerExecClose      = "docker.exec.close"      // Hub → Agent: close exec
	MsgDockerExecClosed     = "docker.exec.closed"     // Agent → Hub: exec ended
	MsgDockerComposeAction  = "docker.compose.action"  // Hub → Agent: stack operation
	MsgDockerComposeResult  = "docker.compose.result"  // Agent → Hub: stack result

	// Web service discovery messages.
	MsgWebServiceReport = "webservice.report" // Agent → Hub: discovered services
	MsgWebServiceSync   = "webservice.sync"   // Hub → Agent: request immediate rediscovery
)

// KnownMessageTypes is the set of all valid agent protocol message types.
var KnownMessageTypes = map[string]bool{
	MsgHeartbeat: true, MsgTelemetry: true,
	MsgCommandRequest: true, MsgCommandResult: true,
	MsgPing: true, MsgPong: true,
	MsgLogStream: true, MsgLogBatch: true,
	MsgJournalQuery: true, MsgJournalEntries: true,
	MsgConfigUpdate: true, MsgConfigApplied: true,
	MsgAgentSettingsApply: true, MsgAgentSettingsApplied: true, MsgAgentSettingsState: true,
	MsgUpdateRequest: true, MsgUpdateProgress: true, MsgUpdateResult: true,
	MsgTerminalProbe: true, MsgTerminalProbed: true,
	MsgTerminalStart: true, MsgTerminalStarted: true, MsgTerminalData: true,
	MsgTerminalResize: true, MsgTerminalTmuxKill: true, MsgTerminalClose: true, MsgTerminalClosed: true,
	MsgSSHKeyInstall: true, MsgSSHKeyInstalled: true,
	MsgSSHKeyRemove: true, MsgSSHKeyRemoved: true,
	MsgDesktopStart: true, MsgDesktopStarted: true, MsgDesktopData: true,
	MsgDesktopClose: true, MsgDesktopClosed: true,
	MsgDesktopListDisplays: true, MsgDesktopDisplays: true,
	MsgDesktopDiagnose: true, MsgDesktopDiagnosed: true,
	MsgWebRTCCapabilities: true, MsgWebRTCOffer: true, MsgWebRTCAnswer: true,
	MsgWebRTCICE: true, MsgWebRTCStart: true, MsgWebRTCStarted: true,
	MsgWebRTCStop: true, MsgWebRTCStopped: true, MsgWebRTCInput: true,
	MsgClipboardGet: true, MsgClipboardData: true, MsgClipboardSet: true, MsgClipboardSetAck: true,
	MsgDesktopAudioStart: true, MsgDesktopAudioStop: true, MsgDesktopAudioData: true, MsgDesktopAudioState: true,
	MsgWoLSend: true, MsgWoLResult: true,
	MsgFileList: true, MsgFileListed: true,
	MsgFileRead: true, MsgFileData: true,
	MsgFileWrite: true, MsgFileWritten: true,
	MsgFileMkdir: true, MsgFileDelete: true, MsgFileRename: true, MsgFileCopy: true, MsgFileResult: true,
	MsgFileSearch: true, MsgFileSearchResult: true,
	MsgProcessList: true, MsgProcessListed: true,
	MsgProcessKill: true, MsgProcessKillResult: true,
	MsgServiceList: true, MsgServiceListed: true, MsgServiceAction: true, MsgServiceResult: true,
	MsgDiskList: true, MsgDiskListed: true,
	MsgNetworkList: true, MsgNetworkListed: true, MsgNetworkAction: true, MsgNetworkResult: true,
	MsgPackageList: true, MsgPackageListed: true, MsgPackageAction: true, MsgPackageResult: true,
	MsgCronList: true, MsgCronListed: true,
	MsgUsersList: true, MsgUsersListed: true,
	MsgAlertNotify:         true,
	MsgEnrollmentChallenge: true, MsgEnrollmentProof: true,
	MsgEnrollmentApproved: true, MsgEnrollmentRejected: true,
	MsgDockerDiscovery: true, MsgDockerDiscoveryDelta: true, MsgDockerStats: true, MsgDockerEvents: true,
	MsgDockerAction: true, MsgDockerActionResult: true,
	MsgDockerLogsStart: true, MsgDockerLogsStop: true, MsgDockerLogsStream: true,
	MsgDockerExecStart: true, MsgDockerExecStarted: true, MsgDockerExecData: true,
	MsgDockerExecInput: true, MsgDockerExecResize: true,
	MsgDockerExecClose: true, MsgDockerExecClosed: true,
	MsgDockerComposeAction: true, MsgDockerComposeResult: true,
	MsgWebServiceReport: true, MsgWebServiceSync: true,
}

// IsKnownMessageType returns true if the message type is a valid protocol message.
func IsKnownMessageType(msgType string) bool {
	return KnownMessageTypes[msgType]
}

// Message is the envelope for all agent ↔ hub WebSocket communication.
type Message struct {
	Type string          `json:"type"`
	ID   string          `json:"id,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// HeartbeatData mirrors assets.HeartbeatRequest fields for WebSocket transport.
type HeartbeatData struct {
	AssetID      string            `json:"asset_id"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Source       string            `json:"source"`
	GroupID      string            `json:"group_id,omitempty"`
	Status       string            `json:"status,omitempty"`
	Platform     string            `json:"platform,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"` // e.g. ["webrtc", "desktop", "terminal", "files"]
	Connectors   []ConnectorInfo   `json:"connectors,omitempty"`
}

// ConnectorInfo describes a locally discovered connector endpoint.
type ConnectorInfo struct {
	Type      string `json:"type"`
	Endpoint  string `json:"endpoint"`
	Reachable bool   `json:"reachable"`
}

// TelemetryData carries a telemetry sample over WebSocket.
type TelemetryData struct {
	AssetID          string   `json:"asset_id"`
	CPUPercent       float64  `json:"cpu_percent"`
	MemoryPercent    float64  `json:"memory_percent"`
	DiskPercent      float64  `json:"disk_percent"`
	NetRXBytesPerSec float64  `json:"net_rx_bytes_per_sec"`
	NetTXBytesPerSec float64  `json:"net_tx_bytes_per_sec"`
	TempCelsius      *float64 `json:"temp_celsius,omitempty"`
}

// CommandRequestData is sent from hub to agent to execute a command.
type CommandRequestData struct {
	JobID     string `json:"job_id"`
	SessionID string `json:"session_id"`
	CommandID string `json:"command_id"`
	Command   string `json:"command"`
	Timeout   int    `json:"timeout"`
}

// CommandResultData is sent from agent to hub with the command result.
type CommandResultData struct {
	JobID     string `json:"job_id"`
	SessionID string `json:"session_id"`
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Output    string `json:"output"`
}

// LogStreamData carries a single log entry from agent to hub.
type LogStreamData struct {
	AssetID   string `json:"asset_id"`
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

// LogBatchData carries multiple buffered log entries from agent to hub.
type LogBatchData struct {
	AssetID string          `json:"asset_id"`
	Entries []LogStreamData `json:"entries"`
}

// JournalQueryData is sent from hub to agent to query historical journal entries.
type JournalQueryData struct {
	RequestID string `json:"request_id"`
	Since     string `json:"since,omitempty"`    // e.g. "1h ago", "2026-02-24 08:00:00"
	Until     string `json:"until,omitempty"`    // e.g. "now", "2026-02-24 09:00:00"
	Unit      string `json:"unit,omitempty"`     // systemd unit name (e.g. "ssh.service")
	Priority  string `json:"priority,omitempty"` // debug|info|notice|warning|err|crit|alert|emerg
	Search    string `json:"search,omitempty"`   // free-text grep filter
	Limit     int    `json:"limit"`              // max entries
}

// JournalEntriesData is sent from agent to hub with historical journal entries.
type JournalEntriesData struct {
	RequestID string          `json:"request_id"`
	Entries   []LogStreamData `json:"entries"`
	Error     string          `json:"error,omitempty"`
}

// ConfigUpdateData is sent from hub to agent to update runtime configuration.
type ConfigUpdateData struct {
	CollectIntervalSec   *int    `json:"collect_interval_sec,omitempty"`
	HeartbeatIntervalSec *int    `json:"heartbeat_interval_sec,omitempty"`
	LogLevel             *string `json:"log_level,omitempty"`
}

// ConfigAppliedData is sent from agent to hub confirming config application.
type ConfigAppliedData struct {
	CollectIntervalSec   int `json:"collect_interval_sec"`
	HeartbeatIntervalSec int `json:"heartbeat_interval_sec"`
}

// AgentSettingsApplyData is sent from hub to agent to apply settings.
type AgentSettingsApplyData struct {
	RequestID           string            `json:"request_id,omitempty"`
	Revision            string            `json:"revision,omitempty"`
	Values              map[string]string `json:"values"`
	ExpectedFingerprint string            `json:"expected_fingerprint,omitempty"`
}

// AgentSettingsAppliedData is sent from agent to hub after settings apply attempt.
type AgentSettingsAppliedData struct {
	RequestID       string            `json:"request_id,omitempty"`
	Revision        string            `json:"revision,omitempty"`
	Applied         bool              `json:"applied"`
	RestartRequired bool              `json:"restart_required,omitempty"`
	Error           string            `json:"error,omitempty"`
	AppliedValues   map[string]string `json:"applied_values,omitempty"`
	Fingerprint     string            `json:"fingerprint,omitempty"`
	AppliedAt       string            `json:"applied_at,omitempty"`
}

// AgentSettingsStateData is sent from agent to hub to report current effective settings.
type AgentSettingsStateData struct {
	Revision             string            `json:"revision,omitempty"`
	Values               map[string]string `json:"values,omitempty"`
	Fingerprint          string            `json:"fingerprint,omitempty"`
	AllowRemoteOverrides bool              `json:"allow_remote_overrides"`
	ReportedAt           string            `json:"reported_at,omitempty"`
}

// UpdateRequestData is sent from hub to agent to request a system update.
type UpdateRequestData struct {
	JobID    string   `json:"job_id"`
	Mode     string   `json:"mode"`
	Force    bool     `json:"force,omitempty"`
	Packages []string `json:"packages,omitempty"`
}

// UpdateProgressData is sent from agent to hub with update progress.
type UpdateProgressData struct {
	JobID   string `json:"job_id"`
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// UpdateResultData is sent from agent to hub with the final update result.
type UpdateResultData struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// AlertNotifyData is sent from the hub to the agent when an alert fires or resolves.
type AlertNotifyData struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"` // critical, high, medium, low
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	State     string `json:"state"` // firing, resolved
	Timestamp string `json:"timestamp"`
}

// EnrollmentApprovedData is sent from hub to agent when enrollment is approved.
type EnrollmentApprovedData struct {
	Token   string `json:"token"`
	AssetID string `json:"asset_id"`
}

// EnrollmentChallengeData is sent from hub to agent while pending enrollment.
// The agent signs the challenge and returns EnrollmentProofData.
type EnrollmentChallengeData struct {
	ConnectionID string `json:"connection_id"`
	Nonce        string `json:"nonce"`
	ExpiresAt    string `json:"expires_at,omitempty"`
}

// EnrollmentProofData is sent from agent to hub to prove possession of the
// device private key for pending enrollment verification.
type EnrollmentProofData struct {
	ConnectionID string `json:"connection_id"`
	Nonce        string `json:"nonce"`
	KeyAlgorithm string `json:"key_algorithm"`
	PublicKey    string `json:"public_key"`  // base64 raw public key bytes
	Fingerprint  string `json:"fingerprint"` // human-friendly fingerprint
	Signature    string `json:"signature"`   // base64 signature over canonical challenge payload
}

// EnrollmentRejectedData is sent from hub to agent when enrollment is rejected.
type EnrollmentRejectedData struct {
	Reason string `json:"reason,omitempty"`
}

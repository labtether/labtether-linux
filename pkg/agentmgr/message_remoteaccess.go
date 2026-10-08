package agentmgr

// TerminalProbeResponse is sent from agent to hub with tmux availability info.
type TerminalProbeResponse struct {
	HasTmux  bool   `json:"has_tmux"`
	TmuxPath string `json:"tmux_path,omitempty"`
}

// TerminalStartData is sent from hub to agent to start a PTY shell session.
type TerminalStartData struct {
	SessionID   string `json:"session_id"`
	Cols        int    `json:"cols"`
	Rows        int    `json:"rows"`
	Shell       string `json:"shell,omitempty"`        // optional: override default shell
	UseTmux     bool   `json:"use_tmux,omitempty"`     // if true, start inside tmux
	TmuxSession string `json:"tmux_session,omitempty"` // tmux session name to create/attach
}

// TerminalStartedData is sent from agent to hub when the PTY shell is ready.
type TerminalStartedData struct {
	SessionID    string `json:"session_id"`
	TmuxAttached bool   `json:"tmux_attached,omitempty"` // true if running inside tmux
}

// TerminalDataPayload carries terminal I/O data (base64-encoded bytes).
type TerminalDataPayload struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"` // base64-encoded terminal bytes
}

// TerminalResizeData is sent from hub to agent to resize the PTY window.
type TerminalResizeData struct {
	SessionID string `json:"session_id"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

// TerminalTmuxKillData is sent from hub to agent to end a saved tmux session.
type TerminalTmuxKillData struct {
	JobID       string `json:"job_id"`
	SessionID   string `json:"session_id,omitempty"`
	CommandID   string `json:"command_id,omitempty"`
	TmuxSession string `json:"tmux_session"`
	Timeout     int    `json:"timeout,omitempty"`
}

// TerminalCloseData is sent to close a terminal session.
type TerminalCloseData struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// SSHKeyInstallData is sent from hub to agent to install the hub's public key.
type SSHKeyInstallData struct {
	PublicKey  string `json:"public_key"`
	TargetUser string `json:"target_user,omitempty"`
}

// SSHKeyRemoveData is sent from hub to agent to remove the hub's public key.
// Same shape as SSHKeyInstallData — contains the public key to match and remove.
type SSHKeyRemoveData = SSHKeyInstallData

// SSHKeyInstalledData is sent from agent to hub confirming key installation.
type SSHKeyInstalledData struct {
	Username string `json:"username"`
	Hostname string `json:"hostname"`
	HomeDir  string `json:"home_dir"`
}

// DesktopStartData is sent from hub to agent to start a VNC session.
type DesktopStartData struct {
	SessionID   string `json:"session_id"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Quality     string `json:"quality,omitempty"` // low, medium, high
	Display     string `json:"display,omitempty"` // e.g. ":0"
	VNCPassword string `json:"vnc_password,omitempty"`
}

// DesktopStartedData is sent from agent to hub when VNC is ready.
type DesktopStartedData struct {
	SessionID string `json:"session_id"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
}

// DesktopDataPayload carries VNC protocol bytes (base64-encoded).
type DesktopDataPayload struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"` // base64-encoded VNC bytes
}

// DesktopCloseData is sent to close a desktop session.
type DesktopCloseData struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// WebRTCCapabilitiesData describes the agent's WebRTC streaming capabilities.
type WebRTCCapabilitiesData struct {
	Available                  bool     `json:"available"`
	UnavailableReason          string   `json:"unavailable_reason,omitempty"`   // e.g. "unsupported_platform:darwin", "gst_launch_not_found"
	VideoEncoders              []string `json:"video_encoders,omitempty"`       // e.g. ["nvenc_h264", "vaapi_h264", "vp8", "x264"]
	AudioSources               []string `json:"audio_sources,omitempty"`        // e.g. ["pulseaudio", "pipewire"]
	Displays                   []string `json:"displays,omitempty"`             // available X11 displays
	DesktopSessionType         string   `json:"desktop_session_type,omitempty"` // x11, wayland, headless
	DesktopBackend             string   `json:"desktop_backend,omitempty"`      // x11, wayland_pipewire, headless
	CaptureBackend             string   `json:"capture_backend,omitempty"`      // ximagesrc, pipewiresrc
	VNCRealDesktopSupported    bool     `json:"vnc_real_desktop_supported"`
	WebRTCRealDesktopSupported bool     `json:"webrtc_real_desktop_supported"`
}

// WebRTCSessionData is sent with webrtc.start.
type WebRTCSessionData struct {
	SessionID    string `json:"session_id"`
	Display      string `json:"display,omitempty"`
	Quality      string `json:"quality,omitempty"` // low, medium, high
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	FPS          int    `json:"fps,omitempty"`
	AudioEnabled bool   `json:"audio_enabled"`
}

// WebRTCSDPData carries SDP offer/answer payloads.
type WebRTCSDPData struct {
	SessionID string `json:"session_id"`
	Type      string `json:"type"` // "offer" or "answer"
	SDP       string `json:"sdp"`
}

// WebRTCICEData carries an ICE candidate.
type WebRTCICEData struct {
	SessionID     string `json:"session_id"`
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdp_mid,omitempty"`
	SDPMLineIndex *int   `json:"sdp_mline_index,omitempty"`
}

// WebRTCStartedData confirms WebRTC session is ready for signaling.
type WebRTCStartedData struct {
	SessionID    string `json:"session_id"`
	VideoEncoder string `json:"video_encoder"`
	AudioSource  string `json:"audio_source,omitempty"`
}

// WebRTCStoppedData confirms WebRTC session teardown.
type WebRTCStoppedData struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// WebRTCInputData carries keyboard/mouse events over signaling (fallback for DataChannel).
type WebRTCInputData struct {
	SessionID string `json:"session_id"`
	Type      string `json:"type"` // keydown, keyup, mousemove, mousedown, mouseup
	KeyCode   int    `json:"key_code,omitempty"`
	Code      string `json:"code,omitempty"`
	Key       string `json:"key,omitempty"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	Button    int    `json:"button,omitempty"`
	DeltaY    int    `json:"delta_y,omitempty"`
}

// ClipboardGetData requests the agent's clipboard contents.
type ClipboardGetData struct {
	RequestID string `json:"request_id"`
	Format    string `json:"format"` // "text" or "image"
}

// ClipboardDataPayload carries clipboard contents from agent to hub.
type ClipboardDataPayload struct {
	RequestID string `json:"request_id"`
	Format    string `json:"format"`         // "text" or "image/png"
	Text      string `json:"text,omitempty"` // plaintext content
	Data      string `json:"data,omitempty"` // base64-encoded binary (images)
	Error     string `json:"error,omitempty"`
}

// ClipboardSetData writes content to the agent's clipboard.
type ClipboardSetData struct {
	RequestID string `json:"request_id"`
	Format    string `json:"format"` // "text" or "image/png"
	Text      string `json:"text,omitempty"`
	Data      string `json:"data,omitempty"` // base64-encoded binary
}

// ClipboardSetAckData confirms clipboard write.
type ClipboardSetAckData struct {
	RequestID string `json:"request_id"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
}

// DesktopAudioStartData requests audio capture start for a VNC session.
type DesktopAudioStartData struct {
	SessionID string `json:"session_id"`
	Bitrate   int    `json:"bitrate,omitempty"` // bps, default 128000
}

// DesktopAudioStopData requests audio capture stop for a VNC session.
type DesktopAudioStopData struct {
	SessionID string `json:"session_id"`
}

// DesktopAudioDataPayload carries an Opus audio frame from agent to hub.
type DesktopAudioDataPayload struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"`      // base64-encoded Opus frame
	Timestamp int64  `json:"timestamp"` // unix milliseconds
}

// DesktopAudioStateData reports audio capture state changes.
type DesktopAudioStateData struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"` // "started", "stopped", "unavailable"
	Error     string `json:"error,omitempty"`
}

// DisplayInfo describes one monitor/display geometry on a node.
type DisplayInfo struct {
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Primary bool   `json:"primary"`
	OffsetX int    `json:"offset_x"`
	OffsetY int    `json:"offset_y"`
}

// DisplayListData carries the display enumeration response.
type DisplayListData struct {
	RequestID string        `json:"request_id"`
	Displays  []DisplayInfo `json:"displays"`
	Error     string        `json:"error,omitempty"`
}

// DesktopDiagnosticRequest is sent Hub → Agent to request a desktop stack diagnostic.
type DesktopDiagnosticRequest struct {
	RequestID string `json:"request_id"`
}

// DesktopDiagnosticData carries the full desktop stack diagnostic response from the agent.
type DesktopDiagnosticData struct {
	RequestID string `json:"request_id"`

	DesktopSessionType         string `json:"desktop_session_type,omitempty"`
	DesktopBackend             string `json:"desktop_backend,omitempty"`
	DesktopUser                string `json:"desktop_user,omitempty"`
	RealDisplay                string `json:"real_display,omitempty"`
	VNCRealDesktopSupported    bool   `json:"vnc_real_desktop_supported"`
	WebRTCRealDesktopSupported bool   `json:"webrtc_real_desktop_supported"`
	CaptureBackend             string `json:"capture_backend,omitempty"`

	// Xvfb state
	XvfbRunning  bool     `json:"xvfb_running"`
	XvfbDisplays []string `json:"xvfb_displays"`
	XvfbPIDs     []int    `json:"xvfb_pids"`

	// X11 display state
	ActiveDisplays []string `json:"active_displays"`
	EnvDisplay     string   `json:"env_display"`

	// VNC state
	X11VNCRunning bool   `json:"x11vnc_running"`
	X11VNCDisplay string `json:"x11vnc_display"`
	X11VNCPort    int    `json:"x11vnc_port"`

	// Bootstrap shell state
	BootstrapRunning bool `json:"bootstrap_running"`
	XtermAvailable   bool `json:"xterm_available"`

	// WebRTC state
	GstLaunchAvailable  bool     `json:"gst_launch_available"`
	GstInspectAvailable bool     `json:"gst_inspect_available"`
	VideoEncoders       []string `json:"video_encoders"`
	AudioSources        []string `json:"audio_sources"`
	WebRTCAvailable     bool     `json:"webrtc_available"`
	WebRTCReason        string   `json:"webrtc_reason"`

	// Active sessions
	ActiveVNCSessions    int `json:"active_vnc_sessions"`
	ActiveWebRTCSessions int `json:"active_webrtc_sessions"`

	// Framebuffer check
	FramebufferHasContent bool   `json:"framebuffer_has_content"`
	FramebufferError      string `json:"framebuffer_error,omitempty"`
}

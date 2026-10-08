package remoteaccess

import (
	"context"
	"encoding/json"
	"github.com/labtether/labtether-linux/internal/agentcore/files"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"github.com/pion/webrtc/v4"
	"log"
	"os/exec"
	"strings"
	"sync"
)

type WebRTCSession struct {
	sessionID      string
	pc             *webrtc.PeerConnection
	videoTrack     *webrtc.TrackLocalStaticRTP
	audioTrack     *webrtc.TrackLocalStaticRTP
	gstVideoCmd    *exec.Cmd
	gstAudioCmd    *exec.Cmd
	videoLogPath   string
	audioLogPath   string
	videoPort      int
	audioPort      int
	inputCh        chan WebRTCInputEvent
	cancel         context.CancelFunc
	done           chan struct{}
	closeOnce      sync.Once
	ManagedDisplay string          // non-empty if display was acquired from DisplayManager
	xauthPath      string          // Xauthority for ManagedDisplay when one was created
	dispMgr        *DisplayManager // reference for release on close
	desktopBackend string
	sessionInfo    DesktopSessionInfo
	inputBackend   string
}

func (s *WebRTCSession) close(reason string) {
	s.closeOnce.Do(func() {
		log.Printf(
			"webrtc: closing session=%s reason=%s display=%s xauth=%s video_port=%d audio_port=%d",
			s.sessionID,
			strings.TrimSpace(reason),
			ValueOrDash(strings.TrimSpace(s.ManagedDisplay)),
			ValueOrDash(strings.TrimSpace(s.xauthPath)),
			s.videoPort,
			s.audioPort,
		)
		if s.cancel != nil {
			s.cancel()
		}
		if s.gstVideoCmd != nil && s.gstVideoCmd.Process != nil {
			_ = s.gstVideoCmd.Process.Kill()
		}
		if s.gstAudioCmd != nil && s.gstAudioCmd.Process != nil {
			_ = s.gstAudioCmd.Process.Kill()
		}
		if s.pc != nil {
			_ = s.pc.Close()
		}
		if s.ManagedDisplay != "" && s.dispMgr != nil {
			s.dispMgr.release(s.ManagedDisplay)
		}
		RemoveProcessLog(s.videoLogPath)
		RemoveProcessLog(s.audioLogPath)
		close(s.done)
	})
}

type WebRTCManager struct {
	Mu       sync.Mutex
	Sessions map[string]*WebRTCSession
	caps     agentmgr.WebRTCCapabilitiesData
	settings SettingsProvider
	fileMgr  *files.Manager
	dispMgr  *DisplayManager
}

func NewWebRTCManager(caps agentmgr.WebRTCCapabilitiesData, settings SettingsProvider, fileMgr *files.Manager, dispMgr *DisplayManager) *WebRTCManager {
	return &WebRTCManager{
		Sessions: make(map[string]*WebRTCSession),
		caps:     caps,
		settings: settings,
		fileMgr:  fileMgr,
		dispMgr:  dispMgr,
	}
}

func (wm *WebRTCManager) HandleWebRTCStop(msg agentmgr.Message, transport MessageSender) {
	var req agentmgr.WebRTCStoppedData
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		return
	}
	wm.CleanupWithReason(req.SessionID, "stopped by hub")
	SendWebRTCStopped(transport, req.SessionID, "stopped by hub")
}

func (wm *WebRTCManager) Cleanup(sessionID string) {
	wm.CleanupWithReason(sessionID, "cleanup requested")
}

func (wm *WebRTCManager) MarkAudioPipelineStopped(sessionID string) {
	wm.Mu.Lock()
	defer wm.Mu.Unlock()
	sess, ok := wm.Sessions[sessionID]
	if !ok {
		return
	}
	sess.gstAudioCmd = nil
	sess.audioLogPath = ""
	sess.audioPort = 0
}

func (wm *WebRTCManager) CleanupWithReason(sessionID, reason string) {
	wm.Mu.Lock()
	sess, ok := wm.Sessions[sessionID]
	if ok {
		delete(wm.Sessions, sessionID)
	}
	wm.Mu.Unlock()
	if !ok {
		return
	}
	log.Printf("webrtc: cleanup session=%s trigger=%s", sessionID, strings.TrimSpace(reason))
	sess.close(reason)
}

func (wm *WebRTCManager) CloseAll() {
	wm.Mu.Lock()
	sessions := make([]*WebRTCSession, 0, len(wm.Sessions))
	for id, sess := range wm.Sessions {
		sessions = append(sessions, sess)
		delete(wm.Sessions, id)
	}
	wm.Mu.Unlock()
	for _, sess := range sessions {
		sess.close("manager closeAll")
	}
}

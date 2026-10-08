package remoteaccess

import (
	"context"
	"encoding/json"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"log"
	"strings"
	"time"
)

func ResolveWebRTCDisplay(requested string, caps agentmgr.WebRTCCapabilitiesData) string {
	if strings.EqualFold(strings.TrimSpace(caps.DesktopSessionType), DesktopSessionTypeWayland) {
		return ""
	}
	display := strings.TrimSpace(requested)
	if display != "" {
		for _, candidate := range caps.Displays {
			if display == strings.TrimSpace(candidate) {
				return display
			}
		}
		if strings.HasPrefix(display, ":") {
			return display
		}
	}
	for _, candidate := range caps.Displays {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return trimmed
		}
	}
	return ":0"
}

func WebRTCVideoBitrateForQuality(quality string) int {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case "low":
		return 1000
	case "high":
		return 20000
	default:
		return 5000
	}
}

func (wm *WebRTCManager) HandleWebRTCStart(transport MessageSender, msg agentmgr.Message) {
	var req agentmgr.WebRTCSessionData
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("webrtc: invalid start request: %v", err)
		return
	}
	if strings.TrimSpace(req.SessionID) == "" {
		log.Printf("webrtc: missing session_id")
		return
	}

	if !wm.caps.Available {
		SendWebRTCStopped(transport, req.SessionID, "webrtc unavailable")
		return
	}

	wm.Mu.Lock()
	if _, exists := wm.Sessions[req.SessionID]; exists {
		wm.Mu.Unlock()
		return
	}
	wm.Mu.Unlock()

	settings := map[string]string{}
	if wm.settings != nil {
		settings = wm.settings.ReportedAgentSettings()
	}
	webrtcCfg := LoadWebRTCConfig(settings)
	if !webrtcCfg.Enabled {
		SendWebRTCStopped(transport, req.SessionID, "webrtc disabled")
		return
	}
	sessionInfo := DetectDesktopSessionFn()

	encName, gstEncoder := BestVideoEncoder(wm.caps)
	if gstEncoder == "" {
		SendWebRTCStopped(transport, req.SessionID, "no supported encoder")
		return
	}

	audioSource := ""
	if req.AudioEnabled {
		audioSource = BestAudioSource(wm.caps)
	}

	videoPort, err := FindFreeUDPPort()
	if err != nil {
		SendWebRTCStopped(transport, req.SessionID, "failed to allocate video RTP port")
		return
	}
	audioPort := 0
	if audioSource != "" {
		audioPort, err = FindFreeUDPPort()
		if err != nil {
			log.Printf("webrtc: audio port unavailable for %s, continuing without audio", req.SessionID)
			audioSource = ""
		}
	}

	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		SendWebRTCStopped(transport, req.SessionID, "codec registration failed")
		return
	}
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		SendWebRTCStopped(transport, req.SessionID, "interceptor registration failed")
		return
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))

	pc, err := api.NewPeerConnection(webrtc.Configuration{ICEServers: webrtcCfg.iceServers()})
	if err != nil {
		SendWebRTCStopped(transport, req.SessionID, "peer connection init failed")
		return
	}

	videoCodec := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}
	if encName == "vp8" {
		videoCodec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}
	}
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(videoCodec, "video", "labtether-screen")
	if err != nil {
		_ = pc.Close()
		SendWebRTCStopped(transport, req.SessionID, "video track creation failed")
		return
	}
	if _, err := pc.AddTrack(videoTrack); err != nil {
		_ = pc.Close()
		SendWebRTCStopped(transport, req.SessionID, "video track attach failed")
		return
	}

	var audioTrack *webrtc.TrackLocalStaticRTP
	if audioSource != "" {
		audioTrack, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
			"audio", "labtether-audio",
		)
		if err == nil {
			if _, addErr := pc.AddTrack(audioTrack); addErr != nil {
				audioTrack = nil
				audioSource = ""
				audioPort = 0
			}
		} else {
			audioTrack = nil
			audioSource = ""
			audioPort = 0
		}
	}

	// Resolve and acquire the display before session registration so that
	// OnConnectionStateChange cannot fire with an unset ManagedDisplay field.
	display := ResolveWebRTCDisplay(req.Display, wm.caps)
	var acquiredDisplay string // non-empty if acquired from display manager
	var xauthPath string
	var sessDispMgr *DisplayManager
	if sessionInfo.Type != DesktopSessionTypeWayland && !IsDisplayAvailable(display) && wm.dispMgr != nil {
		dynDisplay, dynXAuthPath, acquireErr := wm.dispMgr.acquire()
		if acquireErr != nil {
			_ = pc.Close()
			SendWebRTCStopped(transport, req.SessionID, "no display available: "+acquireErr.Error())
			return
		}
		display = dynDisplay
		acquiredDisplay = dynDisplay
		xauthPath = dynXAuthPath
		sessDispMgr = wm.dispMgr
	}
	if sessionInfo.Type != DesktopSessionTypeWayland {
		if strings.TrimSpace(xauthPath) == "" {
			xauthPath = DiscoverDisplayXAuthorityFn(display)
		}
		WakeX11Display(display, xauthPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sess := &WebRTCSession{
		sessionID:      req.SessionID,
		pc:             pc,
		videoTrack:     videoTrack,
		audioTrack:     audioTrack,
		videoPort:      videoPort,
		audioPort:      audioPort,
		inputCh:        make(chan WebRTCInputEvent, 128),
		cancel:         cancel,
		done:           make(chan struct{}),
		ManagedDisplay: acquiredDisplay,
		xauthPath:      xauthPath,
		dispMgr:        sessDispMgr,
		desktopBackend: strings.TrimSpace(wm.caps.DesktopBackend),
		sessionInfo:    sessionInfo,
		inputBackend:   strings.TrimSpace(strings.ToLower(webrtcCfg.WaylandInputBackend)),
	}
	if sess.inputBackend == "" {
		sess.inputBackend = "auto"
	}

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc == nil {
			return
		}
		switch dc.Label() {
		case "input":
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				evt, err := DecodeWebRTCInputEvent(msg.Data)
				if err != nil {
					return
				}
				select {
				case sess.inputCh <- evt:
				default:
				}
			})
		case "clipboard":
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				wm.handleClipboardDataChannelMessage(dc, msg)
			})
		case "file-transfer":
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				wm.handleFileTransferDataChannelMessage(dc, msg)
			})
		}
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		candidate := c.ToJSON()
		data := agentmgr.WebRTCICEData{
			SessionID: req.SessionID,
			Candidate: candidate.Candidate,
		}
		if candidate.SDPMid != nil {
			data.SDPMid = *candidate.SDPMid
		}
		if candidate.SDPMLineIndex != nil {
			idx := int(*candidate.SDPMLineIndex)
			data.SDPMLineIndex = &idx
		}
		raw, _ := json.Marshal(data)
		sendCandidate := func() {
			_ = transport.Send(agentmgr.Message{Type: agentmgr.MsgWebRTCICE, ID: req.SessionID, Data: raw})
		}
		if delay := ICECandidateSendDelay(candidate.Candidate); delay > 0 {
			go func() {
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					sendCandidate()
				}
			}()
			return
		}
		sendCandidate()
	})

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("webrtc: ice state session=%s state=%s", req.SessionID, state.String())
	})

	pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		log.Printf("webrtc: ice gathering session=%s state=%s", req.SessionID, state.String())
	})

	pc.OnSignalingStateChange(func(state webrtc.SignalingState) {
		log.Printf("webrtc: signaling state session=%s state=%s", req.SessionID, state.String())
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("webrtc: peer state session=%s state=%s", req.SessionID, state.String())
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			wm.CleanupWithReason(req.SessionID, "peer connection "+state.String())
			SendWebRTCStopped(transport, req.SessionID, "peer connection "+state.String())
		}
	})

	wm.Mu.Lock()
	wm.Sessions[req.SessionID] = sess
	wm.Mu.Unlock()
	width := req.Width
	if width <= 0 {
		width = 1920
	}
	height := req.Height
	if height <= 0 {
		height = 1080
	}
	fps := req.FPS
	if fps <= 0 {
		fps = webrtcCfg.FPS
	}
	videoBitrate := WebRTCVideoBitrateForQuality(req.Quality)

	videoPipeline := BuildGStreamerVideoPipeline(GstPipelineConfig{
		display: display,
		encoder: gstEncoder,
		width:   width,
		height:  height,
		fps:     fps,
		bitrate: videoBitrate,
		rtpPort: videoPort,
	})
	if sessionInfo.Type == DesktopSessionTypeWayland {
		videoPipeline = BuildWaylandPipeWireVideoPipeline(webrtcCfg.WaylandPipeWireNodeID, GstPipelineConfig{
			encoder: gstEncoder,
			width:   width,
			height:  height,
			fps:     fps,
			bitrate: videoBitrate,
			rtpPort: videoPort,
		})
	}
	if DesktopDebugEnabled() {
		log.Printf("webrtc-debug: session=%s video_pipeline=%s", req.SessionID, videoPipeline)
	}
	gstVideoCmd, err := NewWebRTCSecurityCommand("gst-launch-1.0", ParsePipelineArgs(videoPipeline)...)
	if err != nil {
		wm.CleanupWithReason(req.SessionID, "gst-launch unavailable")
		SendWebRTCStopped(transport, req.SessionID, "gst-launch unavailable")
		return
	}
	if sessionInfo.Type == DesktopSessionTypeWayland {
		gstVideoCmd.Env = BuildWaylandPipeWireEnv(sessionInfo)
	} else {
		gstVideoCmd.Env = BuildX11ClientEnv(display, xauthPath)
	}
	videoLogPath, err := StartWebRTCPipelineWithLog(gstVideoCmd, "labtether-webrtc-video-*.log")
	if err != nil {
		wm.CleanupWithReason(req.SessionID, "failed to start video pipeline")
		SendWebRTCStopped(transport, req.SessionID, "failed to start video pipeline")
		return
	}
	sess.gstVideoCmd = gstVideoCmd
	sess.videoLogPath = videoLogPath
	go wm.WatchPipelineExit(ctx, req.SessionID, "video", gstVideoCmd, videoLogPath, transport)

	if audioTrack != nil && audioPort > 0 && audioSource != "" {
		audioPipeline := BuildGStreamerAudioPipeline(GstAudioConfig{
			source:  audioSource,
			rtpPort: audioPort,
		})
		gstAudioCmd, audioErr := NewWebRTCSecurityCommand("gst-launch-1.0", ParsePipelineArgs(audioPipeline)...)
		if audioErr == nil {
			if sessionInfo.Type == DesktopSessionTypeWayland {
				gstAudioCmd.Env = BuildWaylandPipeWireEnv(sessionInfo)
			} else {
				gstAudioCmd.Env = BuildX11ClientEnv(display, xauthPath)
			}
			audioLogPath, startErr := StartWebRTCPipelineWithLog(gstAudioCmd, "labtether-webrtc-audio-*.log")
			if startErr == nil {
				sess.gstAudioCmd = gstAudioCmd
				sess.audioLogPath = audioLogPath
				go wm.WatchPipelineExit(ctx, req.SessionID, "audio", gstAudioCmd, audioLogPath, transport)
			} else {
				log.Printf("webrtc: failed to start audio pipeline for %s: %v", req.SessionID, startErr)
			}
		} else {
			log.Printf("webrtc: failed to build audio pipeline command for %s: %v", req.SessionID, audioErr)
		}
	}

	go ReadRTPToTrack(ctx, videoPort, videoTrack)
	if audioTrack != nil && audioPort > 0 {
		go ReadRTPToTrack(ctx, audioPort, audioTrack)
	}
	go InjectInputEvents(ctx, sess.inputCh, display, xauthPath, sess)

	startedData, _ := json.Marshal(agentmgr.WebRTCStartedData{
		SessionID:    req.SessionID,
		VideoEncoder: encName,
		AudioSource:  audioSource,
	})
	_ = transport.Send(agentmgr.Message{Type: agentmgr.MsgWebRTCStarted, ID: req.SessionID, Data: startedData})

	log.Printf("webrtc: session started id=%s encoder=%s audio=%s", req.SessionID, encName, audioSource)
}

package agentcore

import (
	"github.com/gorilla/websocket"
	"github.com/labtether/labtether-linux/internal/agentcore/docker"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"log"
	"sync"
)

// Session owner operations run in socket receipt order, before asynchronous
// process/network startup. Their work has a separate bounded lane so close can
// bypass a blocked ordinary command pool.
func dispatchSessionMessage(transport *wsTransport, conn *websocket.Conn, msg agentmgr.Message, terminals *terminalManager, desktops *desktopManager, execs *docker.DockerExecManager, webrtc *webrtcManager, audio *audioSidebandManager, logs *docker.DockerLogManager, slots chan struct{}, workers *sync.WaitGroup) bool {
	switch msg.Type {
	case agentmgr.MsgWebRTCInput, agentmgr.MsgTerminalResize, agentmgr.MsgDockerExecResize, agentmgr.MsgWebRTCOffer, agentmgr.MsgWebRTCICE, agentmgr.MsgDockerLogsStart, agentmgr.MsgDockerLogsStop, agentmgr.MsgDesktopStart, agentmgr.MsgDesktopData, agentmgr.MsgDesktopClose, agentmgr.MsgTerminalStart, agentmgr.MsgTerminalData, agentmgr.MsgTerminalClose, agentmgr.MsgDockerExecStart, agentmgr.MsgDockerExecInput, agentmgr.MsgDockerExecClose, agentmgr.MsgWebRTCStart, agentmgr.MsgWebRTCStop, agentmgr.MsgDesktopAudioStart, agentmgr.MsgDesktopAudioStop:
	default:
		return false
	}
	transport.withCurrentSessionMessage(conn, func(sender *sessionMessageSender) {
		defer recoverInlineMessage(msg.Type)
		launch := func(name string, start func()) { launchSessionStart(sender, slots, workers, name, start) }
		switch msg.Type {
		case agentmgr.MsgDesktopStart:
			launch("desktop-start", desktops.PrepareDesktopStart(sender, msg))
		case agentmgr.MsgDesktopData:
			desktops.HandleDesktopData(msg)
		case agentmgr.MsgDesktopClose:
			launchSessionClose(workers, desktops.PrepareDesktopClose(msg))
		case agentmgr.MsgTerminalStart:
			launch("terminal-start", terminals.PrepareTerminalStart(sender, msg))
		case agentmgr.MsgTerminalResize:
			terminals.HandleTerminalResize(msg)
		case agentmgr.MsgWebRTCInput:
			if webrtc != nil {
				webrtc.HandleWebRTCInput(msg, sender)
			}
		case agentmgr.MsgDockerExecResize:
			if execs != nil {
				launch("docker-exec-resize", execs.PrepareExecResize(msg))
			}
		case agentmgr.MsgWebRTCOffer:
			if webrtc != nil {
				launch("webrtc-offer", webrtc.PrepareWebRTCOffer(msg, sender))
			}
		case agentmgr.MsgWebRTCICE:
			if webrtc != nil {
				launch("webrtc-ice", webrtc.PrepareWebRTCICE(msg))
			}
		case agentmgr.MsgDockerLogsStart:
			if logs != nil {
				launch("docker-logs-start", logs.PrepareLogsStart(sender.Context(), sender, msg))
			}
		case agentmgr.MsgDockerLogsStop:
			if logs != nil {
				logs.HandleLogsStop(msg)
			}
		case agentmgr.MsgTerminalData:
			terminals.HandleTerminalData(msg)
		case agentmgr.MsgTerminalClose:
			terminals.HandleTerminalClose(msg)
		case agentmgr.MsgDockerExecStart:
			if execs != nil {
				launch("docker-exec-start", execs.PrepareExecStart(sender, msg))
			}
		case agentmgr.MsgDockerExecInput:
			if execs != nil {
				execs.HandleExecInput(msg)
			}
		case agentmgr.MsgDockerExecClose:
			if execs != nil {
				execs.HandleExecClose(msg)
			}
		case agentmgr.MsgWebRTCStart:
			if webrtc != nil {
				launch("webrtc-start", webrtc.PrepareWebRTCStart(sender, msg))
			}
		case agentmgr.MsgWebRTCStop:
			if webrtc != nil {
				launchSessionClose(workers, webrtc.PrepareWebRTCStop(msg, sender))
			}
		case agentmgr.MsgDesktopAudioStart:
			if audio != nil {
				launch("audio-start", audio.PrepareAudioStart(sender, msg))
			}
		case agentmgr.MsgDesktopAudioStop:
			if audio != nil {
				launchSessionClose(workers, audio.PrepareAudioStop(sender, msg))
			}
		}
	})
	return true
}

func launchSessionStart(sender *sessionMessageSender, slots chan struct{}, workers *sync.WaitGroup, name string, start func()) {
	if start == nil {
		return
	}
	claimed := false
	select {
	case slots <- struct{}{}:
		claimed = true
	default:
		// Do not stall the reader or silently leave a reserved start. Disconnect
		// cancels this socket's pending contexts; its closure then performs cleanup.
		sender.Close()
	}
	if workers != nil {
		workers.Add(1)
	}
	go func() {
		if workers != nil {
			defer workers.Done()
		}
		if claimed {
			defer func() { <-slots }()
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("agentws: %s panic: %v", name, recovered)
			}
		}()
		start()
	}()
}

// Preparation claims the owner exactly once; duplicate stops return nil.
// Teardown gets no start semaphore because it must unblock those starters.
func launchSessionClose(workers *sync.WaitGroup, closeSession func()) {
	if closeSession == nil {
		return
	}
	if workers != nil {
		workers.Add(1)
	}
	go func() {
		if workers != nil {
			defer workers.Done()
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("agentws: session-close panic: %v", recovered)
			}
		}()
		closeSession()
	}()
}

// A full general pool faults its originating socket. Keeping the reader free
// lets close/cancel reach sessions whose work might otherwise hold that pool.
func launchConnectionHandler(sender *sessionMessageSender, slots chan struct{}, workers *sync.WaitGroup, name string, handle func()) {
	if sender.Context().Err() != nil {
		return
	}
	select {
	case slots <- struct{}{}:
	default:
		sender.Close()
		return
	}
	if workers != nil {
		workers.Add(1)
	}
	go func() {
		defer func() { <-slots }()
		if workers != nil {
			defer workers.Done()
		}
		if sender.Context().Err() == nil {
			safeHandler(name, handle)
		}
	}()
}

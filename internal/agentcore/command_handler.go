package agentcore

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"github.com/labtether/labtether-linux/internal/agentcore/backends"
	"github.com/labtether/labtether-linux/internal/agentcore/docker"
	"github.com/labtether/labtether-linux/internal/agentcore/files"
	"github.com/labtether/labtether-linux/internal/agentcore/system"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

// defaultCommandTimeout is defined in remoteaccess_aliases.go

// receiveLoop reads incoming messages from the hub over the WebSocket transport and
// dispatches them (primarily command requests, terminal sessions, desktop sessions,
// file operations, process queries, service management, disk queries, network
// queries, package inventory queries, cron/timer visibility queries, user session
// queries, clipboard operations, audio sideband, and Docker management).
func receiveLoop(ctx context.Context, transport *wsTransport, cfg RuntimeConfig, runtime *Runtime,
	termMgr *terminalManager, deskMgr *desktopManager, webrtcMgr *webrtcManager, fileMgr *files.Manager,
	processMgr *system.ProcessManager, serviceMgr *backends.ServiceManager, journalMgr *backends.JournalManager, diskMgr *system.DiskManager, networkMgr *networkManager, packageMgr *backends.PackageManager, cronMgr *backends.CronManager, usersMgr *system.UsersManager,
	clipMgr *clipboardManager, audioMgr *audioSidebandManager,
	dockerCollector *docker.DockerCollector, webServiceCollector *WebServiceCollector, execMgr *docker.DockerExecManager, dockerLogMgr *docker.DockerLogManager) {
	// Display manager must close after both desktop and WebRTC managers (LIFO order).
	if deskMgr.DisplayMgr != nil {
		defer deskMgr.DisplayMgr.CloseAll()
	}
	defer termMgr.CloseAll()
	defer deskMgr.CloseAll()
	if webrtcMgr != nil {
		defer webrtcMgr.CloseAll()
	}
	defer fileMgr.CloseAll()
	if processMgr != nil {
		defer processMgr.CloseAll()
	}
	if serviceMgr != nil {
		defer serviceMgr.CloseAll()
	}
	if journalMgr != nil {
		defer journalMgr.CloseAll()
	}
	if diskMgr != nil {
		defer diskMgr.CloseAll()
	}
	if networkMgr != nil {
		defer networkMgr.CloseAll()
	}
	if packageMgr != nil {
		defer packageMgr.CloseAll()
	}
	if cronMgr != nil {
		defer cronMgr.CloseAll()
	}
	if usersMgr != nil {
		defer usersMgr.CloseAll()
	}
	if clipMgr != nil {
		defer clipMgr.CloseAll()
	}
	if audioMgr != nil {
		defer audioMgr.CloseAll()
	}
	if execMgr != nil {
		defer execMgr.CloseAll()
	}
	if dockerLogMgr != nil {
		defer dockerLogMgr.CloseAll()
	}

	// Semaphore limiting concurrent heavy command handlers to avoid unbounded
	// goroutine growth under load.
	const maxConcurrentHandlers = 20
	sem := make(chan struct{}, maxConcurrentHandlers)
	sessionStarts := make(chan struct{}, 30)
	fileWriteMessages := make(chan orderedFileWrite, 64)
	go runOrderedFileWriteWorker(ctx, fileMgr, fileWriteMessages)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if !transport.socketOpen() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		msg, enrollmentPending, sourceConn, err := transport.receiveWithConnection()
		if err != nil {
			if sourceConn != nil && transport.socketOpen() {
				if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					log.Printf("agentws: hub shutting down, will reconnect immediately")
				} else {
					log.Printf("agentws: receive error: %v", err)
				}
				transport.markDisconnectedIfCurrent(sourceConn)
			}
			continue
		}
		if !inboundMessageAllowed(enrollmentPending, msg.Type) {
			log.Printf("agentws: ignored inbound %q on pending enrollment socket", msg.Type)
			continue
		}

		sender := transport.senderForConnection(sourceConn)
		if sender == nil {
			continue
		}

		if dispatchSessionMessage(transport, sourceConn, msg, termMgr, deskMgr, execMgr, webrtcMgr, audioMgr, dockerLogMgr, sessionStarts, nil) {
			continue
		}

		dispatch := func(name string, handle func()) { launchConnectionHandler(sender, sem, nil, name, handle) }
		// Lightweight handlers run on this reader; a panic must not end it.
		inline := func(handle func()) { safeHandler(msg.Type, handle) }
		// An agent approved over this WebSocket got its token after the loop
		// started, so commands read the credential the transport holds now.
		messageConfig := runtimeConfigWithCurrentIdentity(cfg, transport)

		switch msg.Type {
		case agentmgr.MsgCommandRequest:
			dispatch(msg.Type, func() {
				handleCommandRequest(sender, msg, messageConfig)
			})
		case agentmgr.MsgPing:
			// Lightweight — no semaphore.
			inline(func() { _ = sender.Send(agentmgr.Message{Type: agentmgr.MsgPong}) })
		case agentmgr.MsgConfigUpdate:
			// Lightweight — no semaphore.
			inline(func() { handleConfigUpdate(sender, msg, runtime) })
		case agentmgr.MsgAgentSettingsApply:
			// Lightweight — no semaphore.
			inline(func() { handleAgentSettingsApply(sender, msg, runtime) })
		case agentmgr.MsgUpdateRequest:
			dispatch(msg.Type, func() {
				handleUpdateRequest(sender, msg, messageConfig)
			})
		case agentmgr.MsgTerminalProbe:
			// Lightweight — no semaphore.
			inline(func() { termMgr.HandleTerminalProbe(transport) })
		case agentmgr.MsgTerminalTmuxKill:
			dispatch(msg.Type, func() {
				termMgr.HandleTerminalTmuxKill(sender, msg)
			})
		case agentmgr.MsgSSHKeyInstall:
			dispatch(msg.Type, func() {
				handleSSHKeyInstall(sender, msg)
			})
		case agentmgr.MsgSSHKeyRemove:
			dispatch(msg.Type, func() {
				handleSSHKeyRemove(sender, msg)
			})
		case agentmgr.MsgDesktopListDisplays:
			dispatch(msg.Type, func() {
				handleListDisplays(sender, msg)
			})
		case agentmgr.MsgDesktopDiagnose:
			dispatch(msg.Type, func() {
				handleDesktopDiagnose(sender, msg, deskMgr, webrtcMgr)
			})
		case agentmgr.MsgWoLSend:
			// Lightweight — no semaphore.
			inline(func() { system.HandleWoLSend(sender, msg) })
		case agentmgr.MsgFileList:
			dispatch(msg.Type, func() {
				fileMgr.HandleFileList(sender, msg)
			})
		case agentmgr.MsgFileRead:
			dispatch(msg.Type, func() {
				fileMgr.HandleFileRead(sender, msg)
			})
		case agentmgr.MsgFileWrite:
			// Lightweight — no semaphore.
			if !enqueueOrderedFileWrite(sender.Context(), sender, fileWriteMessages, msg) {
				continue
			}
		case agentmgr.MsgFileMkdir:
			dispatch(msg.Type, func() {
				fileMgr.HandleFileMkdir(sender, msg)
			})
		case agentmgr.MsgFileDelete:
			dispatch(msg.Type, func() {
				fileMgr.HandleFileDelete(sender, msg)
			})
		case agentmgr.MsgFileRename:
			dispatch(msg.Type, func() {
				fileMgr.HandleFileRename(sender, msg)
			})
		case agentmgr.MsgFileCopy:
			dispatch(msg.Type, func() {
				fileMgr.HandleFileCopy(sender, msg)
			})
		case agentmgr.MsgFileSearch:
			dispatch(msg.Type, func() {
				fileMgr.HandleFileSearch(sender, msg)
			})
		case agentmgr.MsgProcessList:
			if processMgr != nil {
				dispatch(msg.Type, func() {
					processMgr.HandleProcessList(sender, msg)
				})
			}
		case agentmgr.MsgProcessKill:
			if processMgr != nil {
				dispatch(msg.Type, func() {
					processMgr.HandleProcessKill(sender, msg)
				})
			}
		case agentmgr.MsgServiceList:
			if serviceMgr != nil {
				dispatch(msg.Type, func() {
					serviceMgr.HandleServiceList(sender, msg)
				})
			}
		case agentmgr.MsgServiceAction:
			if serviceMgr != nil {
				dispatch(msg.Type, func() {
					serviceMgr.HandleServiceAction(sender, msg)
				})
			}
		case agentmgr.MsgJournalQuery:
			if journalMgr != nil {
				dispatch(msg.Type, func() {
					journalMgr.HandleJournalQuery(sender, msg)
				})
			}
		case agentmgr.MsgDiskList:
			if diskMgr != nil {
				dispatch(msg.Type, func() {
					diskMgr.HandleDiskList(sender, msg)
				})
			}
		case agentmgr.MsgNetworkList:
			if networkMgr != nil {
				dispatch(msg.Type, func() {
					networkMgr.HandleNetworkList(sender, msg)
				})
			}
		case agentmgr.MsgNetworkAction:
			if networkMgr != nil {
				dispatch(msg.Type, func() {
					networkMgr.HandleNetworkAction(sender, msg)
				})
			}
		case agentmgr.MsgPackageList:
			if packageMgr != nil {
				dispatch(msg.Type, func() {
					packageMgr.HandlePackageList(sender, msg)
				})
			}
		case agentmgr.MsgPackageAction:
			if packageMgr != nil {
				dispatch(msg.Type, func() {
					packageMgr.HandlePackageAction(sender, msg)
				})
			}
		case agentmgr.MsgCronList:
			if cronMgr != nil {
				dispatch(msg.Type, func() {
					cronMgr.HandleCronList(sender, msg)
				})
			}
		case agentmgr.MsgUsersList:
			if usersMgr != nil {
				dispatch(msg.Type, func() {
					usersMgr.HandleUsersList(sender, msg)
				})
			}
		case agentmgr.MsgAlertNotify:
			// Lightweight — no semaphore.
			inline(func() { handleAlertNotify(msg, runtime) })
		case agentmgr.MsgEnrollmentChallenge:
			// Lightweight — no semaphore.
			inline(func() { handleEnrollmentChallenge(transport, msg, cfg) })
		case agentmgr.MsgEnrollmentApproved:
			// Lightweight — no semaphore.
			inline(func() { handleEnrollmentApproved(transport, msg, cfg) })
		case agentmgr.MsgEnrollmentRejected:
			// Lightweight — no semaphore.
			inline(func() { handleEnrollmentRejected(msg) })
		// Clipboard messages.
		case agentmgr.MsgClipboardGet:
			if clipMgr != nil {
				dispatch(msg.Type, func() {
					clipMgr.HandleClipboardGet(sender, msg)
				})
			}
		case agentmgr.MsgClipboardSet:
			if clipMgr != nil {
				dispatch(msg.Type, func() {
					clipMgr.HandleClipboardSet(sender, msg)
				})
			}
		// Desktop audio sideband messages.
		case agentmgr.MsgDockerAction:
			if dockerCollector != nil {
				dispatch(msg.Type, func() {
					dockerCollector.HandleDockerAction(sender, msg)
				})
			}
		case agentmgr.MsgDockerComposeAction, agentmgr.MsgDockerComposeActionV2:
			if dockerCollector != nil {
				dispatch(msg.Type, func() {
					dockerCollector.HandleComposeAction(sender, msg)
				})
			}
		case agentmgr.MsgWebServiceSync:
			if webServiceCollector != nil {
				dispatch(msg.Type, func() {
					webServiceCollector.RunCycle(ctx)
				})
			}
		case msgPowerAction:
			// Lightweight — no semaphore.
			inline(func() { handlePowerActionUnsupported(sender, msg, transport.AssetID()) })
		default:
			log.Printf("agentws: unknown message type from hub: %s", msg.Type)
		}
	}
}

func runtimeConfigWithCurrentIdentity(cfg RuntimeConfig, transport *wsTransport) RuntimeConfig {
	if transport == nil {
		return cfg
	}
	identity := transport.identitySnapshot()
	cfg.APIToken = identity.token
	cfg.AssetID = identity.assetID
	if identity.wsBaseURL != "" {
		cfg.WSBaseURL = identity.wsBaseURL
	}
	if identity.apiBaseURL != "" {
		cfg.APIBaseURL = identity.apiBaseURL
	}
	return cfg
}

// handleAlertNotify processes an alert notification from the hub and caches it locally.
func handleAlertNotify(msg agentmgr.Message, runtime *Runtime) {
	var data agentmgr.AlertNotifyData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("agentws: invalid alert.notify: %v", err)
		return
	}

	ts, _ := time.Parse(time.RFC3339, data.Timestamp)
	snapshot := AlertSnapshot{
		ID:        data.ID,
		Severity:  data.Severity,
		Title:     data.Title,
		Summary:   data.Summary,
		State:     data.State,
		Timestamp: ts,
	}
	runtime.pushAlert(snapshot)
	log.Printf("agentws: alert %s [%s] %s: %s", data.ID, data.Severity, data.State, data.Title)
}

package agentcore

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labtether/labtether-linux/internal/agentcore/files"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"log"
	"strings"
	"sync"
)

func startFileReadHandler(ctx context.Context, transport files.MessageSender, fileMgr *files.Manager, msg agentmgr.Message, sem chan struct{}, handlerWG *sync.WaitGroup) bool {
	select {
	case sem <- struct{}{}:
	case <-ctx.Done():
		return false
	default:
		if owner, ok := transport.(interface{ Close() }); ok {
			owner.Close()
		}
		return true
	}

	handlerWG.Add(1)
	go func() {
		defer handlerWG.Done()
		defer func() { <-sem }()
		safeHandler("file-read", func() {
			fileMgr.HandleFileReadContext(ctx, transport, msg)
		})
	}()
	return true
}

const maxFileWriteQueuedMessageSize = 128 * 1024

type orderedFileWrite struct {
	ctx       context.Context
	transport files.MessageSender
	message   agentmgr.Message
}

func enqueueOrderedFileWrite(ctx context.Context, transport files.MessageSender, messages chan<- orderedFileWrite, msg agentmgr.Message) bool {
	messageSize := len(msg.Type) + len(msg.ID) + len(msg.Data)
	if messageSize > maxFileWriteQueuedMessageSize {
		requestID := strings.TrimSpace(msg.ID)
		if len(requestID) > 256 {
			requestID = requestID[:256]
		}
		errMessage := fmt.Sprintf("file.write message exceeds %d byte limit", maxFileWriteQueuedMessageSize)
		log.Printf(
			"agentws: rejected file.write message for request %q: %d bytes exceeds %d byte limit",
			requestID,
			messageSize,
			maxFileWriteQueuedMessageSize,
		)
		data, err := json.Marshal(agentmgr.FileWrittenData{RequestID: requestID, Error: errMessage})
		if err == nil {
			_ = transport.Send(agentmgr.Message{Type: agentmgr.MsgFileWritten, ID: requestID, Data: data})
		}
		return true
	}
	select {
	case messages <- orderedFileWrite{ctx: ctx, transport: transport, message: msg}:
		return true
	case <-ctx.Done():
		return false
	default:
		// A blocked disk must not keep the socket reader from seeing close.
		// Fail the originating transport so its pending upload is canceled.
		if owner, ok := transport.(interface{ Close() }); ok {
			owner.Close()
		}
		return true
	}
}

func runOrderedFileWriteWorker(ctx context.Context, fileMgr *files.Manager, messages <-chan orderedFileWrite) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			safeHandler("file-write", func() {
				if msg.ctx.Err() == nil {
					fileMgr.HandleFileWriteContext(msg.ctx, msg.transport, msg.message)
				}
			})
		}
	}
}

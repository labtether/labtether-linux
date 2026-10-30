package agentcore

import (
	"log"
	"runtime/debug"
)

// Keep a handler panic local to its bounded worker, as in the canonical agent.
func safeHandler(name string, fn func()) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler[%s]: panic recovered: %v\n%s", name, err, debug.Stack())
		}
	}()
	fn()
}

// recoverInlineMessage is deferred by handlers that run on the WebSocket
// reader itself. A panic there would otherwise stop the agent.
func recoverInlineMessage(name string) {
	if err := recover(); err != nil {
		log.Printf("handler[%s]: panic recovered: %v\n%s", name, err, debug.Stack())
	}
}

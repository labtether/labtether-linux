package agentcore

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"sync/atomic"
)

func (t *wsTransport) connectionIsCurrent(conn *websocket.Conn) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return conn != nil && t.conn == conn && t.connected
}

// A session's sends and cancellation belong to the socket that created it.
// Old output must never be delivered through a newly authenticated connection.
type sessionMessageSender struct {
	transport *wsTransport
	conn      *websocket.Conn
	ctx       context.Context
	assetID   string
}

func (s *sessionMessageSender) Send(msg agentmgr.Message) error {
	return s.transport.sendOnConnection(s.conn, msg)
}
func (s *sessionMessageSender) Context() context.Context { return s.ctx }
func (s *sessionMessageSender) Connected() bool {
	s.transport.mu.Lock()
	defer s.transport.mu.Unlock()
	return s.conn != nil && s.transport.conn == s.conn && s.transport.connected && !s.transport.pendingEnrollment
}
func (s *sessionMessageSender) Close()                        { s.transport.markDisconnectedIfCurrent(s.conn) }
func (s *sessionMessageSender) Connect(context.Context) error { return errNotConnected }
func (s *sessionMessageSender) Receive() (agentmgr.Message, error) {
	return agentmgr.Message{}, errNotConnected
}

// Hold only across cheap owner operations: reserve, enqueue, or cancel. The
// reconnect publisher holds the exclusive lock while clearing stale owners.
func (t *wsTransport) withCurrentSessionMessage(conn *websocket.Conn, handle func(*sessionMessageSender)) {
	t.sessionDispatchMu.RLock()
	defer t.sessionDispatchMu.RUnlock()
	t.mu.Lock()
	if conn == nil || t.conn != conn || !t.connected {
		t.mu.Unlock()
		return
	}
	ctx := t.connectionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	t.mu.Unlock()
	handle(&sessionMessageSender{transport: t, conn: conn, ctx: ctx, assetID: t.AssetID()})
}

func (t *wsTransport) Receive() (agentmgr.Message, error) {
	msg, _, err := t.receiveWithEnrollmentState()
	return msg, err
}

func (t *wsTransport) receiveWithEnrollmentState() (agentmgr.Message, bool, error) {
	msg, pending, _, err := t.receiveWithConnection()
	return msg, pending, err
}

func (t *wsTransport) receiveWithConnection() (agentmgr.Message, bool, *websocket.Conn, error) {
	t.mu.Lock()
	conn := t.conn
	pendingEnrollment := t.pendingEnrollment
	t.mu.Unlock()
	if conn == nil {
		return agentmgr.Message{}, pendingEnrollment, conn, errNotConnected
	}

	var msg agentmgr.Message
	err := conn.ReadJSON(&msg)
	if err == nil && !t.connectionIsCurrent(conn) {
		err = errNotConnected
	}
	if err == nil {
		atomic.AddInt64(&t.messagesReceived, 1)
	}
	return msg, pendingEnrollment, conn, err
}

func (t *wsTransport) senderForConnection(conn *websocket.Conn) *sessionMessageSender {
	var sender *sessionMessageSender
	t.withCurrentSessionMessage(conn, func(current *sessionMessageSender) { sender = current })
	return sender
}

func (s *sessionMessageSender) AssetID() string { return s.assetID }

type agentMessageSender interface{ Send(agentmgr.Message) error }

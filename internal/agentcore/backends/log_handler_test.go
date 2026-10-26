package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

type finiteLogBackend struct{}

func (finiteLogBackend) QueryEntries(agentmgr.JournalQueryData) ([]agentmgr.LogStreamData, error) {
	return nil, nil
}

func (finiteLogBackend) StreamEntries(_ context.Context, emit func(agentmgr.LogStreamData)) error {
	for index := range LogBatchMaxEntries + 3 {
		emit(agentmgr.LogStreamData{Message: fmt.Sprintf("entry-%d", index)})
	}
	return nil
}

type batchLogSender struct{ messages chan agentmgr.Message }

func (s batchLogSender) Send(message agentmgr.Message) error { s.messages <- message; return nil }
func (batchLogSender) Connected() bool                       { return true }
func (batchLogSender) AssetID() string                       { return "asset-1" }

// Releasing the batch's string references after sending must not erase the
// serialized entries or the final partial batch.
func TestLogManagerFlushPreservesEntries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender := batchLogSender{messages: make(chan agentmgr.Message, 2)}
	manager := &LogManager{Backend: finiteLogBackend{}}
	defer manager.CloseAll()
	manager.Start(ctx, sender)
	next := 0
	for _, count := range []int{LogBatchMaxEntries, 3} {
		select {
		case message := <-sender.messages:
			var batch agentmgr.LogBatchData
			if err := json.Unmarshal(message.Data, &batch); err != nil {
				t.Fatal(err)
			}
			if len(batch.Entries) != count || batch.AssetID != "asset-1" {
				t.Fatalf("unexpected batch: %+v", batch)
			}
			for _, entry := range batch.Entries {
				if entry.Message != fmt.Sprintf("entry-%d", next) {
					t.Fatalf("entry %d lost or reordered: %q", next, entry.Message)
				}
				next++
			}
		case <-time.After(time.Second):
			t.Fatal("log batch was not flushed")
		}
	}
}

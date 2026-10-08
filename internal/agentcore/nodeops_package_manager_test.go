package agentcore

import (
	"encoding/json"
	"errors"
	"github.com/labtether/labtether-linux/internal/agentcore/backends"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"reflect"
	"testing"
)

func TestPackageManagerHandlePackageListAndAction(t *testing.T) {
	t.Run("list success and error", func(t *testing.T) {
		transport, messages, cleanup := newDesktopRuntimeTransport(t)
		defer cleanup()

		successBackend := &stubPackageBackend{
			listPackages: []agentmgr.PackageInfo{{Name: "jq", Version: "1.7", Status: "installed"}},
		}
		manager := &backends.PackageManager{Backend: successBackend}
		manager.HandlePackageList(transport, agentmgr.Message{
			Type: agentmgr.MsgPackageList,
			Data: mustMarshalDesktopRuntime(t, agentmgr.PackageListData{RequestID: "req-package-list"}),
		})

		msg := readDesktopRuntimeMessage(t, messages)
		if msg.Type != agentmgr.MsgPackageListed {
			t.Fatalf("message type=%q, want %q", msg.Type, agentmgr.MsgPackageListed)
		}
		var listed agentmgr.PackageListedData
		if err := json.Unmarshal(msg.Data, &listed); err != nil {
			t.Fatalf("decode package listed payload: %v", err)
		}
		if listed.RequestID != "req-package-list" || len(listed.Packages) != 1 || listed.Packages[0].Name != "jq" {
			t.Fatalf("unexpected package list %+v", listed)
		}

		errorBackend := &stubPackageBackend{listErr: errors.New("rpm failed")}
		manager = &backends.PackageManager{Backend: errorBackend}
		manager.HandlePackageList(transport, agentmgr.Message{
			Type: agentmgr.MsgPackageList,
			Data: mustMarshalDesktopRuntime(t, agentmgr.PackageListData{RequestID: "req-package-error"}),
		})

		msg = readDesktopRuntimeMessage(t, messages)
		if err := json.Unmarshal(msg.Data, &listed); err != nil {
			t.Fatalf("decode package listed payload: %v", err)
		}
		if listed.Error != "rpm failed" {
			t.Fatalf("error=%q, want rpm failed", listed.Error)
		}
	})

	t.Run("action normalizes packages and reports result", func(t *testing.T) {
		transport, messages, cleanup := newDesktopRuntimeTransport(t)
		defer cleanup()

		backend := &stubPackageBackend{
			actionResult: backends.PackageActionResult{
				Output:         "installed",
				RebootRequired: true,
			},
		}
		manager := &backends.PackageManager{Backend: backend}
		manager.HandlePackageAction(transport, agentmgr.Message{
			Type: agentmgr.MsgPackageAction,
			Data: mustMarshalDesktopRuntime(t, agentmgr.PackageActionData{
				RequestID: "req-package-action",
				Action:    "install",
				Packages:  []string{" jq ", "jq", "", "curl"},
			}),
		})

		msg := readDesktopRuntimeMessage(t, messages)
		if msg.Type != agentmgr.MsgPackageResult {
			t.Fatalf("message type=%q, want %q", msg.Type, agentmgr.MsgPackageResult)
		}
		var result agentmgr.PackageResultData
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			t.Fatalf("decode package result payload: %v", err)
		}
		if !result.OK || result.Output != "installed" || !result.RebootRequired {
			t.Fatalf("unexpected package result %+v", result)
		}
		if got, want := backend.actionCalls, []stubBackendsPackageActionCall{{
			action:   "install",
			packages: []string{"jq", "curl"},
		}}; !reflect.DeepEqual(got, want) {
			t.Fatalf("backend calls=%#v, want %#v", got, want)
		}
	})

	t.Run("invalid action", func(t *testing.T) {
		transport, messages, cleanup := newDesktopRuntimeTransport(t)
		defer cleanup()

		backend := &stubPackageBackend{}
		manager := &backends.PackageManager{Backend: backend}
		manager.HandlePackageAction(transport, agentmgr.Message{
			Type: agentmgr.MsgPackageAction,
			Data: mustMarshalDesktopRuntime(t, agentmgr.PackageActionData{
				RequestID: "req-package-invalid",
				Action:    "repair",
			}),
		})

		msg := readDesktopRuntimeMessage(t, messages)
		var result agentmgr.PackageResultData
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			t.Fatalf("decode package result payload: %v", err)
		}
		if result.Error != "invalid package action" {
			t.Fatalf("error=%q, want invalid package action", result.Error)
		}
		if len(backend.actionCalls) != 0 {
			t.Fatalf("expected no backend calls, got %#v", backend.actionCalls)
		}
	})
}

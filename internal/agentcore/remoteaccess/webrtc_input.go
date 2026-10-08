package remoteaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labtether/labtether-linux/pkg/agentmgr"
	"os"
	"strconv"
	"strings"
)

type WebRTCInputEvent struct {
	Type    string `json:"type"`
	KeyCode int    `json:"keyCode,omitempty"`
	Code    string `json:"code,omitempty"`
	Key     string `json:"key,omitempty"`
	X       int    `json:"x,omitempty"`
	Y       int    `json:"y,omitempty"`
	Button  int    `json:"button,omitempty"`
	DeltaY  int    `json:"deltaY,omitempty"`
}

func InjectInputEvents(ctx context.Context, ch <-chan WebRTCInputEvent, display, xauthPath string, sess *WebRTCSession) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-ch:
			InjectSingleInput(evt, display, xauthPath, sess)
		}
	}
}

func BuildWaylandPipeWireEnv(session DesktopSessionInfo) []string {
	env := os.Environ()
	filtered := make([]string, 0, len(env)+2)
	for _, e := range env {
		if strings.HasPrefix(e, "XDG_RUNTIME_DIR=") {
			continue
		}
		filtered = append(filtered, e)
	}
	if runtimeDir := strings.TrimSpace(session.XDGRuntimeDir); runtimeDir != "" {
		filtered = append(filtered, "XDG_RUNTIME_DIR="+runtimeDir)
	}
	return filtered
}

func InjectSingleInput(evt WebRTCInputEvent, display, xauthPath string, sess *WebRTCSession) {
	eventType := strings.TrimSpace(strings.ToLower(evt.Type))
	if eventType == "" {
		return
	}
	if sess != nil && sess.sessionInfo.Type == DesktopSessionTypeWayland {
		InjectWaylandInputEvent(evt, sess)
		return
	}
	if strings.TrimSpace(display) == "" {
		display = ":0"
	}

	run := func(args ...string) {
		cmd, err := NewWebRTCSecurityCommand("xdotool", args...)
		if err != nil {
			return
		}
		cmd.Env = BuildX11ClientEnv(display, xauthPath)
		_ = cmd.Run()
	}

	switch eventType {
	case "keydown":
		if keyArg, ok := X11KeyArgument(evt); ok {
			run("keydown", keyArg)
			return
		}
		run("keydown", fmt.Sprintf("0x%x", evt.KeyCode))
	case "keyup":
		if keyArg, ok := X11KeyArgument(evt); ok {
			run("keyup", keyArg)
			return
		}
		run("keyup", fmt.Sprintf("0x%x", evt.KeyCode))
	case "mousemove":
		run("mousemove", "--screen", "0", strconv.Itoa(evt.X), strconv.Itoa(evt.Y))
	case "mousedown":
		run("mousedown", strconv.Itoa(evt.Button+1))
	case "mouseup":
		run("mouseup", strconv.Itoa(evt.Button+1))
	case "scroll":
		if evt.DeltaY < 0 {
			run("click", "4")
		} else if evt.DeltaY > 0 {
			run("click", "5")
		}
	}
}

func X11KeyArgument(evt WebRTCInputEvent) (string, bool) {
	if keysym, ok := DomCodeToX11Keysym(strings.TrimSpace(evt.Code)); ok {
		return keysym, true
	}
	if keysym, ok := DomKeyToX11Keysym(strings.TrimSpace(evt.Key)); ok {
		return keysym, true
	}
	if evt.KeyCode > 0 {
		return fmt.Sprintf("0x%x", evt.KeyCode), true
	}
	return "", false
}

func InjectWaylandInputEvent(evt WebRTCInputEvent, sess *WebRTCSession) {
	if sess == nil {
		return
	}
	backend := ResolveWaylandInputBackend(sess.inputBackend)
	if backend != "ydotool" {
		return
	}

	run := func(args ...string) {
		cmd, err := NewWebRTCSecurityCommand("ydotool", args...)
		if err != nil {
			return
		}
		_ = cmd.Run()
	}

	switch strings.TrimSpace(strings.ToLower(evt.Type)) {
	case "keydown", "keyup":
		code, ok := DomCodeToLinuxInputCode(strings.TrimSpace(evt.Code))
		if !ok {
			return
		}
		state := "0"
		if strings.EqualFold(strings.TrimSpace(evt.Type), "keydown") {
			state = "1"
		}
		run("key", fmt.Sprintf("%d:%s", code, state))
	case "mousemove":
		run("mousemove", "--absolute", "-x", strconv.Itoa(evt.X), "-y", strconv.Itoa(evt.Y))
	case "mousedown":
		if buttonCode, ok := BrowserButtonToYdotoolButton(evt.Button); ok {
			run("click", buttonCode)
		}
	case "scroll":
		if evt.DeltaY < 0 {
			run("click", "0xC3")
		} else if evt.DeltaY > 0 {
			run("click", "0xC4")
		}
	}
}

func ResolveWaylandInputBackend(configured string) string {
	switch strings.TrimSpace(strings.ToLower(configured)) {
	case "none":
		return "none"
	case "ydotool":
		return "ydotool"
	case "auto", "":
		if _, err := WebRTCLookPath("ydotool"); err == nil {
			return "ydotool"
		}
	}
	return "none"
}

func BrowserButtonToYdotoolButton(button int) (string, bool) {
	switch button {
	case 0:
		return "0xC0", true
	case 1:
		return "0xC2", true
	case 2:
		return "0xC1", true
	default:
		return "", false
	}
}

func DecodeWebRTCInputEvent(raw []byte) (WebRTCInputEvent, error) {
	var evt WebRTCInputEvent
	var fallback agentmgr.WebRTCInputData

	directErr := json.Unmarshal(raw, &evt)
	fallbackErr := json.Unmarshal(raw, &fallback)
	if directErr != nil && fallbackErr != nil {
		return WebRTCInputEvent{}, directErr
	}

	if strings.TrimSpace(evt.Type) == "" && strings.TrimSpace(fallback.Type) == "" {
		return WebRTCInputEvent{}, fmt.Errorf("missing type")
	}

	if strings.TrimSpace(evt.Type) == "" {
		evt.Type = fallback.Type
	}
	if evt.KeyCode == 0 && fallback.KeyCode != 0 {
		evt.KeyCode = fallback.KeyCode
	}
	if strings.TrimSpace(evt.Code) == "" && strings.TrimSpace(fallback.Code) != "" {
		evt.Code = fallback.Code
	}
	if strings.TrimSpace(evt.Key) == "" && strings.TrimSpace(fallback.Key) != "" {
		evt.Key = fallback.Key
	}
	if evt.X == 0 && fallback.X != 0 {
		evt.X = fallback.X
	}
	if evt.Y == 0 && fallback.Y != 0 {
		evt.Y = fallback.Y
	}
	if evt.Button == 0 && fallback.Button != 0 {
		evt.Button = fallback.Button
	}
	if evt.DeltaY == 0 && fallback.DeltaY != 0 {
		evt.DeltaY = fallback.DeltaY
	}
	return evt, nil
}

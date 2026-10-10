package system

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

// MessageSender abstracts the agent-to-hub send capability so this package
// does not depend on the concrete wsTransport type in the parent agentcore package.
type MessageSender interface {
	Send(msg agentmgr.Message) error
}

// ProcessManager handles process list requests from the hub.
// It carries no persistent state; the struct exists for consistency
// with the other manager types.
type ProcessManager struct{}

var (
	CollectProcessesFn = CollectProcesses
	KillProcessFn      = KillProcess
)

// NewProcessManager creates a new ProcessManager.
func NewProcessManager() *ProcessManager {
	return &ProcessManager{}
}

// CloseAll is a no-op for ProcessManager -- process list requests are
// stateless and require no cleanup.
func (pm *ProcessManager) CloseAll() {}

// HandleProcessKill sends a signal to a process identified by PID.
func (pm *ProcessManager) HandleProcessKill(transport MessageSender, msg agentmgr.Message) {
	var req agentmgr.ProcessKillData
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("process: invalid process.kill request: %v", err)
		return
	}

	result := agentmgr.ProcessKillResultData{PID: req.PID}

	if req.PID <= 1 {
		result.Error = "refusing to signal PID <= 1"
		sendProcessKillResult(transport, msg.ID, result)
		return
	}

	if err := KillProcessFn(req.PID, req.Signal); err != nil {
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	sendProcessKillResult(transport, msg.ID, result)
}

// sendProcessKillResult marshals and transmits a ProcessKillResultData to the hub.
func sendProcessKillResult(transport MessageSender, requestID string, result agentmgr.ProcessKillResultData) {
	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("process: failed to marshal process.kill_result: %v", err)
		return
	}
	if sendErr := transport.Send(agentmgr.Message{
		Type: agentmgr.MsgProcessKillResult,
		ID:   requestID,
		Data: data,
	}); sendErr != nil {
		log.Printf("process: failed to send process.kill_result for PID %d: %v", result.PID, sendErr)
	}
}

// HandleProcessList collects the running process list and sends it to the hub.
func (pm *ProcessManager) HandleProcessList(transport MessageSender, msg agentmgr.Message) {
	var req agentmgr.ProcessListData
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("process: invalid process.list request: %v", err)
		return
	}

	// Apply defaults.
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "cpu"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 25
	}

	processes, err := CollectProcessesFn()

	var errMsg string
	if err != nil {
		errMsg = err.Error()
		log.Printf("process: failed to collect processes: %v", err)
	}

	// Sort by the requested field.
	switch sortBy {
	case "memory":
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].MemPct > processes[j].MemPct
		})
	default: // "cpu"
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].CPUPct > processes[j].CPUPct
		})
	}

	// Truncate to limit.
	if len(processes) > limit {
		processes = processes[:limit]
	}

	data, marshalErr := json.Marshal(agentmgr.ProcessListedData{
		RequestID: req.RequestID,
		Processes: processes,
		Error:     errMsg,
	})
	if marshalErr != nil {
		log.Printf("process: failed to marshal process.listed response: %v", marshalErr)
		return
	}

	if sendErr := transport.Send(agentmgr.Message{
		Type: agentmgr.MsgProcessListed,
		ID:   req.RequestID,
		Data: data,
	}); sendErr != nil {
		log.Printf("process: failed to send process.listed for request %s: %v", req.RequestID, sendErr)
	}
}

// CollectProcesses requests explicit columns so ps variants cannot silently
// change the field positions. BusyBox lacks CPU and memory percentage columns.
func CollectProcesses() ([]agentmgr.ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return collectProcessesWithPS(func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "ps", args...).CombinedOutput()
	})
}

func collectProcessesWithPS(run func(...string) ([]byte, error)) ([]agentmgr.ProcessInfo, error) {
	out, err := run("-axo", "user=,pid=,%cpu=,%mem=,rss=,command=")
	if err == nil {
		return parseProcessRows(out, true), nil
	}
	out, err = run("-o", "user=,pid=,rss=,args=")
	if err != nil {
		return nil, err
	}
	return parseProcessRows(out, false), nil
}

func parseProcessRows(out []byte, withPercentages bool) []agentmgr.ProcessInfo {
	var processes []agentmgr.ProcessInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		minimumFields := 4
		if withPercentages {
			minimumFields = 6
		}
		if len(fields) < minimumFields {
			continue
		}
		user := fields[0]
		pid, pidErr := strconv.Atoi(fields[1])
		if pidErr != nil {
			continue
		}
		var cpuPct, memPct float64
		rssIndex, commandIndex := 2, 3
		if withPercentages {
			cpuPct, _ = strconv.ParseFloat(fields[2], 64)
			memPct, _ = strconv.ParseFloat(fields[3], 64)
			rssIndex, commandIndex = 4, 5
		}
		memRSS := parsePSRSSKB(fields[rssIndex])
		command := strings.Join(fields[commandIndex:], " ")
		// Derive a short name from the command path (last path component).
		name := command
		if parts := strings.Fields(command); len(parts) > 0 {
			exe := parts[0]
			if idx := strings.LastIndexByte(exe, '/'); idx >= 0 {
				exe = exe[idx+1:]
			}
			name = exe
		}

		processes = append(processes, agentmgr.ProcessInfo{
			PID:     pid,
			Name:    name,
			User:    user,
			CPUPct:  cpuPct,
			MemPct:  memPct,
			MemRSS:  memRSS,
			Command: command,
		})
	}
	return processes
}

func parsePSRSSKB(raw string) int64 {
	if size, err := strconv.ParseInt(raw, 10, 64); err == nil && size >= 0 {
		return size
	}
	if len(raw) < 2 {
		return 0
	}
	factor := float64(0)
	switch raw[len(raw)-1] {
	case 'k', 'K':
		factor = 1
	case 'm', 'M':
		factor = 1024
	case 'g', 'G':
		factor = 1024 * 1024
	}
	size, err := strconv.ParseFloat(raw[:len(raw)-1], 64)
	if err != nil || size < 0 || factor == 0 || size*factor > 1<<50 {
		return 0
	}
	return int64(size*factor + 0.5)
}

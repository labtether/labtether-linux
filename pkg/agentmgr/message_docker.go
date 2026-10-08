package agentmgr

// --- Docker Integration Data Structs ---

// DockerEngineInfo describes the Docker daemon on an agent host.
type DockerEngineInfo struct {
	Version    string `json:"version"`
	APIVersion string `json:"api_version"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

// DockerPortMapping describes a container port mapping.
type DockerPortMapping struct {
	Host      int    `json:"host"`
	Container int    `json:"container"`
	Protocol  string `json:"protocol"`
}

// DockerMountInfo describes a container mount/bind.
type DockerMountInfo struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// DockerContainerInfo describes a container in the discovery payload.
type DockerContainerInfo struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Image    string              `json:"image"`
	State    string              `json:"state"`
	Status   string              `json:"status"`
	Created  string              `json:"created"`
	Ports    []DockerPortMapping `json:"ports,omitempty"`
	Networks []string            `json:"networks,omitempty"`
	Labels   map[string]string   `json:"labels,omitempty"`
	Mounts   []DockerMountInfo   `json:"mounts,omitempty"`
}

// DockerImageInfo describes a Docker image.
type DockerImageInfo struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created string   `json:"created"`
}

// DockerNetworkInfo describes a Docker network.
type DockerNetworkInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
}

// DockerVolumeInfo describes a Docker volume.
type DockerVolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
}

// DockerComposeStack describes an inferred Compose stack.
type DockerComposeStack struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	ConfigFile string   `json:"config_file,omitempty"`
	Containers []string `json:"containers"`
}

// DockerDiscoveryData is the payload for docker.discovery messages.
type DockerDiscoveryData struct {
	HostID        string                `json:"host_id"`
	Engine        DockerEngineInfo      `json:"engine"`
	Containers    []DockerContainerInfo `json:"containers"`
	Images        []DockerImageInfo     `json:"images"`
	Networks      []DockerNetworkInfo   `json:"networks"`
	Volumes       []DockerVolumeInfo    `json:"volumes"`
	ComposeStacks []DockerComposeStack  `json:"compose_stacks"`
}

// DockerDiscoveryDeltaData is the payload for docker.discovery.delta messages.
// It allows the agent to upsert/remove subsets of inventory without sending
// a full docker.discovery snapshot on every change.
type DockerDiscoveryDeltaData struct {
	HostID               string                `json:"host_id"`
	Engine               *DockerEngineInfo     `json:"engine,omitempty"`
	UpsertContainers     []DockerContainerInfo `json:"upsert_containers,omitempty"`
	RemoveContainerIDs   []string              `json:"remove_container_ids,omitempty"`
	UpsertImages         []DockerImageInfo     `json:"upsert_images,omitempty"`
	RemoveImageIDs       []string              `json:"remove_image_ids,omitempty"`
	UpsertNetworks       []DockerNetworkInfo   `json:"upsert_networks,omitempty"`
	RemoveNetworkIDs     []string              `json:"remove_network_ids,omitempty"`
	UpsertVolumes        []DockerVolumeInfo    `json:"upsert_volumes,omitempty"`
	RemoveVolumeNames    []string              `json:"remove_volume_names,omitempty"`
	ComposeStacks        []DockerComposeStack  `json:"compose_stacks,omitempty"`
	ReplaceComposeStacks bool                  `json:"replace_compose_stacks,omitempty"`
}

// DockerContainerStats holds per-container resource metrics.
type DockerContainerStats struct {
	ID              string  `json:"id"`
	CPUPercent      float64 `json:"cpu_percent"`
	MemoryBytes     int64   `json:"memory_bytes"`
	MemoryLimit     int64   `json:"memory_limit"`
	MemoryPercent   float64 `json:"memory_percent"`
	NetRXBytes      int64   `json:"net_rx_bytes"`
	NetTXBytes      int64   `json:"net_tx_bytes"`
	BlockReadBytes  int64   `json:"block_read_bytes"`
	BlockWriteBytes int64   `json:"block_write_bytes"`
	PIDs            int     `json:"pids"`
}

// DockerStatsData is the payload for docker.stats messages.
type DockerStatsData struct {
	HostID     string                 `json:"host_id"`
	Containers []DockerContainerStats `json:"containers"`
}

// DockerEventActor describes the target of a Docker event.
type DockerEventActor struct {
	ID         string            `json:"id"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// DockerEventData is the payload for docker.events messages.
type DockerEventData struct {
	HostID    string           `json:"host_id"`
	Type      string           `json:"type"`
	Action    string           `json:"action"`
	Actor     DockerEventActor `json:"actor"`
	Timestamp int64            `json:"timestamp"`
}

// DockerActionData is the payload for docker.action messages (Hub -> Agent).
type DockerActionData struct {
	RequestID   string            `json:"request_id"`
	Action      string            `json:"action"`
	ContainerID string            `json:"container_id,omitempty"`
	ImageRef    string            `json:"image_ref,omitempty"`
	Params      map[string]string `json:"params,omitempty"`
}

// DockerActionResultData is the payload for docker.action.result messages.
type DockerActionResultData struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Data      string `json:"data,omitempty"`
}

// DockerLogsStartData is the payload for docker.logs.start messages.
type DockerLogsStartData struct {
	SessionID   string `json:"session_id"`
	ContainerID string `json:"container_id"`
	Tail        int    `json:"tail"`
	Follow      bool   `json:"follow"`
	Timestamps  bool   `json:"timestamps,omitempty"`
}

// DockerLogsStopData is the payload for docker.logs.stop messages.
type DockerLogsStopData struct {
	SessionID string `json:"session_id"`
}

// DockerLogsStreamData is the payload for docker.logs.stream messages.
type DockerLogsStreamData struct {
	SessionID string `json:"session_id"`
	Stream    string `json:"stream"` // "stdout" or "stderr"
	Data      string `json:"data"`   // log line content
	Timestamp string `json:"timestamp,omitempty"`
}

// DockerExecStartData is the payload for docker.exec.start messages.
type DockerExecStartData struct {
	SessionID   string   `json:"session_id"`
	ContainerID string   `json:"container_id"`
	Command     []string `json:"command"`
	TTY         bool     `json:"tty"`
	Cols        int      `json:"cols,omitempty"`
	Rows        int      `json:"rows,omitempty"`
}

// DockerExecStartedData is the payload for docker.exec.started messages.
type DockerExecStartedData struct {
	SessionID string `json:"session_id"`
}

// DockerExecDataPayload carries exec session I/O (base64-encoded).
type DockerExecDataPayload struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"` // base64-encoded bytes
}

// DockerExecInputData carries exec stdin from hub to agent (base64-encoded).
type DockerExecInputData struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"` // base64-encoded bytes
}

// DockerExecResizeData is the payload for docker.exec.resize messages.
type DockerExecResizeData struct {
	SessionID string `json:"session_id"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

// DockerExecCloseData is the payload for docker.exec.close/closed messages.
type DockerExecCloseData struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// DockerComposeActionData is the payload for docker.compose.action messages.
type DockerComposeActionData struct {
	RequestID   string `json:"request_id"`
	StackName   string `json:"stack_name"`
	Action      string `json:"action"` // up, down, restart, pull, deploy
	ConfigDir   string `json:"config_dir,omitempty"`
	ComposeYAML string `json:"compose_yaml,omitempty"`
}

// DockerComposeResultData is the payload for docker.compose.result messages.
type DockerComposeResultData struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

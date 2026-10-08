package agentmgr

// --- Web Service Discovery Data Structs ---

// WebServiceReportData is sent by the agent with discovered web services.
type WebServiceReportData struct {
	HostAssetID string                    `json:"host_asset_id"`
	Services    []DiscoveredWebService    `json:"services"`
	Discovery   *WebServiceDiscoveryStats `json:"discovery,omitempty"`
}

// WebServiceDiscoveryStats captures one discovery cycle summary from an agent.
type WebServiceDiscoveryStats struct {
	CollectedAt      string                                   `json:"collected_at"`
	CycleDurationMs  int                                      `json:"cycle_duration_ms"`
	TotalServices    int                                      `json:"total_services"`
	Sources          map[string]WebServiceDiscoverySourceStat `json:"sources,omitempty"`
	FinalSourceCount map[string]int                           `json:"final_source_count,omitempty"`
}

// WebServiceDiscoverySourceStat captures per-source cycle behavior and yield.
type WebServiceDiscoverySourceStat struct {
	Enabled       bool `json:"enabled"`
	DurationMs    int  `json:"duration_ms"`
	ServicesFound int  `json:"services_found"`
}

// WebServiceHealthPoint captures one status check sample for a web service.
type WebServiceHealthPoint struct {
	At         string `json:"at"`
	Status     string `json:"status"`
	ResponseMs int    `json:"response_ms,omitempty"`
}

// WebServiceHealthSummary captures rolling uptime and recent status history.
type WebServiceHealthSummary struct {
	Window        string                  `json:"window"`
	Checks        int                     `json:"checks"`
	UpChecks      int                     `json:"up_checks"`
	UptimePercent float64                 `json:"uptime_percent"`
	LastCheckedAt string                  `json:"last_checked_at,omitempty"`
	LastChangeAt  string                  `json:"last_change_at,omitempty"`
	Recent        []WebServiceHealthPoint `json:"recent,omitempty"`
}

// DiscoveredWebService represents a single discovered web service.
type DiscoveredWebService struct {
	ID          string                   `json:"id"`
	ServiceKey  string                   `json:"service_key"`
	Name        string                   `json:"name"`
	Category    string                   `json:"category"`
	URL         string                   `json:"url"`
	Source      string                   `json:"source"`
	Status      string                   `json:"status"`
	ResponseMs  int                      `json:"response_ms"`
	ContainerID string                   `json:"container_id,omitempty"`
	ServiceUnit string                   `json:"service_unit,omitempty"`
	HostAssetID string                   `json:"host_asset_id"`
	IconKey     string                   `json:"icon_key"`
	Metadata    map[string]string        `json:"metadata,omitempty"`
	Health      *WebServiceHealthSummary `json:"health,omitempty"`
}

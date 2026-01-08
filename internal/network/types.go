package network

import "time"

// ScanResults contains parsed network scan output
type ScanResults struct {
	Hosts        []HostResult  `json:"hosts"`
	ScanTime     time.Time     `json:"scan_time"`
	ScanDuration time.Duration `json:"scan_duration"`
}

// HostResult contains scan results for a single host
type HostResult struct {
	IP       string       `json:"ip"`
	Hostname string       `json:"hostname,omitempty"`
	Status   string       `json:"status"` // up, down
	Ports    []PortResult `json:"ports"`
}

// PortResult contains information about an open port
type PortResult struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp, udp
	State    string `json:"state"`    // open, closed, filtered
	Service  string `json:"service"`  // http, ssh, etc.
	Version  string `json:"version,omitempty"`
	Product  string `json:"product,omitempty"`
}

// LLMAnalysis contains the LLM's security analysis
type LLMAnalysis struct {
	Summary          string   `json:"summary"`
	RiskLevel        string   `json:"risk_level"` // low, medium, high, critical
	HighRiskServices []string `json:"high_risk_services"`
	Recommendations  []string `json:"recommendations"`
	RawResponse      string   `json:"raw_response"`
}

// PingResult represents the result of pinging a single IP
type PingResult struct {
	IP      string  `json:"ip"`
	Alive   bool    `json:"alive"`
	Latency float64 `json:"latency"` // Average latency in ms
}

// PingToolOutput represents the output from the ping tool
type PingToolOutput struct {
	Results []PingResult `json:"results"`
}

// NmapToolOutput represents the output from the nmap tool
type NmapToolOutput struct {
	Hosts    []HostResult `json:"hosts"`
	ScanTime string       `json:"scan_time"` // ISO timestamp
}

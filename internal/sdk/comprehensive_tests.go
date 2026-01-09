package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"text/template"
	"time"

	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/finding"
	"github.com/zero-day-ai/sdk/graphrag"
	"github.com/zero-day-ai/sdk/llm"

	"github.com/zero-day-ai/agents/debug/internal/runner"
)

// ============================================================================
// Types for network scanning and analysis
// ============================================================================

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

// HostAnalysis contains per-host LLM analysis
type HostAnalysis struct {
	IP              string   `json:"ip"`
	Hostname        string   `json:"hostname,omitempty"`
	Purpose         string   `json:"purpose"`         // What this machine likely does
	OperatingSystem string   `json:"operating_system"` // Inferred OS
	RiskLevel       string   `json:"risk_level"`       // low, medium, high, critical
	Vulnerabilities []string `json:"vulnerabilities"`  // Potential vulnerabilities
	Recommendations []string `json:"recommendations"`  // Security recommendations
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

// ============================================================================
// Comprehensive SDK Module
// ============================================================================

// ComprehensiveSDKModule tests all SDK functionality in one module
type ComprehensiveSDKModule struct {
	BaseModule
}

// NewComprehensiveSDKModule creates the comprehensive SDK test module
func NewComprehensiveSDKModule() *ComprehensiveSDKModule {
	return &ComprehensiveSDKModule{
		BaseModule: NewBaseModule(
			"network-recon",
			"Comprehensive network reconnaissance: real ping sweep, nmap scanning, GraphRAG storage, per-host LLM analysis",
			"NR-1..NR-8",
		),
	}
}

// Run executes all SDK tests with real network reconnaissance
func (m *ComprehensiveSDKModule) Run(ctx context.Context, h agent.Harness) []runner.TestResult {
	results := []runner.TestResult{}

	// Phase 1: Parse subnet from target
	subnet, parseResults := m.parseSubnet(ctx, h)
	results = append(results, parseResults...)
	if subnet == "" {
		return results // Cannot continue without subnet
	}

	// Phase 2: Ping sweep - call REAL ping tool
	liveHosts, pingResults := m.pingPhase(ctx, h, subnet)
	results = append(results, pingResults...)

	// Phase 3: Nmap scan - call REAL nmap tool on live hosts
	scanResults, nmapResults := m.nmapPhase(ctx, h, liveHosts)
	results = append(results, nmapResults...)

	// Phase 4: Store in working memory (tests memory system)
	memResults := m.memoryPhase(ctx, h, subnet, liveHosts, scanResults)
	results = append(results, memResults...)

	// Phase 5: GraphRAG storage - build proper graph with relationships
	graphResults := m.graphPhase(ctx, h, subnet, liveHosts, scanResults)
	results = append(results, graphResults...)

	// Phase 6: Per-host LLM analysis via Claude (visible in Langfuse)
	hostAnalyses, llmResults := m.llmPhase(ctx, h, scanResults)
	results = append(results, llmResults...)

	// Phase 7: Store analyses in graph
	analysisGraphResults := m.storeAnalysesInGraph(ctx, h, hostAnalyses)
	results = append(results, analysisGraphResults...)

	// Phase 8: Submit findings for discovered vulnerabilities
	findingsResults := m.findingsPhase(ctx, h, scanResults, hostAnalyses)
	results = append(results, findingsResults...)

	return results
}

// ============================================================================
// Phase 1: Parse subnet from target
// ============================================================================

func (m *ComprehensiveSDKModule) parseSubnet(ctx context.Context, h agent.Harness) (string, []runner.TestResult) {
	testName := "Parse Subnet from Target"
	reqID := "NR-1"

	h.Logger().Info("Phase 1: Parsing subnet from target configuration")

	target := h.Target()
	if target.ID == "" {
		return "", []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Target info not available"),
		}
	}

	h.Logger().Info("Target info retrieved",
		"target_id", target.ID,
		"target_name", target.Name,
		"connection_keys", getMapKeys(target.Connection),
	)

	// Check if subnet exists in connection config (support both "subnet" and "cidr" keys)
	subnetRaw, ok := target.Connection["subnet"]
	if !ok {
		subnetRaw, ok = target.Connection["cidr"]
	}
	if !ok {
		return "", []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Target connection does not contain 'subnet' or 'cidr' field. Add subnet/cidr to target connection config."),
		}
	}

	// Convert to string
	subnet, ok := subnetRaw.(string)
	if !ok {
		return "", []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, 0,
				fmt.Sprintf("Subnet field is not a string, got type %T", subnetRaw),
				fmt.Errorf("invalid subnet type")),
		}
	}

	// Validate CIDR notation
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, 0,
				fmt.Sprintf("Invalid CIDR notation: %s", subnet), err),
		}
	}

	h.Logger().Info("Subnet parsed successfully",
		"subnet", subnet,
		"network", ipNet.String(),
	)

	return subnet, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, 0,
			fmt.Sprintf("Subnet parsed: %s (network: %s)", subnet, ipNet.String())),
	}
}

// ============================================================================
// Phase 2: Ping sweep using REAL ping tool
// ============================================================================

func (m *ComprehensiveSDKModule) pingPhase(ctx context.Context, h agent.Harness, subnet string) ([]string, []runner.TestResult) {
	testName := "Ping Sweep (Real Tool)"
	reqID := "NR-2"
	startTime := time.Now()

	h.Logger().Info("Phase 2: Starting ping sweep with real ping tool",
		"subnet", subnet,
	)

	// Enumerate IPs from CIDR
	ips, err := enumerateIPs(subnet)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to enumerate IPs from subnet %s", subnet), err),
		}
	}

	// Safety check: limit to /24 or smaller
	if len(ips) > 256 {
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				fmt.Sprintf("Subnet too large (%d IPs). Max 256 IPs for safety", len(ips))),
		}
	}

	h.Logger().Info("Starting parallel ping sweep",
		"ip_count", len(ips),
		"max_concurrency", 20,
	)

	// Build tool calls for all IPs
	calls := make([]agent.ToolCall, len(ips))
	for i, ip := range ips {
		calls[i] = agent.ToolCall{
			Name: "ping",
			Input: map[string]any{
				"targets": []string{ip}, // Single target per call for parallel execution
				"timeout": 1000,         // 1 second timeout
				"count":   1,            // 1 ping per host
			},
		}
	}

	// Execute all pings in parallel (max 20 concurrent - pings are lightweight)
	results, err := h.CallToolsParallel(ctx, calls, 20)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Parallel ping execution failed: %v", err), err),
		}
	}

	h.Logger().Info("Ping sweep completed, processing results",
		"result_count", len(results),
	)

	// Extract live hosts from results
	liveHosts := []string{}
	for i, result := range results {
		if result.Error != nil {
			h.Logger().Warn("Ping failed for IP",
				"ip", ips[i],
				"error", result.Error,
			)
			continue
		}

		// Parse ping result
		resultBytes, err := json.Marshal(result.Output)
		if err != nil {
			h.Logger().Warn("Failed to marshal ping result",
				"ip", ips[i],
				"error", err,
			)
			continue
		}

		var pingOutput PingToolOutput
		if err := json.Unmarshal(resultBytes, &pingOutput); err != nil {
			h.Logger().Warn("Failed to parse ping output",
				"ip", ips[i],
				"error", err,
			)
			continue
		}

		// Extract live host from this result
		for _, pingResult := range pingOutput.Results {
			if pingResult.Alive {
				liveHosts = append(liveHosts, pingResult.IP)
				h.Logger().Info("Host alive",
					"ip", pingResult.IP,
					"latency_ms", pingResult.Latency,
				)
			}
		}
	}

	duration := time.Since(startTime)
	h.Logger().Info("Ping sweep completed",
		"duration", duration,
		"scanned", len(ips),
		"alive", len(liveHosts),
	)

	return liveHosts, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Ping sweep: %d/%d hosts alive (%.1f%%)", len(liveHosts), len(ips), float64(len(liveHosts))/float64(len(ips))*100)),
	}
}

// ============================================================================
// Phase 3: Nmap scan using REAL nmap tool
// ============================================================================

func (m *ComprehensiveSDKModule) nmapPhase(ctx context.Context, h agent.Harness, liveHosts []string) (*ScanResults, []runner.TestResult) {
	testName := "Nmap Port Scan (Real Tool)"
	reqID := "NR-3"
	startTime := time.Now()

	if len(liveHosts) == 0 {
		h.Logger().Info("Phase 3: Skipping nmap - no live hosts")
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"No live hosts to scan"),
		}
	}

	h.Logger().Info("Phase 3: Starting nmap scan with real nmap tool (parallel execution)",
		"host_count", len(liveHosts),
		"hosts", liveHosts,
		"max_concurrency", 5,
	)

	// Build tool calls for all hosts
	calls := make([]agent.ToolCall, len(liveHosts))
	for i, host := range liveHosts {
		calls[i] = agent.ToolCall{
			Name: "nmap",
			Input: map[string]any{
				"target":            host,
				"service_detection": true,
				"scan_type":         "connect",
				"ports":             "1-1024", // All well-known ports
				"timing":            4,        // Aggressive timing
				"timeout":           "5m",     // 5 minute timeout per host
			},
		}
		h.Logger().Info("Queued scan for host", "ip", host)
	}

	// Execute all scans in parallel (max 5 concurrent to avoid overwhelming network)
	results, err := h.CallToolsParallel(ctx, calls, 5)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Parallel nmap execution failed: %v", err), err),
		}
	}

	// Process results
	allHosts := []HostResult{}
	for i, result := range results {
		if result.Error != nil {
			h.Logger().Warn("Nmap scan failed for host",
				"ip", liveHosts[i],
				"error", result.Error,
			)
			continue
		}

		// Parse nmap results
		resultBytes, err := json.Marshal(result.Output)
		if err != nil {
			h.Logger().Warn("Failed to marshal nmap result",
				"ip", liveHosts[i],
				"error", err,
			)
			continue
		}

		var nmapOutput NmapToolOutput
		if err := json.Unmarshal(resultBytes, &nmapOutput); err != nil {
			h.Logger().Warn("Failed to parse nmap output",
				"ip", liveHosts[i],
				"error", err,
			)
			continue
		}

		allHosts = append(allHosts, nmapOutput.Hosts...)

		// Log discovered ports
		for _, hostResult := range nmapOutput.Hosts {
			for _, port := range hostResult.Ports {
				if port.State == "open" {
					h.Logger().Info("Port discovered",
						"ip", hostResult.IP,
						"port", port.Port,
						"protocol", port.Protocol,
						"service", port.Service,
						"version", port.Version,
					)
				}
			}
		}
	}

	scanResults := &ScanResults{
		Hosts:        allHosts,
		ScanTime:     time.Now(),
		ScanDuration: time.Since(startTime),
	}

	// Count total open ports
	totalPorts := 0
	for _, host := range scanResults.Hosts {
		for _, port := range host.Ports {
			if port.State == "open" {
				totalPorts++
			}
		}
	}

	duration := time.Since(startTime)
	h.Logger().Info("Nmap scan completed",
		"duration", duration,
		"hosts_scanned", len(liveHosts),
		"hosts_with_results", len(allHosts),
		"total_open_ports", totalPorts,
	)

	return scanResults, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Nmap scan: %d hosts, %d open ports found", len(allHosts), totalPorts)),
	}
}

// ============================================================================
// Phase 4: Store in working memory (tests memory system)
// ============================================================================

func (m *ComprehensiveSDKModule) memoryPhase(ctx context.Context, h agent.Harness, subnet string, liveHosts []string, scan *ScanResults) []runner.TestResult {
	testName := "Working Memory Storage"
	reqID := "NR-4"
	startTime := time.Now()

	h.Logger().Info("Phase 4: Storing scan data in working memory")

	mem := h.Memory()
	if mem == nil {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, 0,
				"Memory store is nil", fmt.Errorf("memory not available")),
		}
	}

	working := mem.Working()
	if working == nil {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, 0,
				"Working memory is nil", fmt.Errorf("working memory not available")),
		}
	}

	// Store subnet
	if err := working.Set(ctx, "scan_subnet", subnet); err != nil {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				"Failed to store subnet", err),
		}
	}
	h.Logger().Info("Stored subnet in memory", "key", "scan_subnet")

	// Store live hosts
	if err := working.Set(ctx, "live_hosts", liveHosts); err != nil {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				"Failed to store live hosts", err),
		}
	}
	h.Logger().Info("Stored live hosts in memory", "key", "live_hosts", "count", len(liveHosts))

	// Store scan results
	if scan != nil {
		if err := working.Set(ctx, "scan_results", scan); err != nil {
			return []runner.TestResult{
				runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
					"Failed to store scan results", err),
			}
		}
		h.Logger().Info("Stored scan results in memory", "key", "scan_results", "hosts", len(scan.Hosts))
	}

	// Verify retrieval
	retrieved, err := working.Get(ctx, "scan_subnet")
	if err != nil || retrieved != subnet {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				"Memory retrieval verification failed", err),
		}
	}
	h.Logger().Info("Verified memory retrieval", "retrieved_subnet", retrieved)

	duration := time.Since(startTime)
	return []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Stored in memory: subnet, %d live hosts, scan results", len(liveHosts))),
	}
}

// ============================================================================
// Phase 5: GraphRAG storage with proper relationships
// ============================================================================

func (m *ComprehensiveSDKModule) graphPhase(ctx context.Context, h agent.Harness, subnet string, liveHosts []string, scan *ScanResults) []runner.TestResult {
	testName := "GraphRAG Storage"
	reqID := "NR-5"
	startTime := time.Now()

	h.Logger().Info("Phase 5: Storing scan data in Neo4j with relationships")

	// Check GraphRAG health
	health := h.GraphRAGHealth(ctx)
	if health.Status != "healthy" {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				fmt.Sprintf("GraphRAG unavailable: %s - %s", health.Status, health.Message)),
		}
	}

	// Get mission context for attack ID
	mission := h.Mission()
	if mission.ID == "" {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Mission context not available"),
		}
	}

	attackID := mission.ID
	h.Logger().Info("Building graph batch",
		"attack_id", attackID,
		"subnet", subnet,
		"live_hosts", len(liveHosts),
	)

	// Build nodes and relationships
	nodes := []graphrag.GraphNode{}
	relationships := []graphrag.Relationship{}

	// 1. Attack node (root)
	attackNode := graphrag.NewGraphNode("Attack").
		WithID(fmt.Sprintf("attack-%s", attackID)).
		WithProperty("attack_id", attackID).
		WithProperty("timestamp", time.Now()).
		WithProperty("agent", "debug-agent").
		WithContent(fmt.Sprintf("Network reconnaissance attack %s", attackID))
	nodes = append(nodes, *attackNode)

	// 2. NetworkScan node
	totalPorts := 0
	if scan != nil {
		for _, host := range scan.Hosts {
			for _, port := range host.Ports {
				if port.State == "open" {
					totalPorts++
				}
			}
		}
	}
	scanNode := graphrag.NewGraphNode("NetworkScan").
		WithID(fmt.Sprintf("scan-%s", attackID)).
		WithProperty("attack_id", attackID).
		WithProperty("subnet", subnet).
		WithProperty("timestamp", time.Now()).
		WithProperty("host_count", len(liveHosts)).
		WithProperty("port_count", totalPorts).
		WithContent(fmt.Sprintf("Scan of %s: %d hosts, %d ports", subnet, len(liveHosts), totalPorts))
	nodes = append(nodes, *scanNode)

	// Attack -> PERFORMED -> Scan
	relationships = append(relationships, *graphrag.NewRelationship(attackNode.ID, scanNode.ID, "PERFORMED"))

	// 3. Host nodes and Port nodes
	if scan != nil {
		for _, host := range scan.Hosts {
			hostNode := graphrag.NewGraphNode("Host").
				WithID(fmt.Sprintf("host-%s-%s", attackID, host.IP)).
				WithProperty("attack_id", attackID).
				WithProperty("ip", host.IP).
				WithProperty("hostname", host.Hostname).
				WithProperty("status", host.Status).
				WithContent(fmt.Sprintf("Host %s (%s)", host.IP, host.Status))
			nodes = append(nodes, *hostNode)

			// Scan -> DISCOVERED -> Host
			relationships = append(relationships, *graphrag.NewRelationship(scanNode.ID, hostNode.ID, "DISCOVERED"))

			h.Logger().Info("Created host node",
				"host_id", hostNode.ID,
				"ip", host.IP,
			)

			// Port nodes
			for _, port := range host.Ports {
				if port.State == "open" {
					portNode := graphrag.NewGraphNode("Port").
						WithID(fmt.Sprintf("port-%s-%s-%d", attackID, host.IP, port.Port)).
						WithProperty("attack_id", attackID).
						WithProperty("port", port.Port).
						WithProperty("protocol", port.Protocol).
						WithProperty("service", port.Service).
						WithProperty("version", port.Version).
						WithProperty("product", port.Product).
						WithContent(fmt.Sprintf("%d/%s: %s %s", port.Port, port.Protocol, port.Service, port.Version))
					nodes = append(nodes, *portNode)

					// Host -> HAS_PORT -> Port
					relationships = append(relationships, *graphrag.NewRelationship(hostNode.ID, portNode.ID, "HAS_PORT"))

					h.Logger().Info("Created port node",
						"port_id", portNode.ID,
						"port", port.Port,
						"service", port.Service,
					)
				}
			}
		}
	}

	// Store batch
	batch := &graphrag.Batch{
		Nodes:         nodes,
		Relationships: relationships,
	}

	h.Logger().Info("Storing graph batch",
		"nodes", len(nodes),
		"relationships", len(relationships),
	)

	nodeIDs, err := h.StoreGraphBatch(ctx, *batch)
	if err != nil {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to store graph batch: %v", err), err),
		}
	}

	duration := time.Since(startTime)
	h.Logger().Info("Graph storage completed",
		"duration", duration,
		"nodes_created", len(nodeIDs),
		"relationships_created", len(relationships),
	)

	return []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Graph: %d nodes, %d relationships stored", len(nodeIDs), len(relationships))),
	}
}

// ============================================================================
// Phase 6: Per-host LLM analysis via Claude (visible in Langfuse)
// ============================================================================

func (m *ComprehensiveSDKModule) llmPhase(ctx context.Context, h agent.Harness, scan *ScanResults) ([]*HostAnalysis, []runner.TestResult) {
	testName := "Per-Host LLM Analysis"
	reqID := "NR-6"
	startTime := time.Now()

	h.Logger().Info("Phase 6: Starting per-host LLM analysis via Claude")

	if scan == nil || len(scan.Hosts) == 0 {
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"No scan results for LLM analysis"),
		}
	}

	hostAnalyses := []*HostAnalysis{}
	analysisCount := 0

	for _, host := range scan.Hosts {
		if len(host.Ports) == 0 {
			h.Logger().Info("Skipping host with no open ports", "ip", host.IP)
			continue
		}

		h.Logger().Info("Analyzing host with Claude",
			"ip", host.IP,
			"hostname", host.Hostname,
			"port_count", len(host.Ports),
		)

		// Build per-host analysis prompt
		prompt := buildHostAnalysisPrompt(host)

		// Call Claude via LLM harness
		messages := []llm.Message{
			{
				Role:    llm.RoleUser,
				Content: prompt,
			},
		}

		response, err := h.Complete(ctx, "primary", messages)
		if err != nil {
			h.Logger().Warn("LLM analysis failed for host",
				"ip", host.IP,
				"error", err,
			)
			continue
		}

		h.Logger().Info("Claude response received",
			"ip", host.IP,
			"response_length", len(response.Content),
			"tokens_used", response.Usage.TotalTokens,
		)

		// Parse LLM response
		analysis, err := parseHostAnalysisResponse(response.Content, host.IP, host.Hostname)
		if err != nil {
			h.Logger().Warn("Failed to parse LLM response",
				"ip", host.IP,
				"error", err,
			)
			continue
		}

		hostAnalyses = append(hostAnalyses, analysis)
		analysisCount++

		h.Logger().Info("Host analysis complete",
			"ip", host.IP,
			"purpose", analysis.Purpose,
			"os", analysis.OperatingSystem,
			"risk", analysis.RiskLevel,
		)
	}

	duration := time.Since(startTime)
	h.Logger().Info("LLM analysis phase completed",
		"duration", duration,
		"hosts_analyzed", analysisCount,
	)

	return hostAnalyses, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("LLM analyzed %d hosts via Claude", analysisCount)),
	}
}

// ============================================================================
// Phase 7: Store analyses in graph
// ============================================================================

func (m *ComprehensiveSDKModule) storeAnalysesInGraph(ctx context.Context, h agent.Harness, analyses []*HostAnalysis) []runner.TestResult {
	testName := "Store Analyses in Graph"
	reqID := "NR-7"
	startTime := time.Now()

	if len(analyses) == 0 {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"No analyses to store"),
		}
	}

	h.Logger().Info("Phase 7: Storing LLM analyses in graph")

	mission := h.Mission()
	if mission.ID == "" {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Mission context not available"),
		}
	}

	attackID := mission.ID
	nodes := []graphrag.GraphNode{}
	relationships := []graphrag.Relationship{}

	for _, analysis := range analyses {
		// Create analysis node
		analysisNode := graphrag.NewGraphNode("HostAnalysis").
			WithID(fmt.Sprintf("analysis-%s-%s", attackID, analysis.IP)).
			WithProperty("attack_id", attackID).
			WithProperty("ip", analysis.IP).
			WithProperty("purpose", analysis.Purpose).
			WithProperty("operating_system", analysis.OperatingSystem).
			WithProperty("risk_level", analysis.RiskLevel).
			WithProperty("vulnerabilities", analysis.Vulnerabilities).
			WithProperty("recommendations", analysis.Recommendations).
			WithContent(fmt.Sprintf("%s: %s (%s) - Risk: %s", analysis.IP, analysis.Purpose, analysis.OperatingSystem, analysis.RiskLevel))
		nodes = append(nodes, *analysisNode)

		// Link to host node: Host -> ANALYZED_AS -> HostAnalysis
		hostID := fmt.Sprintf("host-%s-%s", attackID, analysis.IP)
		relationships = append(relationships, *graphrag.NewRelationship(hostID, analysisNode.ID, "ANALYZED_AS"))

		h.Logger().Info("Created analysis node",
			"analysis_id", analysisNode.ID,
			"purpose", analysis.Purpose,
		)
	}

	batch := &graphrag.Batch{
		Nodes:         nodes,
		Relationships: relationships,
	}

	nodeIDs, err := h.StoreGraphBatch(ctx, *batch)
	if err != nil {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to store analysis nodes: %v", err), err),
		}
	}

	// Also store in working memory
	mem := h.Memory()
	if mem != nil && mem.Working() != nil {
		if err := mem.Working().Set(ctx, "host_analyses", analyses); err != nil {
			h.Logger().Warn("Failed to store analyses in memory", "error", err)
		}
	}

	duration := time.Since(startTime)
	h.Logger().Info("Analysis storage completed",
		"nodes_created", len(nodeIDs),
		"relationships_created", len(relationships),
	)

	return []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Stored %d host analyses in graph and memory", len(analyses))),
	}
}

// ============================================================================
// Phase 8: Submit findings for discovered vulnerabilities
// ============================================================================

func (m *ComprehensiveSDKModule) findingsPhase(ctx context.Context, h agent.Harness, scan *ScanResults, analyses []*HostAnalysis) []runner.TestResult {
	testName := "Submit Security Findings"
	reqID := "NR-8"
	startTime := time.Now()

	h.Logger().Info("Phase 8: Submitting security findings")

	mission := h.Mission()
	if mission.ID == "" {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Mission context not available for finding submission"),
		}
	}

	findingsSubmitted := 0

	// Submit findings for each host analysis with vulnerabilities
	for _, analysis := range analyses {
		if len(analysis.Vulnerabilities) == 0 {
			continue
		}

		// Determine severity from LLM risk level
		var severity finding.Severity
		switch strings.ToLower(analysis.RiskLevel) {
		case "critical":
			severity = finding.SeverityCritical
		case "high":
			severity = finding.SeverityHigh
		case "medium":
			severity = finding.SeverityMedium
		default:
			severity = finding.SeverityLow
		}

		// Create a finding for each vulnerability
		for _, vuln := range analysis.Vulnerabilities {
			f := finding.NewFinding(
				mission.ID,
				"debug-agent",
				fmt.Sprintf("Potential Vulnerability on %s: %s", analysis.IP, truncateString(vuln, 50)),
				fmt.Sprintf("Host: %s (%s)\nPurpose: %s\nOS: %s\n\nVulnerability: %s",
					analysis.IP, analysis.Hostname, analysis.Purpose, analysis.OperatingSystem, vuln),
				finding.CategoryInformationDisclosure,
				severity,
			)

			// Add MITRE ATT&CK mapping for network service scanning
			f.SetMitreAttack(&finding.MitreMapping{
				Matrix:        "enterprise",
				TacticID:      "TA0007",
				TacticName:    "Discovery",
				TechniqueID:   "T1046",
				TechniqueName: "Network Service Scanning",
			})

			// Add evidence from scan
			if scan != nil {
				for _, host := range scan.Hosts {
					if host.IP == analysis.IP {
						var portInfo strings.Builder
						for _, port := range host.Ports {
							if port.State == "open" {
								portInfo.WriteString(fmt.Sprintf("  - %d/%s: %s %s\n",
									port.Port, port.Protocol, port.Service, port.Version))
							}
						}
						f.AddEvidence(finding.Evidence{
							Type:      finding.EvidenceLog,
							Title:     "Port Scan Results",
							Content:   fmt.Sprintf("Open ports on %s:\n%s", analysis.IP, portInfo.String()),
							Timestamp: scan.ScanTime,
						})
						break
					}
				}
			}

			// Add LLM analysis as evidence
			f.AddEvidence(finding.Evidence{
				Type:  finding.EvidenceLog,
				Title: "LLM Security Analysis",
				Content: fmt.Sprintf("Purpose: %s\nOperating System: %s\nRisk Level: %s\n\nRecommendations:\n%s",
					analysis.Purpose, analysis.OperatingSystem, analysis.RiskLevel,
					formatRecommendations(analysis.Recommendations)),
				Timestamp: time.Now(),
			})

			// Add remediation from recommendations
			if len(analysis.Recommendations) > 0 {
				f.Remediation = formatRecommendations(analysis.Recommendations)
			}

			// Add tags
			f.AddTag("network-recon")
			f.AddTag("automated-scan")
			f.AddTag(fmt.Sprintf("host:%s", analysis.IP))

			// Submit the finding
			if err := h.SubmitFinding(ctx, f); err != nil {
				h.Logger().Warn("Failed to submit finding",
					"host", analysis.IP,
					"vulnerability", truncateString(vuln, 30),
					"error", err,
				)
			} else {
				findingsSubmitted++
				h.Logger().Info("Finding submitted",
					"finding_id", f.ID,
					"host", analysis.IP,
					"severity", severity,
					"vulnerability", truncateString(vuln, 30),
				)
			}
		}
	}

	duration := time.Since(startTime)

	if findingsSubmitted == 0 {
		return []runner.TestResult{
			runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
				"No vulnerabilities found to report (this is good!)"),
		}
	}

	h.Logger().Info("Findings submission completed",
		"findings_submitted", findingsSubmitted,
		"duration", duration,
	)

	return []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Submitted %d security findings", findingsSubmitted)),
	}
}

// formatRecommendations formats recommendations as a numbered list
func formatRecommendations(recommendations []string) string {
	var result strings.Builder
	for i, rec := range recommendations {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
	}
	return result.String()
}

// truncateString truncates a string to maxLen characters with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ============================================================================
// Helper functions
// ============================================================================

// buildHostAnalysisPrompt creates an LLM prompt for analyzing a single host
func buildHostAnalysisPrompt(host HostResult) string {
	const promptTemplate = `You are a network security analyst. Analyze this host and determine what it likely does.

## Host Information
- IP Address: {{.IP}}
{{if .Hostname}}- Hostname: {{.Hostname}}{{end}}
- Status: {{.Status}}

## Open Ports and Services
{{range .Ports}}
- Port {{.Port}}/{{.Protocol}}: {{.Service}}{{if .Version}} (version: {{.Version}}){{end}}{{if .Product}} [{{.Product}}]{{end}}
{{end}}

## Analysis Required
Based on the open ports and services, determine:
1. What is the primary PURPOSE of this machine? (e.g., "Web Server", "Database Server", "Domain Controller", "File Server", "Development Machine")
2. What OPERATING SYSTEM is it likely running? (e.g., "Linux/Ubuntu", "Windows Server 2019", "macOS")
3. What is the RISK LEVEL? (low, medium, high, critical)
4. What VULNERABILITIES might exist based on the services?
5. What are your RECOMMENDATIONS for securing this host?

Respond in JSON format:
{
  "purpose": "...",
  "operating_system": "...",
  "risk_level": "low|medium|high|critical",
  "vulnerabilities": ["...", "..."],
  "recommendations": ["...", "..."]
}`

	tmpl, err := template.New("host").Parse(promptTemplate)
	if err != nil {
		return fmt.Sprintf("Analyze host %s with ports: %v", host.IP, host.Ports)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, host); err != nil {
		return fmt.Sprintf("Analyze host %s with ports: %v", host.IP, host.Ports)
	}

	return buf.String()
}

// parseHostAnalysisResponse parses the LLM JSON response into HostAnalysis
func parseHostAnalysisResponse(response, ip, hostname string) (*HostAnalysis, error) {
	// Find JSON in response
	jsonStart := -1
	jsonEnd := -1
	for i := 0; i < len(response); i++ {
		if response[i] == '{' && jsonStart == -1 {
			jsonStart = i
		}
		if response[i] == '}' {
			jsonEnd = i + 1
		}
	}

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart:jsonEnd]

	var analysis HostAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	analysis.IP = ip
	analysis.Hostname = hostname

	return &analysis, nil
}

// enumerateIPs generates all IP addresses from a CIDR range
func enumerateIPs(cidr string) ([]string, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	var ips []string
	for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network and broadcast addresses for subnets larger than /31
	ones, _ := ipNet.Mask.Size()
	if ones < 31 && len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

// incIP increments an IP address in place
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// getMapKeys returns the keys of a map as a slice
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

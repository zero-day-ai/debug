package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/finding"
	"github.com/zero-day-ai/sdk/graphrag"
	"github.com/zero-day-ai/sdk/llm"

	"github.com/zero-day-ai/agents/debug/internal/runner"
	"github.com/zero-day-ai/agents/debug/internal/sdk"
)

// NetworkReconModule tests network reconnaissance capabilities
type NetworkReconModule struct {
	sdk.BaseModule
}

// NewNetworkReconModule creates the network recon test module
func NewNetworkReconModule() *NetworkReconModule {
	return &NetworkReconModule{
		BaseModule: sdk.NewBaseModule(
			"network-recon",
			"Network reconnaissance tests: ping sweep, nmap scan, GraphRAG storage, LLM analysis, finding submission",
			"NR-1..NR-8",
		),
	}
}

// Category returns the test category
func (m *NetworkReconModule) Category() runner.Category {
	return runner.CategorySDK
}

// Run executes all network recon tests
func (m *NetworkReconModule) Run(ctx context.Context, h agent.Harness) []runner.TestResult {
	results := []runner.TestResult{}

	// Phase 1: Parse subnet from target
	subnet, parseResults := m.parseSubnet(ctx, h)
	results = append(results, parseResults...)
	if subnet == "" {
		return results // Cannot continue without subnet
	}

	// Phase 2: Ping sweep
	liveHosts, pingResults := m.pingPhase(ctx, h, subnet)
	results = append(results, pingResults...)

	// Phase 3: Nmap scan
	scanResults, nmapResults := m.nmapPhase(ctx, h, liveHosts)
	results = append(results, nmapResults...)

	// Phase 4: GraphRAG storage
	graphResults := m.graphPhase(ctx, h, subnet, liveHosts, scanResults)
	results = append(results, graphResults...)

	// Phase 5: LLM analysis
	analysis, llmResults := m.llmPhase(ctx, h, subnet, liveHosts, scanResults)
	results = append(results, llmResults...)

	// Phase 6: Finding submission
	findingResults := m.findingPhase(ctx, h, analysis, scanResults)
	results = append(results, findingResults...)

	return results
}

// parseSubnet extracts and validates CIDR from target connection
func (m *NetworkReconModule) parseSubnet(ctx context.Context, h agent.Harness) (string, []runner.TestResult) {
	testName := "Parse Subnet from Target"
	reqID := "NR-1"

	target := h.Target()
	if target.ID == "" {
		return "", []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Target info not available"),
		}
	}

	// Check if subnet exists in connection config
	subnetRaw, ok := target.Connection["subnet"]
	if !ok {
		return "", []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Target connection does not contain 'subnet' field. Add subnet to target connection config: {\"subnet\": \"192.168.1.0/24\"}"),
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
				fmt.Sprintf("Invalid CIDR notation: %s (error: %v)", subnet, err),
				err),
		}
	}

	// Success - subnet is valid
	return subnet, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, 0,
			fmt.Sprintf("Successfully parsed subnet: %s (network: %s)", subnet, ipNet.String())),
	}
}

// pingPhase performs ping sweep and returns live hosts
func (m *NetworkReconModule) pingPhase(ctx context.Context, h agent.Harness, subnet string) ([]string, []runner.TestResult) {
	testName := "Ping Sweep"
	reqID := "NR-2"
	startTime := time.Now()

	// Enumerate IPs from CIDR
	ips, err := enumerateIPs(subnet)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to enumerate IPs from subnet %s", subnet), err),
		}
	}

	// Safety check: limit to reasonable subnet sizes (e.g., /24 = 256 IPs)
	if len(ips) > 256 {
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				fmt.Sprintf("Subnet too large (%d IPs). For safety, limiting to /24 or smaller subnets (max 256 IPs)", len(ips))),
		}
	}

	h.Logger().Info("Starting ping sweep",
		"subnet", subnet,
		"ip_count", len(ips),
	)

	// Call ping tool
	toolInput := map[string]interface{}{
		"targets": ips,
		"timeout": 1000, // 1 second per host
		"count":   1,    // 1 ping per host
	}

	result, err := h.CallTool(ctx, "ping", toolInput)
	if err != nil {
		// Tool not available or execution failed
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				fmt.Sprintf("Ping tool not available or failed: %v. Install ping tool to enable network reconnaissance", err)),
		}
	}

	// Parse ping results - tool output is already a map[string]any
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to marshal ping tool result: %v", err), err),
		}
	}

	var pingOutput PingToolOutput
	if err := json.Unmarshal(resultBytes, &pingOutput); err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to parse ping tool output: %v", err), err),
		}
	}

	// Extract live hosts
	liveHosts := []string{}
	for _, pingResult := range pingOutput.Results {
		if pingResult.Alive {
			liveHosts = append(liveHosts, pingResult.IP)
		}
	}

	// Store in working memory
	mem := h.Memory()
	if mem != nil && mem.Working() != nil {
		if err := mem.Working().Set(ctx, "live_hosts", liveHosts); err != nil {
			h.Logger().Warn("Failed to store live hosts in working memory", "error", err)
		}
	}

	duration := time.Since(startTime)
	h.Logger().Info("Ping sweep completed",
		"duration", duration,
		"scanned", len(ips),
		"alive", len(liveHosts),
	)

	if len(liveHosts) == 0 {
		return liveHosts, []runner.TestResult{
			runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
				fmt.Sprintf("Ping sweep completed: %d IPs scanned, no live hosts discovered", len(ips))),
		}
	}

	return liveHosts, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Ping sweep completed: %d/%d hosts alive (%.1f%%)", len(liveHosts), len(ips), float64(len(liveHosts))/float64(len(ips))*100)),
	}
}

// nmapPhase scans live hosts and returns port/service data
func (m *NetworkReconModule) nmapPhase(ctx context.Context, h agent.Harness, liveHosts []string) (*ScanResults, []runner.TestResult) {
	testName := "Nmap Port Scan"
	reqID := "NR-3"
	startTime := time.Now()

	// Skip if no live hosts
	if len(liveHosts) == 0 {
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"No live hosts to scan (skipping nmap phase)"),
		}
	}

	h.Logger().Info("Starting nmap scan",
		"host_count", len(liveHosts),
	)

	// Call nmap tool with service version detection and default scripts
	toolInput := map[string]interface{}{
		"targets": liveHosts,
		"options": "-sV -sC", // Service version detection + default scripts
	}

	result, err := h.CallTool(ctx, "nmap", toolInput)
	if err != nil {
		// Tool not available or execution failed
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				fmt.Sprintf("Nmap tool not available or failed: %v. Install nmap tool to enable port scanning", err)),
		}
	}

	// Parse nmap results - tool output is already a map[string]any
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to marshal nmap tool result: %v", err), err),
		}
	}

	var nmapOutput NmapToolOutput
	if err := json.Unmarshal(resultBytes, &nmapOutput); err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to parse nmap tool output: %v", err), err),
		}
	}

	// Parse scan time
	scanTime, err := time.Parse(time.RFC3339, nmapOutput.ScanTime)
	if err != nil {
		scanTime = time.Now()
	}

	// Build ScanResults structure
	scanResults := &ScanResults{
		Hosts:        nmapOutput.Hosts,
		ScanTime:     scanTime,
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

	// Store in working memory
	mem := h.Memory()
	if mem != nil && mem.Working() != nil {
		if err := mem.Working().Set(ctx, "scan_results", scanResults); err != nil {
			h.Logger().Warn("Failed to store scan results in working memory", "error", err)
		}
	}

	duration := time.Since(startTime)
	h.Logger().Info("Nmap scan completed",
		"duration", duration,
		"hosts_scanned", len(liveHosts),
		"hosts_with_ports", len(scanResults.Hosts),
		"total_open_ports", totalPorts,
	)

	if totalPorts == 0 {
		return scanResults, []runner.TestResult{
			runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
				fmt.Sprintf("Nmap scan completed: %d hosts scanned, no open ports discovered", len(liveHosts))),
		}
	}

	return scanResults, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Nmap scan completed: %d hosts scanned, %d open ports discovered across %d hosts",
				len(liveHosts), totalPorts, len(scanResults.Hosts))),
	}
}

// graphPhase stores all data in Neo4j
func (m *NetworkReconModule) graphPhase(ctx context.Context, h agent.Harness, subnet string, liveHosts []string, scan *ScanResults) []runner.TestResult {
	testName := "GraphRAG Storage"
	reqID := "NR-4"
	startTime := time.Now()

	// Check GraphRAG health first
	health := h.GraphRAGHealth(ctx)
	if health.Status != "healthy" {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				fmt.Sprintf("GraphRAG unavailable (status: %s): %s. Skipping graph storage", health.Status, health.Message)),
		}
	}

	// Get mission context for attack ID
	mission := h.Mission()
	if mission.ID == "" {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Mission context not available, cannot create graph correlation nodes"),
		}
	}

	attackID := mission.ID

	h.Logger().Info("Storing scan results in Neo4j",
		"attack_id", attackID,
		"subnet", subnet,
		"live_hosts", len(liveHosts),
	)

	// Build graph batch (analysis will be nil at this point, added later in llmPhase if needed)
	nodes, relationships := buildGraphBatch(attackID, subnet, liveHosts, scan, nil)

	// Create batch
	batch := &graphrag.Batch{
		Nodes:         nodes,
		Relationships: relationships,
	}

	// Store batch atomically
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
			fmt.Sprintf("Graph storage completed: %d nodes and %d relationships created in Neo4j",
				len(nodeIDs), len(relationships))),
	}
}

// llmPhase analyzes results with LLM
func (m *NetworkReconModule) llmPhase(ctx context.Context, h agent.Harness, subnet string, liveHosts []string, scan *ScanResults) (*LLMAnalysis, []runner.TestResult) {
	testName := "LLM Security Analysis"
	reqID := "NR-5"
	startTime := time.Now()

	// Skip if no scan results
	if scan == nil || len(scan.Hosts) == 0 {
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"No scan results available for LLM analysis"),
		}
	}

	h.Logger().Info("Starting LLM analysis of scan results",
		"subnet", subnet,
		"hosts", len(scan.Hosts),
	)

	// Build analysis prompt
	prompt, err := buildAnalysisPrompt(subnet, liveHosts, scan)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to build analysis prompt: %v", err), err),
		}
	}

	// Create LLM messages
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	// Call LLM using "primary" slot
	response, err := h.Complete(ctx, "primary", messages)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				fmt.Sprintf("LLM completion failed: %v. LLM analysis unavailable", err)),
		}
	}

	// Parse LLM response
	analysis, err := parseLLMResponse(response.Content)
	if err != nil {
		return nil, []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				fmt.Sprintf("Failed to parse LLM response: %v", err), err),
		}
	}

	// Store analysis in working memory
	mem := h.Memory()
	if mem != nil && mem.Working() != nil {
		if err := mem.Working().Set(ctx, "llm_analysis", analysis); err != nil {
			h.Logger().Warn("Failed to store LLM analysis in working memory", "error", err)
		}
	}

	duration := time.Since(startTime)
	h.Logger().Info("LLM analysis completed",
		"duration", duration,
		"risk_level", analysis.RiskLevel,
		"high_risk_services", len(analysis.HighRiskServices),
	)

	return analysis, []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("LLM analysis completed: Risk Level=%s, High-Risk Services=%d, Recommendations=%d",
				analysis.RiskLevel, len(analysis.HighRiskServices), len(analysis.Recommendations))),
	}
}

// findingPhase submits findings for notable discoveries
func (m *NetworkReconModule) findingPhase(ctx context.Context, h agent.Harness, analysis *LLMAnalysis, scan *ScanResults) []runner.TestResult {
	testName := "Finding Submission"
	reqID := "NR-6"
	startTime := time.Now()

	// Skip if no analysis or no high-risk services
	if analysis == nil {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"No LLM analysis available, skipping finding submission"),
		}
	}

	if len(analysis.HighRiskServices) == 0 {
		return []runner.TestResult{
			runner.NewPassResult(testName, reqID, runner.CategorySDK, time.Since(startTime),
				"No high-risk services identified, no findings warranted"),
		}
	}

	h.Logger().Info("Submitting findings for high-risk services",
		"count", len(analysis.HighRiskServices),
		"risk_level", analysis.RiskLevel,
	)

	// Get mission context
	mission := h.Mission()
	if mission.ID == "" {
		return []runner.TestResult{
			runner.NewSkipResult(testName, reqID, runner.CategorySDK,
				"Mission context not available, cannot submit findings"),
		}
	}

	findingsSubmitted := 0

	// Create findings for each high-risk service
	for _, servicePort := range analysis.HighRiskServices {
		// Determine severity based on risk level
		var severity finding.Severity
		switch analysis.RiskLevel {
		case "critical":
			severity = finding.SeverityCritical
		case "high":
			severity = finding.SeverityHigh
		case "medium":
			severity = finding.SeverityMedium
		default:
			severity = finding.SeverityLow
		}

		// Create finding
		f := finding.NewFinding(
			mission.ID,
			"debug-agent-network-recon",
			fmt.Sprintf("High-Risk Network Service Detected: %s", servicePort),
			fmt.Sprintf("Network scan discovered %s which is identified as a high-risk service. %s", servicePort, analysis.Summary),
			finding.CategoryInformationDisclosure, // Network exposure falls under information disclosure
			severity,
		)

		// Add evidence from scan results
		if scan != nil {
			scanEvidence := finding.Evidence{
				Type:      finding.EvidenceLog,
				Title:     "Network Scan Results",
				Content:   formatScanResultsForEvidence(scan),
				Timestamp: scan.ScanTime,
			}
			f.Evidence = append(f.Evidence, scanEvidence)
		}

		// Add evidence from LLM analysis
		analysisEvidence := finding.Evidence{
			Type:      finding.EvidenceLog,
			Title:     "LLM Security Analysis",
			Content:   fmt.Sprintf("Risk Level: %s\n\nSummary: %s\n\nRecommendations:\n%s", analysis.RiskLevel, analysis.Summary, formatRecommendations(analysis.Recommendations)),
			Timestamp: time.Now(),
		}
		f.Evidence = append(f.Evidence, analysisEvidence)

		// Add recommendations as remediation
		if len(analysis.Recommendations) > 0 {
			f.Remediation = formatRecommendations(analysis.Recommendations)
		}

		// Submit finding
		if err := h.SubmitFinding(ctx, f); err != nil {
			h.Logger().Warn("Failed to submit finding",
				"service", servicePort,
				"error", err,
			)
		} else {
			findingsSubmitted++
			h.Logger().Info("Finding submitted",
				"service", servicePort,
				"severity", severity,
			)
		}
	}

	duration := time.Since(startTime)

	if findingsSubmitted == 0 {
		return []runner.TestResult{
			runner.NewFailResult(testName, reqID, runner.CategorySDK, duration,
				"Failed to submit any findings", fmt.Errorf("no findings submitted")),
		}
	}

	return []runner.TestResult{
		runner.NewPassResult(testName, reqID, runner.CategorySDK, duration,
			fmt.Sprintf("Finding submission completed: %d findings submitted for high-risk services", findingsSubmitted)),
	}
}

// formatScanResultsForEvidence formats scan results as text evidence
func formatScanResultsForEvidence(scan *ScanResults) string {
	var result string
	result += fmt.Sprintf("Scan Time: %s\n", scan.ScanTime.Format("2006-01-02 15:04:05 MST"))
	result += fmt.Sprintf("Duration: %s\n\n", scan.ScanDuration)

	for _, host := range scan.Hosts {
		result += fmt.Sprintf("Host: %s", host.IP)
		if host.Hostname != "" {
			result += fmt.Sprintf(" (%s)", host.Hostname)
		}
		result += fmt.Sprintf(" [%s]\n", host.Status)

		for _, port := range host.Ports {
			result += fmt.Sprintf("  Port %d/%s: %s", port.Port, port.Protocol, port.Service)
			if port.Version != "" {
				result += fmt.Sprintf(" v%s", port.Version)
			}
			if port.Product != "" {
				result += fmt.Sprintf(" (%s)", port.Product)
			}
			result += fmt.Sprintf(" [%s]\n", port.State)
		}
		result += "\n"
	}

	return result
}

// formatRecommendations formats recommendations as a numbered list
func formatRecommendations(recommendations []string) string {
	var result string
	for i, rec := range recommendations {
		result += fmt.Sprintf("%d. %s\n", i+1, rec)
	}
	return result
}

// enumerateIPs generates all IP addresses from a CIDR range
func enumerateIPs(cidr string) ([]string, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	var ips []string
	for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		// Skip network and broadcast addresses for typical /24 networks
		// But include them for smaller subnets as they might be valid hosts
		ips = append(ips, ip.String())
	}

	// Remove network address (first) and broadcast address (last) for subnets larger than /31
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

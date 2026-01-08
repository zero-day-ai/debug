package network

import (
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/graphrag"
)

// buildNetworkScanNode creates a NetworkScan node representing the scan execution
func buildNetworkScanNode(attackID, subnet string, hostCount, portCount int) *graphrag.GraphNode {
	node := graphrag.NewGraphNode("NetworkScan").
		WithID(fmt.Sprintf("scan-%s", attackID)).
		WithProperty("attack_id", attackID).
		WithProperty("subnet", subnet).
		WithProperty("timestamp", time.Now()).
		WithProperty("host_count", hostCount).
		WithProperty("port_count", portCount).
		WithContent(fmt.Sprintf("Network scan of subnet %s discovered %d hosts with %d open ports", subnet, hostCount, portCount))

	return node
}

// buildHostNode creates a Host node representing a discovered host
func buildHostNode(attackID, ip, hostname, status string) *graphrag.GraphNode {
	nodeID := fmt.Sprintf("host-%s-%s", attackID, ip)
	content := fmt.Sprintf("Host %s (status: %s)", ip, status)
	if hostname != "" {
		content = fmt.Sprintf("Host %s (%s, status: %s)", ip, hostname, status)
	}

	node := graphrag.NewGraphNode("Host").
		WithID(nodeID).
		WithProperty("attack_id", attackID).
		WithProperty("ip", ip).
		WithProperty("hostname", hostname).
		WithProperty("status", status).
		WithContent(content)

	return node
}

// buildPortNode creates a Port node representing a discovered open port
func buildPortNode(attackID, ip string, port int, protocol, service, version string) *graphrag.GraphNode {
	nodeID := fmt.Sprintf("port-%s-%s-%d-%s", attackID, ip, port, protocol)
	content := fmt.Sprintf("Port %d/%s on %s running %s", port, protocol, ip, service)
	if version != "" {
		content = fmt.Sprintf("Port %d/%s on %s running %s version %s", port, protocol, ip, service, version)
	}

	node := graphrag.NewGraphNode("Port").
		WithID(nodeID).
		WithProperty("attack_id", attackID).
		WithProperty("port", port).
		WithProperty("protocol", protocol).
		WithProperty("service", service).
		WithProperty("version", version).
		WithContent(content)

	return node
}

// buildLLMAnalysisNode creates an LLMAnalysis node representing the security analysis
func buildLLMAnalysisNode(attackID string, analysis *LLMAnalysis) *graphrag.GraphNode {
	node := graphrag.NewGraphNode("LLMAnalysis").
		WithID(fmt.Sprintf("analysis-%s", attackID)).
		WithProperty("attack_id", attackID).
		WithProperty("summary", analysis.Summary).
		WithProperty("risk_level", analysis.RiskLevel).
		WithProperty("high_risk_services", analysis.HighRiskServices).
		WithProperty("recommendations", analysis.Recommendations).
		WithContent(fmt.Sprintf("Security Analysis: %s (Risk: %s)", analysis.Summary, analysis.RiskLevel))

	return node
}

// buildAttackCorrelationNode creates an AttackCorrelation node to link all scan artifacts
func buildAttackCorrelationNode(attackID string) *graphrag.GraphNode {
	node := graphrag.NewGraphNode("AttackCorrelation").
		WithID(fmt.Sprintf("attack-%s", attackID)).
		WithProperty("attack_id", attackID).
		WithProperty("timestamp", time.Now()).
		WithContent(fmt.Sprintf("Attack correlation node for attack %s", attackID))

	return node
}

// buildScannedHostRelationship creates a SCANNED_HOST relationship from NetworkScan to Host
func buildScannedHostRelationship(scanID, hostID string) *graphrag.Relationship {
	return graphrag.NewRelationship(scanID, hostID, "SCANNED_HOST")
}

// buildHasPortRelationship creates a HAS_PORT relationship from Host to Port
func buildHasPortRelationship(hostID, portID string) *graphrag.Relationship {
	return graphrag.NewRelationship(hostID, portID, "HAS_PORT")
}

// buildPerformedScanRelationship creates a PERFORMED_SCAN relationship from AttackCorrelation to NetworkScan
func buildPerformedScanRelationship(attackID, scanID string) *graphrag.Relationship {
	return graphrag.NewRelationship(attackID, scanID, "PERFORMED_SCAN")
}

// buildAnalyzedByRelationship creates an ANALYZED_BY relationship from AttackCorrelation to LLMAnalysis
func buildAnalyzedByRelationship(attackID, analysisID string) *graphrag.Relationship {
	return graphrag.NewRelationship(attackID, analysisID, "ANALYZED_BY")
}

// buildGraphBatch creates a complete batch of nodes and relationships for the scan
func buildGraphBatch(attackID, subnet string, liveHosts []string, scan *ScanResults, analysis *LLMAnalysis) ([]graphrag.GraphNode, []graphrag.Relationship) {
	nodes := []graphrag.GraphNode{}
	relationships := []graphrag.Relationship{}

	// Create AttackCorrelation node
	attackCorrelation := buildAttackCorrelationNode(attackID)
	nodes = append(nodes, *attackCorrelation)

	// Count total open ports
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

	// Create NetworkScan node
	scanNode := buildNetworkScanNode(attackID, subnet, len(liveHosts), totalPorts)
	nodes = append(nodes, *scanNode)

	// Create PERFORMED_SCAN relationship
	relationships = append(relationships, *buildPerformedScanRelationship(attackCorrelation.ID, scanNode.ID))

	// Create Host nodes and relationships
	if scan != nil {
		for _, host := range scan.Hosts {
			// Create Host node
			hostNode := buildHostNode(attackID, host.IP, host.Hostname, host.Status)
			nodes = append(nodes, *hostNode)

			// Create SCANNED_HOST relationship
			relationships = append(relationships, *buildScannedHostRelationship(scanNode.ID, hostNode.ID))

			// Create Port nodes and HAS_PORT relationships
			for _, port := range host.Ports {
				if port.State == "open" {
					portNode := buildPortNode(attackID, host.IP, port.Port, port.Protocol, port.Service, port.Version)
					nodes = append(nodes, *portNode)

					// Create HAS_PORT relationship
					relationships = append(relationships, *buildHasPortRelationship(hostNode.ID, portNode.ID))
				}
			}
		}
	}

	// Create LLMAnalysis node and relationship if analysis exists
	if analysis != nil {
		analysisNode := buildLLMAnalysisNode(attackID, analysis)
		nodes = append(nodes, *analysisNode)

		// Create ANALYZED_BY relationship
		relationships = append(relationships, *buildAnalyzedByRelationship(attackCorrelation.ID, analysisNode.ID))
	}

	return nodes, relationships
}

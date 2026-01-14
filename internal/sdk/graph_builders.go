package sdk

import (
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/graphrag"
)

// ============================================================================
// Graph Node Builders - Taxonomy-Compliant Node Construction
// ============================================================================
// These functions use SDK taxonomy constants to create type-safe nodes
// that comply with the canonical GraphRAG taxonomy.

// buildHostNode creates a taxonomy-compliant host node
// Uses graphrag.NodeTypeHost constant and proper property names
func buildHostNode(attackID, ip, hostname, status string) *graphrag.GraphNode {
	node := graphrag.NewNodeWithValidation(graphrag.NodeTypeHost).
		WithID(fmt.Sprintf("host-%s-%s", attackID, ip)).
		WithProperty("attack_id", attackID).
		WithProperty("ip", ip).
		WithProperty("status", status)

	if hostname != "" {
		node.WithProperty("hostname", hostname)
	}

	node.WithContent(fmt.Sprintf("Host %s (%s)", ip, status))

	return node
}

// buildPortNode creates a taxonomy-compliant port node
// IMPORTANT: Uses "number" property (not "port") per taxonomy definition
func buildPortNode(attackID, hostIP string, portNum int, protocol, service, version, product string) *graphrag.GraphNode {
	node := graphrag.NewNodeWithValidation(graphrag.NodeTypePort).
		WithID(fmt.Sprintf("port-%s-%s-%d", attackID, hostIP, portNum)).
		WithProperty("attack_id", attackID).
		WithProperty("number", portNum). // "number" per taxonomy, not "port"
		WithProperty("protocol", protocol)

	if service != "" {
		node.WithProperty("service", service)
	}
	if version != "" {
		node.WithProperty("version", version)
	}
	if product != "" {
		node.WithProperty("product", product)
	}

	node.WithContent(fmt.Sprintf("%d/%s: %s %s", portNum, protocol, service, version))

	return node
}

// buildServiceNode creates a taxonomy-compliant service node
func buildServiceNode(attackID, hostIP string, portNum int, serviceName, version string) *graphrag.GraphNode {
	node := graphrag.NewNodeWithValidation(graphrag.NodeTypeService).
		WithID(fmt.Sprintf("service-%s-%s-%d-%s", attackID, hostIP, portNum, serviceName)).
		WithProperty("attack_id", attackID).
		WithProperty("name", serviceName)

	if version != "" {
		node.WithProperty("version", version)
	}

	node.WithContent(fmt.Sprintf("Service: %s %s", serviceName, version))

	return node
}

// buildAgentRunNode creates a taxonomy-compliant agent_run node
// Represents a single execution run of an agent within a mission
func buildAgentRunNode(attackID, agentName string, startTime time.Time) *graphrag.GraphNode {
	node := graphrag.NewNodeWithValidation(graphrag.NodeTypeAgentRun).
		WithID(fmt.Sprintf("agent-run-%s", attackID)).
		WithProperty("attack_id", attackID).
		WithProperty("agent", agentName).
		WithProperty("start_time", startTime).
		WithProperty("timestamp", startTime).
		WithContent(fmt.Sprintf("Agent run: %s (%s)", agentName, attackID))

	return node
}

// buildToolExecutionNode creates a taxonomy-compliant tool_execution node
// Represents execution of a security tool (nmap, ping, etc.)
func buildToolExecutionNode(attackID, toolName, target string, startTime time.Time) *graphrag.GraphNode {
	node := graphrag.NewNodeWithValidation(graphrag.NodeTypeToolExecution).
		WithID(fmt.Sprintf("tool-exec-%s-%s-%s", attackID, toolName, target)).
		WithProperty("attack_id", attackID).
		WithProperty("tool", toolName).
		WithProperty("target", target).
		WithProperty("start_time", startTime).
		WithProperty("timestamp", startTime).
		WithContent(fmt.Sprintf("Tool execution: %s on %s", toolName, target))

	return node
}

// ============================================================================
// Graph Relationship Builders - Taxonomy-Compliant Relationship Construction
// ============================================================================
// These functions use SDK taxonomy constants to create type-safe relationships
// that comply with the canonical GraphRAG taxonomy.

// buildDiscoveredRel creates a DISCOVERED relationship (agent_run -> asset)
// Used when an agent run discovers a host, service, or other asset
func buildDiscoveredRel(fromID, toID string) *graphrag.Relationship {
	return graphrag.NewRelationshipWithValidation(fromID, toID, graphrag.RelTypeDiscovered)
}

// buildHasPortRel creates a HAS_PORT relationship (host -> port)
// Links a host to one of its open ports
func buildHasPortRel(hostID, portID string) *graphrag.Relationship {
	return graphrag.NewRelationshipWithValidation(hostID, portID, graphrag.RelTypeHasPort)
}

// buildRunsServiceRel creates a RUNS_SERVICE relationship (port -> service)
// Links a port to the service running on it
func buildRunsServiceRel(portID, serviceID string) *graphrag.Relationship {
	return graphrag.NewRelationshipWithValidation(portID, serviceID, graphrag.RelTypeRunsService)
}

// buildPartOfRel creates a PART_OF relationship (agent_run -> mission)
// Links an agent run to its parent mission
func buildPartOfRel(agentRunID, missionID string) *graphrag.Relationship {
	return graphrag.NewRelationshipWithValidation(agentRunID, missionID, graphrag.RelTypePartOf)
}

// buildExecutedByRel creates an EXECUTED_BY relationship (tool_execution -> agent_run)
// Links a tool execution to the agent run that performed it
func buildExecutedByRel(toolExecID, agentRunID string) *graphrag.Relationship {
	return graphrag.NewRelationshipWithValidation(toolExecID, agentRunID, graphrag.RelTypeExecutedBy)
}

package sdk

import (
	"testing"
	"time"

	"github.com/zero-day-ai/sdk/graphrag"
)

// ============================================================================
// Node Builder Tests
// ============================================================================

func TestBuildHostNode(t *testing.T) {
	attackID := "attack-123"
	ip := "192.168.1.100"
	hostname := "webserver.local"
	status := "up"

	node := buildHostNode(attackID, ip, hostname, status)

	// Test node type uses taxonomy constant
	if node.Type != graphrag.NodeTypeHost {
		t.Errorf("Expected node type %q, got %q", graphrag.NodeTypeHost, node.Type)
	}

	// Test ID format
	expectedID := "host-attack-123-192.168.1.100"
	if node.ID != expectedID {
		t.Errorf("Expected ID %q, got %q", expectedID, node.ID)
	}

	// Test properties
	if node.Properties["attack_id"] != attackID {
		t.Errorf("Expected attack_id %q, got %v", attackID, node.Properties["attack_id"])
	}
	if node.Properties["ip"] != ip {
		t.Errorf("Expected ip %q, got %v", ip, node.Properties["ip"])
	}
	if node.Properties["hostname"] != hostname {
		t.Errorf("Expected hostname %q, got %v", hostname, node.Properties["hostname"])
	}
	if node.Properties["status"] != status {
		t.Errorf("Expected status %q, got %v", status, node.Properties["status"])
	}

	// Test content
	if node.Content == "" {
		t.Error("Expected non-empty content")
	}
}

func TestBuildHostNodeWithoutHostname(t *testing.T) {
	attackID := "attack-456"
	ip := "10.0.0.1"
	hostname := ""
	status := "down"

	node := buildHostNode(attackID, ip, hostname, status)

	// Hostname should not be set if empty
	if _, exists := node.Properties["hostname"]; exists {
		t.Error("Expected hostname property to not exist when empty string provided")
	}
}

func TestBuildPortNode(t *testing.T) {
	attackID := "attack-123"
	hostIP := "192.168.1.100"
	portNum := 443
	protocol := "tcp"
	service := "https"
	version := "Apache/2.4.41"
	product := "Apache httpd"

	node := buildPortNode(attackID, hostIP, portNum, protocol, service, version, product)

	// Test node type uses taxonomy constant
	if node.Type != graphrag.NodeTypePort {
		t.Errorf("Expected node type %q, got %q", graphrag.NodeTypePort, node.Type)
	}

	// Test ID format
	expectedID := "port-attack-123-192.168.1.100-443"
	if node.ID != expectedID {
		t.Errorf("Expected ID %q, got %q", expectedID, node.ID)
	}

	// CRITICAL: Test that "number" property is used (not "port")
	// This is the taxonomy requirement
	if _, exists := node.Properties["number"]; !exists {
		t.Error("Expected 'number' property to exist (taxonomy requirement)")
	}
	if node.Properties["number"] != portNum {
		t.Errorf("Expected number %d, got %v", portNum, node.Properties["number"])
	}

	// Ensure "port" property is NOT used (common mistake)
	if _, exists := node.Properties["port"]; exists {
		t.Error("Property 'port' should NOT exist - use 'number' per taxonomy")
	}

	// Test other properties
	if node.Properties["protocol"] != protocol {
		t.Errorf("Expected protocol %q, got %v", protocol, node.Properties["protocol"])
	}
	if node.Properties["service"] != service {
		t.Errorf("Expected service %q, got %v", service, node.Properties["service"])
	}
	if node.Properties["version"] != version {
		t.Errorf("Expected version %q, got %v", version, node.Properties["version"])
	}
	if node.Properties["product"] != product {
		t.Errorf("Expected product %q, got %v", product, node.Properties["product"])
	}
}

func TestBuildPortNodeMinimal(t *testing.T) {
	attackID := "attack-789"
	hostIP := "10.0.0.5"
	portNum := 22
	protocol := "tcp"

	node := buildPortNode(attackID, hostIP, portNum, protocol, "", "", "")

	// Should still have "number" property even without service/version/product
	if node.Properties["number"] != portNum {
		t.Errorf("Expected number %d, got %v", portNum, node.Properties["number"])
	}

	// Optional fields should not be present if empty
	if val, exists := node.Properties["service"]; exists && val == "" {
		t.Error("Empty service should not be set as property")
	}
}

func TestBuildServiceNode(t *testing.T) {
	attackID := "attack-123"
	hostIP := "192.168.1.100"
	portNum := 3306
	serviceName := "mysql"
	version := "8.0.32"

	node := buildServiceNode(attackID, hostIP, portNum, serviceName, version)

	// Test node type uses taxonomy constant
	if node.Type != graphrag.NodeTypeService {
		t.Errorf("Expected node type %q, got %q", graphrag.NodeTypeService, node.Type)
	}

	// Test properties
	if node.Properties["name"] != serviceName {
		t.Errorf("Expected name %q, got %v", serviceName, node.Properties["name"])
	}
	if node.Properties["version"] != version {
		t.Errorf("Expected version %q, got %v", version, node.Properties["version"])
	}
}

func TestBuildAgentRunNode(t *testing.T) {
	attackID := "attack-123"
	agentName := "debug-agent"
	startTime := time.Now()

	node := buildAgentRunNode(attackID, agentName, startTime)

	// Test node type uses taxonomy constant
	if node.Type != graphrag.NodeTypeAgentRun {
		t.Errorf("Expected node type %q, got %q", graphrag.NodeTypeAgentRun, node.Type)
	}

	// Test ID format
	expectedID := "agent-run-attack-123"
	if node.ID != expectedID {
		t.Errorf("Expected ID %q, got %q", expectedID, node.ID)
	}

	// Test properties
	if node.Properties["agent"] != agentName {
		t.Errorf("Expected agent %q, got %v", agentName, node.Properties["agent"])
	}
	if node.Properties["start_time"] != startTime {
		t.Errorf("Expected start_time %v, got %v", startTime, node.Properties["start_time"])
	}
}

func TestBuildToolExecutionNode(t *testing.T) {
	attackID := "attack-123"
	toolName := "nmap"
	target := "192.168.1.0/24"
	startTime := time.Now()

	node := buildToolExecutionNode(attackID, toolName, target, startTime)

	// Test node type uses taxonomy constant
	if node.Type != graphrag.NodeTypeToolExecution {
		t.Errorf("Expected node type %q, got %q", graphrag.NodeTypeToolExecution, node.Type)
	}

	// Test ID format
	expectedID := "tool-exec-attack-123-nmap-192.168.1.0/24"
	if node.ID != expectedID {
		t.Errorf("Expected ID %q, got %q", expectedID, node.ID)
	}

	// Test properties
	if node.Properties["tool"] != toolName {
		t.Errorf("Expected tool %q, got %v", toolName, node.Properties["tool"])
	}
	if node.Properties["target"] != target {
		t.Errorf("Expected target %q, got %v", target, node.Properties["target"])
	}
}

// ============================================================================
// Relationship Builder Tests
// ============================================================================

func TestBuildDiscoveredRel(t *testing.T) {
	fromID := "agent-run-123"
	toID := "host-123-192.168.1.1"

	rel := buildDiscoveredRel(fromID, toID)

	// Test relationship type uses taxonomy constant
	if rel.Type != graphrag.RelTypeDiscovered {
		t.Errorf("Expected relationship type %q, got %q", graphrag.RelTypeDiscovered, rel.Type)
	}

	// Test IDs
	if rel.FromID != fromID {
		t.Errorf("Expected FromID %q, got %q", fromID, rel.FromID)
	}
	if rel.ToID != toID {
		t.Errorf("Expected ToID %q, got %q", toID, rel.ToID)
	}
}

func TestBuildHasPortRel(t *testing.T) {
	hostID := "host-123-192.168.1.1"
	portID := "port-123-192.168.1.1-80"

	rel := buildHasPortRel(hostID, portID)

	// Test relationship type uses taxonomy constant
	if rel.Type != graphrag.RelTypeHasPort {
		t.Errorf("Expected relationship type %q, got %q", graphrag.RelTypeHasPort, rel.Type)
	}

	// Test direction: host -> port
	if rel.FromID != hostID {
		t.Errorf("Expected FromID %q, got %q", hostID, rel.FromID)
	}
	if rel.ToID != portID {
		t.Errorf("Expected ToID %q, got %q", portID, rel.ToID)
	}
}

func TestBuildRunsServiceRel(t *testing.T) {
	portID := "port-123-192.168.1.1-3306"
	serviceID := "service-123-192.168.1.1-3306-mysql"

	rel := buildRunsServiceRel(portID, serviceID)

	// Test relationship type uses taxonomy constant
	if rel.Type != graphrag.RelTypeRunsService {
		t.Errorf("Expected relationship type %q, got %q", graphrag.RelTypeRunsService, rel.Type)
	}

	// Test direction: port -> service
	if rel.FromID != portID {
		t.Errorf("Expected FromID %q, got %q", portID, rel.FromID)
	}
	if rel.ToID != serviceID {
		t.Errorf("Expected ToID %q, got %q", serviceID, rel.ToID)
	}
}

func TestBuildPartOfRel(t *testing.T) {
	agentRunID := "agent-run-123"
	missionID := "mission-456"

	rel := buildPartOfRel(agentRunID, missionID)

	// Test relationship type uses taxonomy constant
	if rel.Type != graphrag.RelTypePartOf {
		t.Errorf("Expected relationship type %q, got %q", graphrag.RelTypePartOf, rel.Type)
	}

	// Test direction: agent_run -> mission
	if rel.FromID != agentRunID {
		t.Errorf("Expected FromID %q, got %q", agentRunID, rel.FromID)
	}
	if rel.ToID != missionID {
		t.Errorf("Expected ToID %q, got %q", missionID, rel.ToID)
	}
}

func TestBuildExecutedByRel(t *testing.T) {
	toolExecID := "tool-exec-123-nmap-192.168.1.0/24"
	agentRunID := "agent-run-123"

	rel := buildExecutedByRel(toolExecID, agentRunID)

	// Test relationship type uses taxonomy constant
	if rel.Type != graphrag.RelTypeExecutedBy {
		t.Errorf("Expected relationship type %q, got %q", graphrag.RelTypeExecutedBy, rel.Type)
	}

	// Test direction: tool_execution -> agent_run
	if rel.FromID != toolExecID {
		t.Errorf("Expected FromID %q, got %q", toolExecID, rel.FromID)
	}
	if rel.ToID != agentRunID {
		t.Errorf("Expected ToID %q, got %q", agentRunID, rel.ToID)
	}
}

// ============================================================================
// Integration Tests - Verify Full Graph Construction
// ============================================================================

func TestTaxonomyCompliantGraphConstruction(t *testing.T) {
	// Simulate a small scan: 1 agent run, 1 tool execution, 1 host, 1 port
	attackID := "attack-integration-test"
	agentName := "debug-agent"
	missionID := "mission-test"
	hostIP := "192.168.1.50"
	portNum := 80

	// Build nodes
	agentRun := buildAgentRunNode(attackID, agentName, time.Now())
	toolExec := buildToolExecutionNode(attackID, "nmap", "192.168.1.0/24", time.Now())
	host := buildHostNode(attackID, hostIP, "webserver", "up")
	port := buildPortNode(attackID, hostIP, portNum, "tcp", "http", "nginx/1.18", "nginx")

	// Build relationships
	partOf := buildPartOfRel(agentRun.ID, missionID)
	executedBy := buildExecutedByRel(toolExec.ID, agentRun.ID)
	discovered := buildDiscoveredRel(agentRun.ID, host.ID)
	hasPort := buildHasPortRel(host.ID, port.ID)

	// Verify all node types are taxonomy-compliant
	nodeTypes := []string{agentRun.Type, toolExec.Type, host.Type, port.Type}
	expectedTypes := []string{
		graphrag.NodeTypeAgentRun,
		graphrag.NodeTypeToolExecution,
		graphrag.NodeTypeHost,
		graphrag.NodeTypePort,
	}

	for i, nodeType := range nodeTypes {
		if nodeType != expectedTypes[i] {
			t.Errorf("Node %d: expected type %q, got %q", i, expectedTypes[i], nodeType)
		}
	}

	// Verify all relationship types are taxonomy-compliant
	relTypes := []string{partOf.Type, executedBy.Type, discovered.Type, hasPort.Type}
	expectedRelTypes := []string{
		graphrag.RelTypePartOf,
		graphrag.RelTypeExecutedBy,
		graphrag.RelTypeDiscovered,
		graphrag.RelTypeHasPort,
	}

	for i, relType := range relTypes {
		if relType != expectedRelTypes[i] {
			t.Errorf("Relationship %d: expected type %q, got %q", i, expectedRelTypes[i], relType)
		}
	}

	// Verify port uses "number" property (critical taxonomy requirement)
	if _, exists := port.Properties["number"]; !exists {
		t.Fatal("Port node must have 'number' property per taxonomy")
	}
	if port.Properties["number"] != portNum {
		t.Errorf("Expected port number %d, got %v", portNum, port.Properties["number"])
	}
}

// ============================================================================
// Negative Tests - Ensure We Don't Use Non-Canonical Types
// ============================================================================

func TestDoesNotCreateNonCanonicalNodes(t *testing.T) {
	// Verify none of our builder functions create non-canonical types
	// These node types should NEVER be created (not in taxonomy):
	// - "attack" (should be agent_run)
	// - "network_scan" (should be tool_execution)
	// - "host_analysis" (should be properties on host)

	agentRun := buildAgentRunNode("test", "test", time.Now())
	toolExec := buildToolExecutionNode("test", "test", "test", time.Now())

	if agentRun.Type == "attack" {
		t.Error("buildAgentRunNode should not create 'attack' type - use agent_run")
	}
	if toolExec.Type == "network_scan" {
		t.Error("buildToolExecutionNode should not create 'network_scan' type - use tool_execution")
	}

	// Verify our types are lowercase (taxonomy convention)
	if agentRun.Type != "agent_run" {
		t.Errorf("Expected lowercase type 'agent_run', got %q", agentRun.Type)
	}
	if toolExec.Type != "tool_execution" {
		t.Errorf("Expected lowercase type 'tool_execution', got %q", toolExec.Type)
	}
}

func TestDoesNotUseWrongPropertyNames(t *testing.T) {
	// Test that port node does NOT use "port" property (should be "number")
	port := buildPortNode("test", "10.0.0.1", 443, "tcp", "https", "", "")

	if _, exists := port.Properties["port"]; exists {
		t.Error("Port node should NOT have 'port' property - use 'number' per taxonomy")
	}

	if _, exists := port.Properties["number"]; !exists {
		t.Error("Port node MUST have 'number' property per taxonomy")
	}
}

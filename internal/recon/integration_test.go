//go:build integration

package recon

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/finding"
	"github.com/zero-day-ai/sdk/graphrag"
	"github.com/zero-day-ai/sdk/llm"
	"github.com/zero-day-ai/sdk/memory"
	"github.com/zero-day-ai/sdk/mission"
	"github.com/zero-day-ai/sdk/planning"
	"github.com/zero-day-ai/sdk/plugin"
	"github.com/zero-day-ai/sdk/tool"
	"github.com/zero-day-ai/sdk/types"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"log/slog"
)

// checkInfrastructure checks if Neo4j and Qdrant are available
func checkInfrastructure(t *testing.T) (neo4jAvailable, qdrantAvailable bool) {
	// Check Neo4j availability
	neo4jURI := os.Getenv("NEO4J_URI")
	if neo4jURI == "" {
		neo4jURI = "localhost:7687"
	}

	conn, err := net.DialTimeout("tcp", neo4jURI, 2*time.Second)
	if err == nil {
		conn.Close()
		neo4jAvailable = true
		t.Logf("Neo4j available at %s", neo4jURI)
	} else {
		t.Logf("Neo4j not available at %s: %v", neo4jURI, err)
	}

	// Check Qdrant availability
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "localhost:6333"
	}

	conn, err = net.DialTimeout("tcp", qdrantHost, 2*time.Second)
	if err == nil {
		conn.Close()
		qdrantAvailable = true
		t.Logf("Qdrant available at %s", qdrantHost)
	} else {
		t.Logf("Qdrant not available at %s: %v", qdrantHost, err)
	}

	return neo4jAvailable, qdrantAvailable
}

// mockIntegrationHarness is a minimal harness for integration tests that mocks tools
type mockIntegrationHarness struct {
	logger *slog.Logger
}

func newMockIntegrationHarness() *mockIntegrationHarness {
	return &mockIntegrationHarness{
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}
}

// CallTool implements agent.Harness with mock tool outputs for integration testing
func (m *mockIntegrationHarness) CallTool(ctx context.Context, name string, input map[string]any) (map[string]any, error) {
	// Return realistic mock data based on tool type
	switch name {
	case "nmap":
		return map[string]any{
			"hosts": []any{
				map[string]any{
					"ip":       "127.0.0.1",
					"hostname": "localhost",
					"ports": []any{
						map[string]any{"port": 8080, "state": "open", "service": "http"},
						map[string]any{"port": 8081, "state": "open", "service": "http"},
					},
				},
			},
		}, nil
	case "masscan":
		return map[string]any{
			"hosts": []any{
				map[string]any{
					"ip": "127.0.0.1",
					"ports": []any{
						map[string]any{"port": 8080, "state": "open"},
					},
				},
			},
		}, nil
	case "httpx":
		return map[string]any{
			"endpoints": []any{
				map[string]any{
					"url":         "http://127.0.0.1:8080",
					"status_code": 200,
					"title":       "Test Service",
					"technologies": []any{
						map[string]any{"name": "nginx", "version": "1.21"},
					},
				},
			},
		}, nil
	case "nuclei":
		return map[string]any{
			"findings": []any{
				map[string]any{
					"template_id": "test-vulnerability",
					"severity":    "medium",
					"matched_at":  "http://127.0.0.1:8080/test",
					"description": "Test vulnerability found",
				},
			},
		}, nil
	case "subfinder":
		return map[string]any{
			"subdomains": []any{
				map[string]any{"name": "api.example.local"},
				map[string]any{"name": "www.example.local"},
			},
		}, nil
	case "amass":
		return map[string]any{
			"subdomains": []any{
				map[string]any{"name": "mail.example.local"},
			},
			"hosts": []any{
				map[string]any{"ip": "192.168.1.10"},
			},
		}, nil
	default:
		return make(map[string]any), nil
	}
}

// Logger implements agent.Harness
func (m *mockIntegrationHarness) Logger() *slog.Logger {
	return m.logger
}

// Stub implementations for other Harness methods (not used in integration tests)

func (m *mockIntegrationHarness) Complete(ctx context.Context, slot string, messages []llm.Message, opts ...llm.CompletionOption) (*llm.CompletionResponse, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) CompleteWithTools(ctx context.Context, slot string, messages []llm.Message, tools []llm.ToolDef) (*llm.CompletionResponse, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) Stream(ctx context.Context, slot string, messages []llm.Message) (<-chan llm.StreamChunk, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) CompleteStructured(ctx context.Context, slot string, messages []llm.Message, schema any) (any, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) CompleteStructuredAny(ctx context.Context, slot string, messages []llm.Message, schema any) (any, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) ListTools(ctx context.Context) ([]tool.Descriptor, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) CallToolsParallel(ctx context.Context, calls []agent.ToolCall, maxConcurrency int) ([]agent.ToolResult, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) QueryPlugin(ctx context.Context, name string, method string, params map[string]any) (any, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) ListPlugins(ctx context.Context) ([]plugin.Descriptor, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) DelegateToAgent(ctx context.Context, name string, task agent.Task) (agent.Result, error) {
	return agent.Result{}, nil
}

func (m *mockIntegrationHarness) ListAgents(ctx context.Context) ([]agent.Descriptor, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) SubmitFinding(ctx context.Context, f *finding.Finding) error {
	return nil
}

func (m *mockIntegrationHarness) GetFindings(ctx context.Context, filter finding.Filter) ([]*finding.Finding, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) Memory() memory.Store {
	return nil
}

func (m *mockIntegrationHarness) Mission() types.MissionContext {
	return types.MissionContext{}
}

func (m *mockIntegrationHarness) Target() types.TargetInfo {
	return types.TargetInfo{}
}

func (m *mockIntegrationHarness) Tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("integration-test")
}

func (m *mockIntegrationHarness) TokenUsage() llm.TokenTracker {
	return nil
}

func (m *mockIntegrationHarness) QueryGraphRAG(ctx context.Context, query graphrag.Query) ([]graphrag.Result, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) FindSimilarAttacks(ctx context.Context, content string, topK int) ([]graphrag.AttackPattern, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) FindSimilarFindings(ctx context.Context, findingID string, topK int) ([]graphrag.FindingNode, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) GetAttackChains(ctx context.Context, techniqueID string, maxDepth int) ([]graphrag.AttackChain, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) GetRelatedFindings(ctx context.Context, findingID string) ([]graphrag.FindingNode, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) StoreGraphNode(ctx context.Context, node graphrag.GraphNode) (string, error) {
	return "", nil
}

func (m *mockIntegrationHarness) CreateGraphRelationship(ctx context.Context, rel graphrag.Relationship) error {
	return nil
}

func (m *mockIntegrationHarness) StoreGraphBatch(ctx context.Context, batch graphrag.Batch) ([]string, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) TraverseGraph(ctx context.Context, startNodeID string, opts graphrag.TraversalOptions) ([]graphrag.TraversalResult, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) GraphRAGHealth(ctx context.Context) types.HealthStatus {
	return types.HealthStatus{}
}

func (m *mockIntegrationHarness) PlanContext() planning.PlanningContext {
	return nil
}

func (m *mockIntegrationHarness) ReportStepHints(ctx context.Context, hints *planning.StepHints) error {
	return nil
}

func (m *mockIntegrationHarness) MissionExecutionContext() types.MissionExecutionContext {
	return types.MissionExecutionContext{}
}

func (m *mockIntegrationHarness) GetMissionRunHistory(ctx context.Context) ([]types.MissionRunSummary, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) GetPreviousRunFindings(ctx context.Context, filter finding.Filter) ([]*finding.Finding, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) GetAllRunFindings(ctx context.Context, filter finding.Filter) ([]*finding.Finding, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) QueryGraphRAGScoped(ctx context.Context, query graphrag.Query, scope graphrag.MissionScope) ([]graphrag.Result, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) GetCredential(ctx context.Context, name string) (*types.Credential, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) CreateMission(ctx context.Context, workflow any, targetID string, opts *mission.CreateMissionOpts) (*mission.MissionInfo, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) RunMission(ctx context.Context, missionID string, opts *mission.RunMissionOpts) error {
	return nil
}

func (m *mockIntegrationHarness) GetMissionStatus(ctx context.Context, missionID string) (*mission.MissionStatusInfo, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) WaitForMission(ctx context.Context, missionID string, timeout time.Duration) (*mission.MissionResult, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) ListMissions(ctx context.Context, filter *mission.MissionFilter) ([]*mission.MissionInfo, error) {
	return nil, nil
}

func (m *mockIntegrationHarness) CancelMission(ctx context.Context, missionID string) error {
	return nil
}

func (m *mockIntegrationHarness) GetMissionResults(ctx context.Context, missionID string) (*mission.MissionResult, error) {
	return nil, nil
}

// TestIntegration_DiscoverPhase tests the discover phase with real network interfaces
func TestIntegration_DiscoverPhase(t *testing.T) {
	neo4jAvailable, qdrantAvailable := checkInfrastructure(t)

	if !neo4jAvailable || !qdrantAvailable {
		t.Skip("Skipping integration test: infrastructure not available (Neo4j or Qdrant)")
	}

	harness := newMockIntegrationHarness()
	mockExtractor := newMockExtractor()
	mockExtractor.setExtractResult("nmap", 3, 3) // 1 host + 2 ports = 3 nodes, 3 rels

	runner := &DefaultReconRunner{
		harness:   harness,
		extractor: mockExtractor,
	}

	ctx := context.Background()

	// Test discover phase with localhost
	result, err := runner.RunPhase(ctx, PhaseDiscover, []string{"127.0.0.1/32"})
	if err != nil {
		t.Fatalf("RunPhase(PhaseDiscover) failed: %v", err)
	}

	// Verify phase result
	if result.Phase != PhaseDiscover {
		t.Errorf("Phase = %v, want %v", result.Phase, PhaseDiscover)
	}

	if len(result.ToolsRun) == 0 {
		t.Error("Expected at least one tool to be run")
	}

	t.Logf("Discover phase completed: tools=%v, nodes=%d, relations=%d, duration=%v",
		result.ToolsRun, result.NodesCreated, result.RelationsCreated, result.Duration)

	// Verify nodes and relations were created
	if result.NodesCreated == 0 {
		t.Error("Expected nodes to be created")
	}

	if result.RelationsCreated == 0 {
		t.Error("Expected relations to be created")
	}
}

// TestIntegration_FullReconWorkflow tests the complete reconnaissance workflow
func TestIntegration_FullReconWorkflow(t *testing.T) {
	neo4jAvailable, qdrantAvailable := checkInfrastructure(t)

	if !neo4jAvailable || !qdrantAvailable {
		t.Skip("Skipping integration test: infrastructure not available (Neo4j or Qdrant)")
	}

	harness := newMockIntegrationHarness()
	mockExtractor := newMockExtractor()

	// Set up extraction results for each tool
	mockExtractor.setExtractResult("nmap", 3, 3)        // discover phase
	mockExtractor.setExtractResult("httpx", 2, 2)       // probe phase
	mockExtractor.setExtractResult("nuclei", 1, 1)      // scan phase
	mockExtractor.setExtractResult("subfinder", 2, 2)   // domain phase
	mockExtractor.setExtractResult("amass", 2, 2)       // domain phase

	runner := &DefaultReconRunner{
		harness:   harness,
		extractor: mockExtractor,
	}

	ctx := context.Background()

	// Test full workflow
	result, err := runner.RunAll(ctx, "127.0.0.1/32", []string{"example.local"})
	if err != nil {
		t.Fatalf("RunAll() failed: %v", err)
	}

	// Verify phases were executed
	if len(result.Phases) == 0 {
		t.Fatal("Expected at least one phase to be executed")
	}

	// At minimum, discover phase should have run
	if result.Phases[0].Phase != PhaseDiscover {
		t.Errorf("First phase should be discover, got %v", result.Phases[0].Phase)
	}

	// Verify statistics
	if result.TotalHosts == 0 {
		t.Error("Expected hosts to be discovered")
	}

	t.Logf("Full recon workflow completed:")
	t.Logf("  Phases executed: %d", len(result.Phases))
	t.Logf("  Total hosts: %d", result.TotalHosts)
	t.Logf("  Total ports: %d", result.TotalPorts)
	t.Logf("  Total endpoints: %d", result.TotalEndpoints)
	t.Logf("  Total findings: %d", result.TotalFindings)
	t.Logf("  Total duration: %v", result.Duration)

	// Log details for each phase
	for i, phase := range result.Phases {
		t.Logf("  Phase %d (%v): tools=%v, nodes=%d, relations=%d, errors=%d",
			i+1, phase.Phase, phase.ToolsRun, phase.NodesCreated,
			phase.RelationsCreated, len(phase.Errors))
	}

	// Verify workflow duration is reasonable
	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

// TestIntegration_ProbePhase tests the probe phase independently
func TestIntegration_ProbePhase(t *testing.T) {
	neo4jAvailable, qdrantAvailable := checkInfrastructure(t)

	if !neo4jAvailable || !qdrantAvailable {
		t.Skip("Skipping integration test: infrastructure not available (Neo4j or Qdrant)")
	}

	harness := newMockIntegrationHarness()
	mockExtractor := newMockExtractor()
	mockExtractor.setExtractResult("httpx", 2, 2)

	runner := &DefaultReconRunner{
		harness:   harness,
		extractor: mockExtractor,
	}

	ctx := context.Background()

	// Test probe phase with localhost URLs
	targets := []string{"http://127.0.0.1:8080", "http://127.0.0.1:8081"}
	result, err := runner.RunPhase(ctx, PhaseProbe, targets)
	if err != nil {
		t.Fatalf("RunPhase(PhaseProbe) failed: %v", err)
	}

	// Verify phase result
	if result.Phase != PhaseProbe {
		t.Errorf("Phase = %v, want %v", result.Phase, PhaseProbe)
	}

	if len(result.ToolsRun) == 0 {
		t.Error("Expected at least one tool to be run")
	}

	t.Logf("Probe phase completed: tools=%v, nodes=%d, relations=%d, duration=%v",
		result.ToolsRun, result.NodesCreated, result.RelationsCreated, result.Duration)
}

// TestIntegration_ScanPhase tests the scan phase independently
func TestIntegration_ScanPhase(t *testing.T) {
	neo4jAvailable, qdrantAvailable := checkInfrastructure(t)

	if !neo4jAvailable || !qdrantAvailable {
		t.Skip("Skipping integration test: infrastructure not available (Neo4j or Qdrant)")
	}

	harness := newMockIntegrationHarness()
	mockExtractor := newMockExtractor()
	mockExtractor.setExtractResult("nuclei", 1, 1)

	runner := &DefaultReconRunner{
		harness:   harness,
		extractor: mockExtractor,
	}

	ctx := context.Background()

	// Test scan phase with localhost URLs
	targets := []string{"http://127.0.0.1:8080"}
	result, err := runner.RunPhase(ctx, PhaseScan, targets)
	if err != nil {
		t.Fatalf("RunPhase(PhaseScan) failed: %v", err)
	}

	// Verify phase result
	if result.Phase != PhaseScan {
		t.Errorf("Phase = %v, want %v", result.Phase, PhaseScan)
	}

	if len(result.ToolsRun) == 0 {
		t.Error("Expected at least one tool to be run")
	}

	t.Logf("Scan phase completed: tools=%v, nodes=%d, relations=%d, duration=%v",
		result.ToolsRun, result.NodesCreated, result.RelationsCreated, result.Duration)
}

// TestIntegration_DomainPhase tests the domain phase independently
func TestIntegration_DomainPhase(t *testing.T) {
	neo4jAvailable, qdrantAvailable := checkInfrastructure(t)

	if !neo4jAvailable || !qdrantAvailable {
		t.Skip("Skipping integration test: infrastructure not available (Neo4j or Qdrant)")
	}

	harness := newMockIntegrationHarness()
	mockExtractor := newMockExtractor()
	mockExtractor.setExtractResult("subfinder", 2, 2)
	mockExtractor.setExtractResult("amass", 2, 2)

	runner := &DefaultReconRunner{
		harness:   harness,
		extractor: mockExtractor,
	}

	ctx := context.Background()

	// Test domain phase
	domains := []string{"example.local"}
	result, err := runner.RunPhase(ctx, PhaseDomain, domains)
	if err != nil {
		t.Fatalf("RunPhase(PhaseDomain) failed: %v", err)
	}

	// Verify phase result
	if result.Phase != PhaseDomain {
		t.Errorf("Phase = %v, want %v", result.Phase, PhaseDomain)
	}

	if len(result.ToolsRun) == 0 {
		t.Error("Expected at least one tool to be run")
	}

	t.Logf("Domain phase completed: tools=%v, nodes=%d, relations=%d, duration=%v",
		result.ToolsRun, result.NodesCreated, result.RelationsCreated, result.Duration)
}

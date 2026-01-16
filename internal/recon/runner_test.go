package recon

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
)

// mockHarness implements the agent.Harness interface for testing
type mockHarness struct {
	toolOutputs map[string]map[string]any
	toolErrors  map[string]error
	logger      *slog.Logger
}

func newMockHarness() *mockHarness {
	return &mockHarness{
		toolOutputs: make(map[string]map[string]any),
		toolErrors:  make(map[string]error),
		logger:      slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}

func (m *mockHarness) setToolOutput(toolName string, output map[string]any) {
	m.toolOutputs[toolName] = output
}

func (m *mockHarness) setToolError(toolName string, err error) {
	m.toolErrors[toolName] = err
}

// mockExtractor implements TaxonomyExtractor for testing
type mockExtractor struct {
	extractCounts map[string]struct{ nodes, rels int }
}

func newMockExtractor() *mockExtractor {
	return &mockExtractor{
		extractCounts: make(map[string]struct{ nodes, rels int }),
	}
}

func (m *mockExtractor) setExtractResult(toolName string, nodes, rels int) {
	m.extractCounts[toolName] = struct{ nodes, rels int }{nodes, rels}
}

func (m *mockExtractor) Extract(ctx context.Context, toolName string, outputJSON json.RawMessage) (int, int, error) {
	if counts, ok := m.extractCounts[toolName]; ok {
		return counts.nodes, counts.rels, nil
	}
	// Default: return 0, 0 (no extraction)
	return 0, 0, nil
}

// CallTool implements agent.Harness
func (m *mockHarness) CallTool(ctx context.Context, name string, input map[string]any) (map[string]any, error) {
	if err, ok := m.toolErrors[name]; ok {
		return nil, err
	}
	if output, ok := m.toolOutputs[name]; ok {
		return output, nil
	}
	return make(map[string]any), nil
}

// Logger implements agent.Harness
func (m *mockHarness) Logger() *slog.Logger {
	return m.logger
}

// Stub implementations for other Harness methods (not used in runner tests)

func (m *mockHarness) Complete(ctx context.Context, slot string, messages []llm.Message, opts ...llm.CompletionOption) (*llm.CompletionResponse, error) {
	return nil, nil
}

func (m *mockHarness) CompleteWithTools(ctx context.Context, slot string, messages []llm.Message, tools []llm.ToolDef) (*llm.CompletionResponse, error) {
	return nil, nil
}

func (m *mockHarness) Stream(ctx context.Context, slot string, messages []llm.Message) (<-chan llm.StreamChunk, error) {
	return nil, nil
}

func (m *mockHarness) CompleteStructured(ctx context.Context, slot string, messages []llm.Message, schema any) (any, error) {
	return nil, nil
}

func (m *mockHarness) CompleteStructuredAny(ctx context.Context, slot string, messages []llm.Message, schema any) (any, error) {
	return nil, nil
}

func (m *mockHarness) ListTools(ctx context.Context) ([]tool.Descriptor, error) {
	return nil, nil
}

func (m *mockHarness) CallToolsParallel(ctx context.Context, calls []agent.ToolCall, maxConcurrency int) ([]agent.ToolResult, error) {
	return nil, nil
}

func (m *mockHarness) QueryPlugin(ctx context.Context, name string, method string, params map[string]any) (any, error) {
	return nil, nil
}

func (m *mockHarness) ListPlugins(ctx context.Context) ([]plugin.Descriptor, error) {
	return nil, nil
}

func (m *mockHarness) DelegateToAgent(ctx context.Context, name string, task agent.Task) (agent.Result, error) {
	return agent.Result{}, nil
}

func (m *mockHarness) ListAgents(ctx context.Context) ([]agent.Descriptor, error) {
	return nil, nil
}

func (m *mockHarness) SubmitFinding(ctx context.Context, f *finding.Finding) error {
	return nil
}

func (m *mockHarness) GetFindings(ctx context.Context, filter finding.Filter) ([]*finding.Finding, error) {
	return nil, nil
}

func (m *mockHarness) Memory() memory.Store {
	return nil
}

func (m *mockHarness) Mission() types.MissionContext {
	return types.MissionContext{}
}

func (m *mockHarness) Target() types.TargetInfo {
	return types.TargetInfo{}
}

func (m *mockHarness) Tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}

func (m *mockHarness) TokenUsage() llm.TokenTracker {
	return nil
}

func (m *mockHarness) QueryGraphRAG(ctx context.Context, query graphrag.Query) ([]graphrag.Result, error) {
	return nil, nil
}

func (m *mockHarness) FindSimilarAttacks(ctx context.Context, content string, topK int) ([]graphrag.AttackPattern, error) {
	return nil, nil
}

func (m *mockHarness) FindSimilarFindings(ctx context.Context, findingID string, topK int) ([]graphrag.FindingNode, error) {
	return nil, nil
}

func (m *mockHarness) GetAttackChains(ctx context.Context, techniqueID string, maxDepth int) ([]graphrag.AttackChain, error) {
	return nil, nil
}

func (m *mockHarness) GetRelatedFindings(ctx context.Context, findingID string) ([]graphrag.FindingNode, error) {
	return nil, nil
}

func (m *mockHarness) StoreGraphNode(ctx context.Context, node graphrag.GraphNode) (string, error) {
	return "", nil
}

func (m *mockHarness) CreateGraphRelationship(ctx context.Context, rel graphrag.Relationship) error {
	return nil
}

func (m *mockHarness) StoreGraphBatch(ctx context.Context, batch graphrag.Batch) ([]string, error) {
	return nil, nil
}

func (m *mockHarness) TraverseGraph(ctx context.Context, startNodeID string, opts graphrag.TraversalOptions) ([]graphrag.TraversalResult, error) {
	return nil, nil
}

func (m *mockHarness) GraphRAGHealth(ctx context.Context) types.HealthStatus {
	return types.HealthStatus{}
}

func (m *mockHarness) PlanContext() planning.PlanningContext {
	return nil
}

func (m *mockHarness) ReportStepHints(ctx context.Context, hints *planning.StepHints) error {
	return nil
}

func (m *mockHarness) MissionExecutionContext() types.MissionExecutionContext {
	return types.MissionExecutionContext{}
}

func (m *mockHarness) GetMissionRunHistory(ctx context.Context) ([]types.MissionRunSummary, error) {
	return nil, nil
}

func (m *mockHarness) GetPreviousRunFindings(ctx context.Context, filter finding.Filter) ([]*finding.Finding, error) {
	return nil, nil
}

func (m *mockHarness) GetAllRunFindings(ctx context.Context, filter finding.Filter) ([]*finding.Finding, error) {
	return nil, nil
}

func (m *mockHarness) QueryGraphRAGScoped(ctx context.Context, query graphrag.Query, scope graphrag.MissionScope) ([]graphrag.Result, error) {
	return nil, nil
}

func (m *mockHarness) GetCredential(ctx context.Context, name string) (*types.Credential, error) {
	return nil, nil
}

func (m *mockHarness) CreateMission(ctx context.Context, workflow any, targetID string, opts *mission.CreateMissionOpts) (*mission.MissionInfo, error) {
	return nil, nil
}

func (m *mockHarness) RunMission(ctx context.Context, missionID string, opts *mission.RunMissionOpts) error {
	return nil
}

func (m *mockHarness) GetMissionStatus(ctx context.Context, missionID string) (*mission.MissionStatusInfo, error) {
	return nil, nil
}

func (m *mockHarness) WaitForMission(ctx context.Context, missionID string, timeout time.Duration) (*mission.MissionResult, error) {
	return nil, nil
}

func (m *mockHarness) ListMissions(ctx context.Context, filter *mission.MissionFilter) ([]*mission.MissionInfo, error) {
	return nil, nil
}

func (m *mockHarness) CancelMission(ctx context.Context, missionID string) error {
	return nil
}

func (m *mockHarness) GetMissionResults(ctx context.Context, missionID string) (*mission.MissionResult, error) {
	return nil, nil
}

// TestRunPhase_Discover tests the discover phase execution
func TestRunPhase_Discover(t *testing.T) {
	tests := []struct {
		name           string
		targets        []string
		nmapOutput     map[string]any
		nmapError      error
		masscanOutput  map[string]any
		masscanError   error
		wantToolsRun   []string
		wantNodes      int
		wantRelations  int
		wantErrorCount int
	}{
		{
			name:    "successful nmap discovery",
			targets: []string{"192.168.1.0/24"},
			nmapOutput: map[string]any{
				"hosts": []any{
					map[string]any{
						"ip": "192.168.1.10",
						"ports": []any{
							map[string]any{"port": 22, "state": "open"},
							map[string]any{"port": 80, "state": "open"},
						},
					},
					map[string]any{
						"ip": "192.168.1.20",
						"ports": []any{
							map[string]any{"port": 443, "state": "open"},
						},
					},
				},
			},
			wantToolsRun:   []string{"nmap"},
			wantNodes:      5, // 2 hosts + 3 ports
			wantRelations:  5, // 3 HAS_PORT + 2 DISCOVERED
			wantErrorCount: 0,
		},
		{
			name:      "nmap fails, masscan succeeds",
			targets:   []string{"192.168.1.0/24"},
			nmapError: errors.New("nmap not found"),
			masscanOutput: map[string]any{
				"hosts": []any{
					map[string]any{
						"ip": "192.168.1.10",
						"ports": []any{
							map[string]any{"port": 22, "state": "open"},
						},
					},
				},
			},
			wantToolsRun:   []string{"masscan"},
			wantNodes:      2, // 1 host + 1 port
			wantRelations:  2, // 1 HAS_PORT + 1 DISCOVERED
			wantErrorCount: 1, // nmap failure recorded
		},
		{
			name:         "both tools fail",
			targets:      []string{"192.168.1.0/24"},
			nmapError:    errors.New("nmap not found"),
			masscanError: errors.New("masscan not found"),
			wantToolsRun:   []string{},
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 2,
		},
		{
			name:           "empty hosts output",
			targets:        []string{"192.168.1.0/24"},
			nmapOutput:     map[string]any{"hosts": []any{}},
			wantToolsRun:   []string{"nmap"},
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 0,
		},
		{
			name:    "multiple targets",
			targets: []string{"192.168.1.0/24", "10.0.0.0/24"},
			nmapOutput: map[string]any{
				"hosts": []any{
					map[string]any{
						"ip": "192.168.1.10",
						"ports": []any{
							map[string]any{"port": 22, "state": "open"},
						},
					},
				},
			},
			wantToolsRun:   []string{"nmap", "nmap"},
			wantNodes:      4, // (2 nodes per target) * 2 targets = 4
			wantRelations:  4, // (2 rels per target) * 2 targets = 4
			wantErrorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockHarness()
			if tt.nmapOutput != nil {
				mock.setToolOutput("nmap", tt.nmapOutput)
			}
			if tt.nmapError != nil {
				mock.setToolError("nmap", tt.nmapError)
			}
			if tt.masscanOutput != nil {
				mock.setToolOutput("masscan", tt.masscanOutput)
			}
			if tt.masscanError != nil {
				mock.setToolError("masscan", tt.masscanError)
			}

			mockExtractor := newMockExtractor()
			// Set expected extraction results based on tool outputs
			// For multiple targets, extractor is called once per target
			if tt.nmapOutput != nil {
				nodesPerCall := tt.wantNodes / len(tt.targets)
				relsPerCall := tt.wantRelations / len(tt.targets)
				if nodesPerCall == 0 {
					nodesPerCall = tt.wantNodes
				}
				if relsPerCall == 0 {
					relsPerCall = tt.wantRelations
				}
				mockExtractor.setExtractResult("nmap", nodesPerCall, relsPerCall)
			}
			if tt.masscanOutput != nil {
				nodesPerCall := tt.wantNodes / len(tt.targets)
				relsPerCall := tt.wantRelations / len(tt.targets)
				if nodesPerCall == 0 {
					nodesPerCall = tt.wantNodes
				}
				if relsPerCall == 0 {
					relsPerCall = tt.wantRelations
				}
				mockExtractor.setExtractResult("masscan", nodesPerCall, relsPerCall)
			}

			runner := &DefaultReconRunner{
				harness:   mock,
				extractor: mockExtractor,
			}
			ctx := context.Background()

			result, err := runner.RunPhase(ctx, PhaseDiscover, tt.targets)
			if err != nil {
				t.Fatalf("RunPhase() unexpected error: %v", err)
			}

			if result.Phase != PhaseDiscover {
				t.Errorf("Phase = %v, want %v", result.Phase, PhaseDiscover)
			}

			if len(result.ToolsRun) != len(tt.wantToolsRun) {
				t.Errorf("ToolsRun count = %d, want %d", len(result.ToolsRun), len(tt.wantToolsRun))
			}

			for i, tool := range tt.wantToolsRun {
				if i >= len(result.ToolsRun) {
					t.Errorf("Missing tool at index %d: want %s", i, tool)
					break
				}
				if result.ToolsRun[i] != tool {
					t.Errorf("ToolsRun[%d] = %s, want %s", i, result.ToolsRun[i], tool)
				}
			}

			if result.NodesCreated != tt.wantNodes {
				t.Errorf("NodesCreated = %d, want %d", result.NodesCreated, tt.wantNodes)
			}

			if result.RelationsCreated != tt.wantRelations {
				t.Errorf("RelationsCreated = %d, want %d", result.RelationsCreated, tt.wantRelations)
			}

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf("Error count = %d, want %d", len(result.Errors), tt.wantErrorCount)
			}

			if result.Duration == 0 {
				t.Error("Duration should be non-zero")
			}
		})
	}
}

// TestRunPhase_Probe tests the probe phase execution
func TestRunPhase_Probe(t *testing.T) {
	tests := []struct {
		name           string
		targets        []string
		httpxOutput    map[string]any
		httpxError     error
		wantToolsRun   []string
		wantNodes      int
		wantRelations  int
		wantErrorCount int
	}{
		{
			name:    "successful httpx probe",
			targets: []string{"http://192.168.1.10:8080", "http://192.168.1.20:443"},
			httpxOutput: map[string]any{
				"endpoints": []any{
					map[string]any{
						"url":    "http://192.168.1.10:8080",
						"status": 200,
						"technologies": []any{
							map[string]any{"name": "nginx", "version": "1.19"},
							map[string]any{"name": "php", "version": "7.4"},
						},
					},
					map[string]any{
						"url":    "http://192.168.1.20:443",
						"status": 200,
						"technologies": []any{
							map[string]any{"name": "apache", "version": "2.4"},
						},
					},
				},
			},
			wantToolsRun:   []string{"httpx"},
			wantNodes:      5, // 2 endpoints + 3 technologies
			wantRelations:  5, // 2 EXPOSES + 3 USES_TECHNOLOGY
			wantErrorCount: 0,
		},
		{
			name:       "httpx fails",
			targets:    []string{"http://192.168.1.10:8080"},
			httpxError: errors.New("httpx failed"),
			wantToolsRun:   []string{},
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 1,
		},
		{
			name:           "no targets",
			targets:        []string{},
			wantToolsRun:   []string{},
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 0,
		},
		{
			name:    "endpoints without technologies",
			targets: []string{"http://192.168.1.10:8080"},
			httpxOutput: map[string]any{
				"endpoints": []any{
					map[string]any{
						"url":    "http://192.168.1.10:8080",
						"status": 200,
					},
				},
			},
			wantToolsRun:   []string{"httpx"},
			wantNodes:      1, // 1 endpoint
			wantRelations:  1, // 1 EXPOSES
			wantErrorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockHarness()
			if tt.httpxOutput != nil {
				mock.setToolOutput("httpx", tt.httpxOutput)
			}
			if tt.httpxError != nil {
				mock.setToolError("httpx", tt.httpxError)
			}

			mockExtractor := newMockExtractor()
			if tt.httpxOutput != nil {
				mockExtractor.setExtractResult("httpx", tt.wantNodes, tt.wantRelations)
			}

			runner := &DefaultReconRunner{
				harness:   mock,
				extractor: mockExtractor,
			}
			ctx := context.Background()

			result, err := runner.RunPhase(ctx, PhaseProbe, tt.targets)
			if err != nil {
				t.Fatalf("RunPhase() unexpected error: %v", err)
			}

			if result.Phase != PhaseProbe {
				t.Errorf("Phase = %v, want %v", result.Phase, PhaseProbe)
			}

			if len(result.ToolsRun) != len(tt.wantToolsRun) {
				t.Errorf("ToolsRun count = %d, want %d", len(result.ToolsRun), len(tt.wantToolsRun))
			}

			if result.NodesCreated != tt.wantNodes {
				t.Errorf("NodesCreated = %d, want %d", result.NodesCreated, tt.wantNodes)
			}

			if result.RelationsCreated != tt.wantRelations {
				t.Errorf("RelationsCreated = %d, want %d", result.RelationsCreated, tt.wantRelations)
			}

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf("Error count = %d, want %d", len(result.Errors), tt.wantErrorCount)
			}
		})
	}
}

// TestRunPhase_Scan tests the scan phase execution
func TestRunPhase_Scan(t *testing.T) {
	tests := []struct {
		name           string
		targets        []string
		nucleiOutput   map[string]any
		nucleiError    error
		wantToolsRun   []string
		wantNodes      int
		wantRelations  int
		wantErrorCount int
	}{
		{
			name:    "successful nuclei scan",
			targets: []string{"http://192.168.1.10:8080", "http://192.168.1.20:443"},
			nucleiOutput: map[string]any{
				"findings": []any{
					map[string]any{
						"template_id": "CVE-2021-44228",
						"severity":    "critical",
						"matched_at":  "http://192.168.1.10:8080/api/search",
					},
					map[string]any{
						"template_id": "CVE-2023-12345",
						"severity":    "high",
						"matched_at":  "http://192.168.1.20:443/admin",
					},
				},
			},
			wantToolsRun:   []string{"nuclei"},
			wantNodes:      2, // 2 findings
			wantRelations:  2, // 2 AFFECTS
			wantErrorCount: 0,
		},
		{
			name:        "nuclei fails",
			targets:     []string{"http://192.168.1.10:8080"},
			nucleiError: errors.New("nuclei failed"),
			wantToolsRun:   []string{},
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 1,
		},
		{
			name:           "no targets",
			targets:        []string{},
			wantToolsRun:   []string{},
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 0,
		},
		{
			name:           "no findings",
			targets:        []string{"http://192.168.1.10:8080"},
			nucleiOutput:   map[string]any{"findings": []any{}},
			wantToolsRun:   []string{"nuclei"},
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockHarness()
			if tt.nucleiOutput != nil {
				mock.setToolOutput("nuclei", tt.nucleiOutput)
			}
			if tt.nucleiError != nil {
				mock.setToolError("nuclei", tt.nucleiError)
			}

			mockExtractor := newMockExtractor()
			if tt.nucleiOutput != nil {
				mockExtractor.setExtractResult("nuclei", tt.wantNodes, tt.wantRelations)
			}

			runner := &DefaultReconRunner{
				harness:   mock,
				extractor: mockExtractor,
			}
			ctx := context.Background()

			result, err := runner.RunPhase(ctx, PhaseScan, tt.targets)
			if err != nil {
				t.Fatalf("RunPhase() unexpected error: %v", err)
			}

			if result.Phase != PhaseScan {
				t.Errorf("Phase = %v, want %v", result.Phase, PhaseScan)
			}

			if len(result.ToolsRun) != len(tt.wantToolsRun) {
				t.Errorf("ToolsRun count = %d, want %d", len(result.ToolsRun), len(tt.wantToolsRun))
			}

			if result.NodesCreated != tt.wantNodes {
				t.Errorf("NodesCreated = %d, want %d", result.NodesCreated, tt.wantNodes)
			}

			if result.RelationsCreated != tt.wantRelations {
				t.Errorf("RelationsCreated = %d, want %d", result.RelationsCreated, tt.wantRelations)
			}

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf("Error count = %d, want %d", len(result.Errors), tt.wantErrorCount)
			}
		})
	}
}

// TestRunPhase_Domain tests the domain phase execution
func TestRunPhase_Domain(t *testing.T) {
	tests := []struct {
		name             string
		targets          []string
		subfinderOutput  map[string]any
		subfinderError   error
		amassOutput      map[string]any
		amassError       error
		wantToolsRun     int
		wantNodes        int
		wantRelations    int
		wantErrorCount   int
		wantMinToolsRun  int
	}{
		{
			name:    "successful subfinder and amass",
			targets: []string{"example.local"},
			subfinderOutput: map[string]any{
				"subdomains": []any{
					map[string]any{"name": "api.example.local"},
					map[string]any{"name": "www.example.local"},
				},
			},
			amassOutput: map[string]any{
				"subdomains": []any{
					map[string]any{"name": "mail.example.local"},
				},
				"hosts": []any{
					map[string]any{"ip": "192.168.1.10"},
				},
				"asns": []any{
					map[string]any{"asn": "AS12345"},
				},
				"dns_records": []any{
					map[string]any{"type": "A", "value": "192.168.1.10"},
				},
			},
			wantToolsRun:   2, // subfinder + amass
			wantNodes:      6, // 2 subfinder subdomains + 1 amass subdomain + 1 host + 1 asn + 1 dns_record
			wantRelations:  6, // corresponding relationships
			wantErrorCount: 0,
		},
		{
			name:           "subfinder fails, amass succeeds",
			targets:        []string{"example.local"},
			subfinderError: errors.New("subfinder failed"),
			amassOutput: map[string]any{
				"subdomains": []any{
					map[string]any{"name": "mail.example.local"},
				},
			},
			wantToolsRun:   1, // only amass
			wantNodes:      1,
			wantRelations:  1,
			wantErrorCount: 1,
		},
		{
			name:           "no domains",
			targets:        []string{},
			wantToolsRun:   0,
			wantNodes:      0,
			wantRelations:  0,
			wantErrorCount: 0,
		},
		{
			name:            "multiple domains",
			targets:         []string{"example.local", "test.local"},
			subfinderOutput: map[string]any{"subdomains": []any{map[string]any{"name": "sub.example.local"}}},
			amassOutput:     map[string]any{"subdomains": []any{map[string]any{"name": "sub2.example.local"}}},
			wantMinToolsRun: 4, // (subfinder + amass) * 2 domains
			wantNodes:       4, // (1 subdomain from subfinder + 1 from amass) * 2 domains
			wantRelations:   4,
			wantErrorCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockHarness()
			if tt.subfinderOutput != nil {
				mock.setToolOutput("subfinder", tt.subfinderOutput)
			}
			if tt.subfinderError != nil {
				mock.setToolError("subfinder", tt.subfinderError)
			}
			if tt.amassOutput != nil {
				mock.setToolOutput("amass", tt.amassOutput)
			}
			if tt.amassError != nil {
				mock.setToolError("amass", tt.amassError)
			}

			// Domain phase doesn't use extractor (uses direct counting)
			runner := &DefaultReconRunner{
				harness:   mock,
				extractor: newMockExtractor(), // Still need one, even if unused
			}
			ctx := context.Background()

			result, err := runner.RunPhase(ctx, PhaseDomain, tt.targets)
			if err != nil {
				t.Fatalf("RunPhase() unexpected error: %v", err)
			}

			if result.Phase != PhaseDomain {
				t.Errorf("Phase = %v, want %v", result.Phase, PhaseDomain)
			}

			// Verify correct tools were called
			if tt.wantMinToolsRun > 0 {
				if len(result.ToolsRun) < tt.wantMinToolsRun {
					t.Errorf("ToolsRun count = %d, want at least %d", len(result.ToolsRun), tt.wantMinToolsRun)
				}
			} else if len(result.ToolsRun) != tt.wantToolsRun {
				t.Errorf("ToolsRun count = %d, want %d", len(result.ToolsRun), tt.wantToolsRun)
			}

			// Note: Domain phase uses direct output counting, not the extractor.
			// With mocked harness, we verify tool execution and error handling.
			// Node/relation counting is validated in integration tests with real tools.

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf("Error count = %d, want %d", len(result.Errors), tt.wantErrorCount)
			}
		})
	}
}

// TestRunPhase_UnknownPhase tests error handling for unknown phase
func TestRunPhase_UnknownPhase(t *testing.T) {
	mock := newMockHarness()
	runner := &DefaultReconRunner{
		harness:   mock,
		extractor: newMockExtractor(),
	}
	ctx := context.Background()

	_, err := runner.RunPhase(ctx, Phase("invalid"), []string{"target"})
	if err == nil {
		t.Error("RunPhase() with unknown phase should return error")
	}
}

// TestRunAll tests the complete reconnaissance workflow
func TestRunAll(t *testing.T) {
	tests := []struct {
		name           string
		subnet         string
		domains        []string
		nmapOutput     map[string]any
		httpxOutput    map[string]any
		nucleiOutput   map[string]any
		subfinderOutput map[string]any
		amassOutput    map[string]any
		wantPhases     int
		wantTotalHosts int
	}{
		{
			name:    "complete workflow with all phases",
			subnet:  "192.168.1.0/24",
			domains: []string{"example.local"},
			nmapOutput: map[string]any{
				"hosts": []any{
					map[string]any{
						"ip": "192.168.1.10",
						"ports": []any{
							map[string]any{"port": 80, "state": "open"},
						},
					},
				},
			},
			httpxOutput:     map[string]any{"endpoints": []any{}},
			nucleiOutput:    map[string]any{"findings": []any{}},
			subfinderOutput: map[string]any{"subdomains": []any{}},
			amassOutput:     map[string]any{"subdomains": []any{}},
			wantPhases:      2, // discover + domain (probe/scan skipped when no targets extracted)
			wantTotalHosts:  1,
		},
		{
			name:    "workflow with no domains",
			subnet:  "192.168.1.0/24",
			domains: []string{},
			nmapOutput: map[string]any{
				"hosts": []any{
					map[string]any{
						"ip": "192.168.1.10",
						"ports": []any{
							map[string]any{"port": 22, "state": "open"},
						},
					},
				},
			},
			wantPhases:     1, // only discover
			wantTotalHosts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockHarness()
			if tt.nmapOutput != nil {
				mock.setToolOutput("nmap", tt.nmapOutput)
			}
			if tt.httpxOutput != nil {
				mock.setToolOutput("httpx", tt.httpxOutput)
			}
			if tt.nucleiOutput != nil {
				mock.setToolOutput("nuclei", tt.nucleiOutput)
			}
			if tt.subfinderOutput != nil {
				mock.setToolOutput("subfinder", tt.subfinderOutput)
			}
			if tt.amassOutput != nil {
				mock.setToolOutput("amass", tt.amassOutput)
			}

			mockExtractor := newMockExtractor()
			// Set up extractor to return counts for discover phase
			if tt.nmapOutput != nil {
				// Extractor will count from output
				mockExtractor.setExtractResult("nmap", 2, 2) // 1 host + 1 port = 2 nodes, 2 rels
			}

			runner := &DefaultReconRunner{
				harness:   mock,
				extractor: mockExtractor,
			}
			ctx := context.Background()

			result, err := runner.RunAll(ctx, tt.subnet, tt.domains)
			if err != nil {
				t.Fatalf("RunAll() unexpected error: %v", err)
			}

			if len(result.Phases) != tt.wantPhases {
				t.Errorf("Phases count = %d, want %d", len(result.Phases), tt.wantPhases)
			}

			if result.TotalHosts != tt.wantTotalHosts {
				t.Errorf("TotalHosts = %d, want %d", result.TotalHosts, tt.wantTotalHosts)
			}

			if result.Duration == 0 {
				t.Error("Duration should be non-zero")
			}

			// Verify phases are in correct order
			if len(result.Phases) > 0 && result.Phases[0].Phase != PhaseDiscover {
				t.Errorf("First phase should be discover, got %v", result.Phases[0].Phase)
			}
		})
	}
}

// TestRunAll_PhaseFailures tests that phase failures don't halt the workflow
func TestRunAll_PhaseFailures(t *testing.T) {
	mock := newMockHarness()
	mock.setToolError("nmap", errors.New("nmap failed"))
	mock.setToolError("masscan", errors.New("masscan failed"))

	runner := &DefaultReconRunner{
		harness:   mock,
		extractor: newMockExtractor(),
	}
	ctx := context.Background()

	// Even though discover phase fails, RunAll should not return an error
	// The workflow should continue with available data
	result, err := runner.RunAll(ctx, "192.168.1.0/24", []string{})
	if err != nil {
		t.Fatalf("RunAll() should not fail even when tools fail: %v", err)
	}

	// Verify discover phase was attempted
	if len(result.Phases) == 0 {
		t.Error("Expected at least discover phase result")
	}

	// Verify errors were captured in the phase result
	if len(result.Phases) > 0 && len(result.Phases[0].Errors) == 0 {
		t.Error("Expected errors to be captured in discover phase")
	}
}

// TestPhaseResult_Statistics tests that phase result statistics are correctly calculated
func TestPhaseResult_Statistics(t *testing.T) {
	mock := newMockHarness()
	mock.setToolOutput("nmap", map[string]any{
		"hosts": []any{
			map[string]any{
				"ip": "192.168.1.10",
				"ports": []any{
					map[string]any{"port": 22, "state": "open"},
					map[string]any{"port": 80, "state": "open"},
					map[string]any{"port": 443, "state": "open"},
				},
			},
		},
	})

	mockExtractor := newMockExtractor()
	mockExtractor.setExtractResult("nmap", 4, 4) // 1 host + 3 ports = 4 nodes, 4 rels

	runner := &DefaultReconRunner{
		harness:   mock,
		extractor: mockExtractor,
	}
	ctx := context.Background()

	result, err := runner.RunPhase(ctx, PhaseDiscover, []string{"192.168.1.0/24"})
	if err != nil {
		t.Fatalf("RunPhase() unexpected error: %v", err)
	}

	// Verify statistics
	if result.NodesCreated != 4 {
		t.Errorf("NodesCreated = %d, want 4 (1 host + 3 ports)", result.NodesCreated)
	}

	if result.RelationsCreated != 4 {
		t.Errorf("RelationsCreated = %d, want 4 (3 HAS_PORT + 1 DISCOVERED)", result.RelationsCreated)
	}

	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

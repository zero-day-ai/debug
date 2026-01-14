package main

import (
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"
)

// OutputFormat defines how the report should be formatted
type OutputFormat string

const (
	// OutputJSON produces JSON format output
	OutputJSON OutputFormat = "json"

	// OutputText produces human-readable text output
	OutputText OutputFormat = "text"

	// OutputBoth produces both JSON and text output
	OutputBoth OutputFormat = "both"
)

// DebugConfig holds configuration for debug agent execution
type DebugConfig struct {
	// Verbose enables detailed output during execution
	Verbose bool

	// TargetTests specifies which test modules to run in single mode
	// Example: ["llm-integration", "daemon-service"]
	TargetTests []string

	// Timeout is the overall execution timeout
	Timeout time.Duration

	// CategoryTimeout is the timeout per test category
	CategoryTimeout time.Duration

	// TestTimeout is the timeout per individual test
	TestTimeout time.Duration

	// SkipCategories lists test categories to skip
	SkipCategories []string

	// SkipTests lists individual tests to skip by name
	SkipTests []string

	// OutputFormat determines report output format (json, text, both)
	OutputFormat OutputFormat

	// SubmitFindings determines whether to submit failed tests as findings
	SubmitFindings bool

	// Debug Mission Context Fields

	// Component specifies which component to test in health-check mode
	// Valid values: "graphrag", "tools", "memory", "plugins"
	Component string

	// Method specifies which LLM method to test in llm-test mode
	// Valid values: "complete", "structured", "with_tools"
	Method string

	// Prefix is the prefix to use for test data (e.g., "[DEBUG]")
	Prefix string

	// Network Reconnaissance Configuration

	// Subnet is the CIDR subnet to scan (optional, auto-discovered if empty)
	Subnet string

	// Domains is the list of domains to enumerate (optional, from /etc/hosts if empty)
	Domains []string

	// SkipPhases lists reconnaissance phases to skip
	// Valid values: "discover", "probe", "scan", "domain", "analyze"
	SkipPhases []string

	// GenerateIntelligence determines whether to run LLM analysis
	GenerateIntelligence bool
}

// DefaultConfig returns a DebugConfig with sensible defaults
func DefaultConfig() *DebugConfig {
	return &DebugConfig{
		Verbose:              false,
		TargetTests:          []string{},
		Timeout:              10 * time.Minute,  // 10 minutes for full suite
		CategoryTimeout:      60 * time.Second,  // 60 seconds per category
		TestTimeout:          10 * time.Second,  // 10 seconds per test
		SkipCategories:       []string{},
		SkipTests:            []string{},
		OutputFormat:         OutputBoth,
		SubmitFindings:       true,
		Subnet:               "",       // Auto-discover if empty
		Domains:              []string{}, // Auto-discover from /etc/hosts if empty
		SkipPhases:           []string{},
		GenerateIntelligence: true, // Generate LLM analysis by default
	}
}

// ParseConfig extracts configuration from the agent task context
// The task context can contain configuration in either Context or Metadata field
// Context is used by workflow YAML, Metadata is used by direct API calls
func ParseConfig(task agent.Task) (*DebugConfig, error) {
	cfg := DefaultConfig()

	// DEBUG: Log what we receive
	fmt.Printf("[DEBUG] ParseConfig: task.ID=%s\n", task.ID)
	fmt.Printf("[DEBUG] ParseConfig: task.Context keys=%v\n", getConfigMapKeys(task.Context))
	fmt.Printf("[DEBUG] ParseConfig: task.Metadata keys=%v\n", getConfigMapKeys(task.Metadata))
	if subnet, ok := task.Context["subnet"]; ok {
		fmt.Printf("[DEBUG] ParseConfig: task.Context[subnet]=%v\n", subnet)
	}

	// Merge Context and Metadata - Context takes precedence (from workflow YAML)
	configMap := make(map[string]any)
	for k, v := range task.Metadata {
		configMap[k] = v
	}
	for k, v := range task.Context {
		configMap[k] = v
	}

	// Parse configuration from merged map
	if len(configMap) == 0 {
		// Validate and return if no config
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// Parse verbose flag
	if verbose, ok := configMap["verbose"].(bool); ok {
		cfg.Verbose = verbose
	}

	// Parse target tests for single mode
	if tests, ok := configMap["tests"].([]interface{}); ok {
		cfg.TargetTests = make([]string, 0, len(tests))
		for _, test := range tests {
			if testName, ok := test.(string); ok {
				cfg.TargetTests = append(cfg.TargetTests, testName)
			}
		}
	}

	// Parse timeout
	if timeout, ok := configMap["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = d
		} else {
			return nil, fmt.Errorf("invalid timeout duration: %s", timeout)
		}
	}

	// Parse category timeout
	if categoryTimeout, ok := configMap["category_timeout"].(string); ok {
		if d, err := time.ParseDuration(categoryTimeout); err == nil {
			cfg.CategoryTimeout = d
		} else {
			return nil, fmt.Errorf("invalid category_timeout duration: %s", categoryTimeout)
		}
	}

	// Parse test timeout
	if testTimeout, ok := configMap["test_timeout"].(string); ok {
		if d, err := time.ParseDuration(testTimeout); err == nil {
			cfg.TestTimeout = d
		} else {
			return nil, fmt.Errorf("invalid test_timeout duration: %s", testTimeout)
		}
	}

	// Parse skip categories
	if skipCats, ok := configMap["skip_categories"].([]interface{}); ok {
		cfg.SkipCategories = make([]string, 0, len(skipCats))
		for _, cat := range skipCats {
			if catName, ok := cat.(string); ok {
				cfg.SkipCategories = append(cfg.SkipCategories, catName)
			}
		}
	}

	// Parse skip tests
	if skipTests, ok := configMap["skip_tests"].([]interface{}); ok {
		cfg.SkipTests = make([]string, 0, len(skipTests))
		for _, test := range skipTests {
			if testName, ok := test.(string); ok {
				cfg.SkipTests = append(cfg.SkipTests, testName)
			}
		}
	}

	// Parse output format
	if format, ok := configMap["output_format"].(string); ok {
		switch OutputFormat(format) {
		case OutputJSON, OutputText, OutputBoth:
			cfg.OutputFormat = OutputFormat(format)
		default:
			return nil, fmt.Errorf("invalid output format: %s (must be json, text, or both)", format)
		}
	}

	// Parse submit findings flag
	if submitFindings, ok := configMap["submit_findings"].(bool); ok {
		cfg.SubmitFindings = submitFindings
	}

	// Parse debug mission context fields
	if component, ok := configMap["component"].(string); ok {
		cfg.Component = component
	}
	if method, ok := configMap["method"].(string); ok {
		cfg.Method = method
	}
	if prefix, ok := configMap["prefix"].(string); ok {
		cfg.Prefix = prefix
	}

	// Parse network reconnaissance config fields
	if subnet, ok := configMap["subnet"].(string); ok {
		cfg.Subnet = subnet
	}
	if domains, ok := configMap["domains"].([]interface{}); ok {
		cfg.Domains = make([]string, 0, len(domains))
		for _, domain := range domains {
			if domainName, ok := domain.(string); ok {
				cfg.Domains = append(cfg.Domains, domainName)
			}
		}
	}
	if skipPhases, ok := configMap["skip_phases"].([]interface{}); ok {
		cfg.SkipPhases = make([]string, 0, len(skipPhases))
		for _, phase := range skipPhases {
			if phaseName, ok := phase.(string); ok {
				cfg.SkipPhases = append(cfg.SkipPhases, phaseName)
			}
		}
	}
	if generateIntel, ok := configMap["generate_intelligence"].(bool); ok {
		cfg.GenerateIntelligence = generateIntel
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *DebugConfig) Validate() error {
	// Validate timeouts
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}
	if c.CategoryTimeout <= 0 {
		return fmt.Errorf("category_timeout must be positive, got %v", c.CategoryTimeout)
	}
	if c.TestTimeout <= 0 {
		return fmt.Errorf("test_timeout must be positive, got %v", c.TestTimeout)
	}

	// Validate output format
	switch c.OutputFormat {
	case OutputJSON, OutputText, OutputBoth:
		// valid
	default:
		return fmt.Errorf("invalid output format: %s", c.OutputFormat)
	}

	// Validate skip_phases contains only valid phase names
	validPhases := map[string]bool{
		"discover": true,
		"probe":    true,
		"scan":     true,
		"domain":   true,
		"analyze":  true,
	}
	for _, phase := range c.SkipPhases {
		if !validPhases[phase] {
			return fmt.Errorf("invalid phase name in skip_phases: %s (must be: discover, probe, scan, domain, or analyze)", phase)
		}
	}

	return nil
}

// String returns a human-readable representation of the configuration
func (c *DebugConfig) String() string {
	return fmt.Sprintf("DebugConfig{verbose=%v, timeout=%v, output=%s}",
		c.Verbose, c.Timeout, c.OutputFormat)
}

// ShouldRunPhase returns true if the given reconnaissance phase should be run
func (c *DebugConfig) ShouldRunPhase(phase string) bool {
	// Check if phase is explicitly skipped
	for _, skip := range c.SkipPhases {
		if skip == phase {
			return false
		}
	}
	// All phases are enabled by default unless explicitly skipped
	return true
}

// ShouldRunTest returns true if the given test should be run
func (c *DebugConfig) ShouldRunTest(testName string) bool {
	// Check if test is explicitly skipped
	for _, skip := range c.SkipTests {
		if skip == testName {
			return false
		}
	}

	// If target tests are specified, only run those
	if len(c.TargetTests) > 0 {
		for _, target := range c.TargetTests {
			if target == testName {
				return true
			}
		}
		return false
	}

	return true
}

// ShouldRunCategory returns true if tests in the given category should be run
func (c *DebugConfig) ShouldRunCategory(category string) bool {
	// Check if category is explicitly skipped
	for _, skip := range c.SkipCategories {
		if skip == category {
			return false
		}
	}

	return true
}

// getConfigMapKeys returns the keys of a map[string]any for debugging
func getConfigMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

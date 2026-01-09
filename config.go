package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/zero-day-ai/sdk/agent"
)

// ExecutionMode defines how the debug agent should run its test suite
type ExecutionMode string

const (
	// ModeFullSuite runs all SDK and Framework tests (default)
	ModeFullSuite ExecutionMode = "full"

	// ModeSDKOnly runs only SDK tests (Requirements 1-16)
	ModeSDKOnly ExecutionMode = "sdk"

	// ModeFrameworkOnly runs only Framework tests (Requirements 17-31)
	ModeFrameworkOnly ExecutionMode = "framework"

	// ModeNetworkRecon runs network reconnaissance module
	ModeNetworkRecon ExecutionMode = "network-recon"

	// ModeSingleTest runs specific test modules by name
	ModeSingleTest ExecutionMode = "single"
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
	// Mode determines which tests to run (full, sdk, framework, single)
	Mode ExecutionMode

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
}

// DefaultConfig returns a DebugConfig with sensible defaults
func DefaultConfig() *DebugConfig {
	return &DebugConfig{
		Mode:            ModeFullSuite,
		Verbose:         false,
		TargetTests:     []string{},
		Timeout:         10 * time.Minute,  // 10 minutes for full suite
		CategoryTimeout: 60 * time.Second,  // 60 seconds per category
		TestTimeout:     10 * time.Second,  // 10 seconds per test
		SkipCategories:  []string{},
		SkipTests:       []string{},
		OutputFormat:    OutputBoth,
		SubmitFindings:  true,
	}
}

// ParseConfig extracts configuration from the agent task context
// The task context can contain configuration in the Metadata field
func ParseConfig(task agent.Task) (*DebugConfig, error) {
	cfg := DefaultConfig()

	// Auto-detect mode from goal if it contains "network-recon"
	// This allows: gibson attack --target test-network --agent debug-agent --goal network-recon
	if strings.Contains(strings.ToLower(task.Goal), "network-recon") {
		cfg.Mode = ModeNetworkRecon
		// Network recon needs more time - scanning 256 IPs with 20 concurrent pings
		// at 1 second timeout each = ~13 seconds minimum, plus nmap scans
		cfg.TestTimeout = 5 * time.Minute
	}

	// Parse configuration from task metadata (overrides goal-based detection)
	if task.Metadata == nil {
		// Validate and return if no metadata
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// Parse mode (overrides goal-based detection)
	if mode, ok := task.Metadata["mode"].(string); ok {
		switch ExecutionMode(mode) {
		case ModeFullSuite, ModeSDKOnly, ModeFrameworkOnly, ModeNetworkRecon, ModeSingleTest:
			cfg.Mode = ExecutionMode(mode)
		default:
			return nil, fmt.Errorf("invalid execution mode: %s (must be full, sdk, framework, network-recon, or single)", mode)
		}
	}

	// Parse verbose flag
	if verbose, ok := task.Metadata["verbose"].(bool); ok {
		cfg.Verbose = verbose
	}

	// Parse target tests for single mode
	if tests, ok := task.Metadata["tests"].([]interface{}); ok {
		cfg.TargetTests = make([]string, 0, len(tests))
		for _, test := range tests {
			if testName, ok := test.(string); ok {
				cfg.TargetTests = append(cfg.TargetTests, testName)
			}
		}
	}

	// Parse timeout
	if timeout, ok := task.Metadata["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = d
		} else {
			return nil, fmt.Errorf("invalid timeout duration: %s", timeout)
		}
	}

	// Parse category timeout
	if categoryTimeout, ok := task.Metadata["category_timeout"].(string); ok {
		if d, err := time.ParseDuration(categoryTimeout); err == nil {
			cfg.CategoryTimeout = d
		} else {
			return nil, fmt.Errorf("invalid category_timeout duration: %s", categoryTimeout)
		}
	}

	// Parse test timeout
	if testTimeout, ok := task.Metadata["test_timeout"].(string); ok {
		if d, err := time.ParseDuration(testTimeout); err == nil {
			cfg.TestTimeout = d
		} else {
			return nil, fmt.Errorf("invalid test_timeout duration: %s", testTimeout)
		}
	}

	// Parse skip categories
	if skipCats, ok := task.Metadata["skip_categories"].([]interface{}); ok {
		cfg.SkipCategories = make([]string, 0, len(skipCats))
		for _, cat := range skipCats {
			if catName, ok := cat.(string); ok {
				cfg.SkipCategories = append(cfg.SkipCategories, catName)
			}
		}
	}

	// Parse skip tests
	if skipTests, ok := task.Metadata["skip_tests"].([]interface{}); ok {
		cfg.SkipTests = make([]string, 0, len(skipTests))
		for _, test := range skipTests {
			if testName, ok := test.(string); ok {
				cfg.SkipTests = append(cfg.SkipTests, testName)
			}
		}
	}

	// Parse output format
	if format, ok := task.Metadata["output_format"].(string); ok {
		switch OutputFormat(format) {
		case OutputJSON, OutputText, OutputBoth:
			cfg.OutputFormat = OutputFormat(format)
		default:
			return nil, fmt.Errorf("invalid output format: %s (must be json, text, or both)", format)
		}
	}

	// Parse submit findings flag
	if submitFindings, ok := task.Metadata["submit_findings"].(bool); ok {
		cfg.SubmitFindings = submitFindings
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *DebugConfig) Validate() error {
	// Validate mode
	switch c.Mode {
	case ModeFullSuite, ModeSDKOnly, ModeFrameworkOnly, ModeNetworkRecon, ModeSingleTest:
		// valid
	default:
		return fmt.Errorf("invalid execution mode: %s", c.Mode)
	}

	// Validate single mode has target tests
	if c.Mode == ModeSingleTest && len(c.TargetTests) == 0 {
		return fmt.Errorf("single mode requires target_tests to be specified")
	}

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

	return nil
}

// String returns a human-readable representation of the configuration
func (c *DebugConfig) String() string {
	return fmt.Sprintf("DebugConfig{mode=%s, verbose=%v, timeout=%v, output=%s}",
		c.Mode, c.Verbose, c.Timeout, c.OutputFormat)
}

// IsSDKEnabled returns true if SDK tests should be run
func (c *DebugConfig) IsSDKEnabled() bool {
	return c.Mode == ModeFullSuite || c.Mode == ModeSDKOnly
}

// IsFrameworkEnabled returns true if Framework tests should be run
func (c *DebugConfig) IsFrameworkEnabled() bool {
	return c.Mode == ModeFullSuite || c.Mode == ModeFrameworkOnly
}

// IsNetworkReconEnabled returns true if Network Recon tests should be run
func (c *DebugConfig) IsNetworkReconEnabled() bool {
	return c.Mode == ModeNetworkRecon
}

// ShouldRunTest returns true if the given test should be run
func (c *DebugConfig) ShouldRunTest(testName string) bool {
	// Check if test is explicitly skipped
	for _, skip := range c.SkipTests {
		if skip == testName {
			return false
		}
	}

	// In single mode, only run target tests
	if c.Mode == ModeSingleTest {
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

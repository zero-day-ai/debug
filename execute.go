package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"

	"github.com/zero-day-ai/agents/debug/internal/framework"
	"github.com/zero-day-ai/agents/debug/internal/network"
	"github.com/zero-day-ai/agents/debug/internal/runner"
	"github.com/zero-day-ai/agents/debug/internal/sdk"
)

// executeDebugAgent is the main execution function for the debug agent.
// It orchestrates the test suite execution based on configuration from the task context.
func executeDebugAgent(ctx context.Context, h agent.Harness, task agent.Task) (agent.Result, error) {
	logger := h.Logger()
	startTime := time.Now()

	logger.Info("Debug agent execution started",
		"task_id", task.ID,
		"goal", task.Goal,
		"agent", agentName,
		"version", agentVersion,
	)

	// Parse configuration from task context
	cfg, err := ParseConfig(task)
	if err != nil {
		logger.Error("Failed to parse configuration",
			"error", err,
		)
		return agent.Result{
			Status: agent.StatusFailed,
			Output: fmt.Sprintf("Configuration error: %v", err),
		}, nil
	}

	logger.Info("Configuration parsed",
		"mode", cfg.Mode,
		"verbose", cfg.Verbose,
		"timeout", cfg.Timeout,
		"output_format", cfg.OutputFormat,
	)

	// Create the test runner
	testRunner := runner.NewRunner(h, string(cfg.Mode), cfg.Timeout, cfg.TestTimeout)

	// Register test modules based on mode
	if err := registerTestModules(testRunner, cfg); err != nil {
		logger.Error("Failed to register test modules",
			"error", err,
		)
		return agent.Result{
			Status: agent.StatusFailed,
			Output: fmt.Sprintf("Module registration error: %v", err),
		}, nil
	}

	logger.Info("Test modules registered",
		"total_modules", len(testRunner.GetModules()),
		"sdk_modules", len(testRunner.GetModulesByCategory(runner.CategorySDK)),
		"framework_modules", len(testRunner.GetModulesByCategory(runner.CategoryFramework)),
	)

	// Execute the test suite based on mode
	var suiteResult *runner.SuiteResult

	switch cfg.Mode {
	case ModeFullSuite:
		suiteResult, err = testRunner.Run(ctx)

	case ModeSDKOnly:
		suiteResult, err = testRunner.RunCategory(ctx, runner.CategorySDK)

	case ModeFrameworkOnly:
		suiteResult, err = testRunner.RunCategory(ctx, runner.CategoryFramework)

	case ModeNetworkRecon:
		suiteResult = runner.NewSuiteResult(string(cfg.Mode))
		results, testErr := testRunner.RunSingle(ctx, "network-recon")
		if testErr != nil {
			logger.Warn("Failed to run network-recon test",
				"error", testErr,
			)
			suiteResult.AddResult(runner.NewErrorResult(
				"network-recon",
				"NR-1..NR-8",
				runner.CategorySDK,
				0,
				testErr,
			))
		} else {
			suiteResult.AddResults(results)
		}
		suiteResult.Finalize()

	case ModeSingleTest:
		// For single mode, run each target test and aggregate results
		suiteResult = runner.NewSuiteResult(string(cfg.Mode))
		for _, testName := range cfg.TargetTests {
			results, testErr := testRunner.RunSingle(ctx, testName)
			if testErr != nil {
				logger.Warn("Failed to run single test",
					"test", testName,
					"error", testErr,
				)
				// Add error result for the failed test
				suiteResult.AddResult(runner.NewErrorResult(
					testName,
					"unknown",
					runner.CategorySDK,
					0,
					testErr,
				))
			} else {
				suiteResult.AddResults(results)
			}
		}
		suiteResult.Finalize()

	default:
		return agent.Result{
			Status: agent.StatusFailed,
			Output: fmt.Sprintf("Unknown execution mode: %s", cfg.Mode),
		}, nil
	}

	if err != nil {
		logger.Error("Test suite execution failed",
			"error", err,
			"duration", time.Since(startTime),
		)
		return agent.Result{
			Status: agent.StatusFailed,
			Output: fmt.Sprintf("Execution error: %v", err),
		}, nil
	}

	// Log execution summary
	logger.Info("Test suite execution completed",
		"duration", suiteResult.Duration(),
		"total_tests", suiteResult.TotalTests(),
		"passed", suiteResult.TotalPassed(),
		"failed", suiteResult.TotalFailed(),
		"skipped", suiteResult.TotalSkipped(),
		"errors", suiteResult.TotalErrors(),
		"overall_status", suiteResult.OverallStatus,
		"pass_rate", fmt.Sprintf("%.2f%%", suiteResult.OverallPassRate()*100),
	)

	// Generate output based on format
	output := formatOutput(suiteResult, cfg)

	// Determine result status
	resultStatus := agent.StatusSuccess
	if suiteResult.OverallStatus == runner.TestStatusFail {
		resultStatus = agent.StatusFailed
	} else if suiteResult.OverallStatus == runner.TestStatusError {
		resultStatus = agent.StatusFailed
	}

	// Build result metadata
	metadata := map[string]any{
		"mode":           cfg.Mode,
		"duration":       suiteResult.Duration().String(),
		"total_tests":    suiteResult.TotalTests(),
		"passed":         suiteResult.TotalPassed(),
		"failed":         suiteResult.TotalFailed(),
		"skipped":        suiteResult.TotalSkipped(),
		"errors":         suiteResult.TotalErrors(),
		"pass_rate":      suiteResult.OverallPassRate(),
		"overall_status": suiteResult.OverallStatus,
		"sdk_summary": map[string]any{
			"total":   suiteResult.SDKSummary.Total,
			"passed":  suiteResult.SDKSummary.Passed,
			"failed":  suiteResult.SDKSummary.Failed,
			"skipped": suiteResult.SDKSummary.Skipped,
			"errors":  suiteResult.SDKSummary.Errors,
		},
		"framework_summary": map[string]any{
			"total":   suiteResult.FrameworkSummary.Total,
			"passed":  suiteResult.FrameworkSummary.Passed,
			"failed":  suiteResult.FrameworkSummary.Failed,
			"skipped": suiteResult.FrameworkSummary.Skipped,
			"errors":  suiteResult.FrameworkSummary.Errors,
		},
	}

	logger.Info("Debug agent execution finished",
		"status", resultStatus,
		"total_duration", time.Since(startTime),
	)

	return agent.Result{
		Status:   resultStatus,
		Output:   output,
		Metadata: metadata,
	}, nil
}

// registerTestModules registers test modules with the runner based on configuration
func registerTestModules(testRunner *runner.Runner, cfg *DebugConfig) error {
	// Register SDK test modules if enabled
	if cfg.IsSDKEnabled() {
		testRunner.RegisterModule(sdk.NewComprehensiveSDKModule())
	}

	// Register Framework test modules if enabled
	if cfg.IsFrameworkEnabled() {
		testRunner.RegisterModule(framework.NewComprehensiveFrameworkModule())
	}

	// Register Network Recon module if enabled
	if cfg.IsNetworkReconEnabled() {
		testRunner.RegisterModule(network.NewNetworkReconModule())
	}

	return nil
}

// formatOutput generates the output string based on configured format
func formatOutput(suiteResult *runner.SuiteResult, cfg *DebugConfig) string {
	var output string

	// Build text output
	if cfg.OutputFormat == OutputText || cfg.OutputFormat == OutputBoth {
		output += formatTextOutput(suiteResult)
	}

	// Build JSON output
	if cfg.OutputFormat == OutputJSON || cfg.OutputFormat == OutputBoth {
		if output != "" {
			output += "\n\n--- JSON Output ---\n\n"
		}
		output += formatJSONOutput(suiteResult)
	}

	return output
}

// formatTextOutput creates human-readable text output
func formatTextOutput(suiteResult *runner.SuiteResult) string {
	output := fmt.Sprintf(`
=== Debug Agent Test Report ===
Mode: %s
Duration: %s
Status: %s

=== Overall Summary ===
Total Tests: %d
Passed: %d (%.1f%%)
Failed: %d
Skipped: %d
Errors: %d

=== SDK Tests ===
Total: %d
Passed: %d
Failed: %d
Skipped: %d
Errors: %d

=== Framework Tests ===
Total: %d
Passed: %d
Failed: %d
Skipped: %d
Errors: %d
`,
		suiteResult.Mode,
		suiteResult.Duration(),
		suiteResult.OverallStatus,
		suiteResult.TotalTests(),
		suiteResult.TotalPassed(),
		suiteResult.OverallPassRate()*100,
		suiteResult.TotalFailed(),
		suiteResult.TotalSkipped(),
		suiteResult.TotalErrors(),
		suiteResult.SDKSummary.Total,
		suiteResult.SDKSummary.Passed,
		suiteResult.SDKSummary.Failed,
		suiteResult.SDKSummary.Skipped,
		suiteResult.SDKSummary.Errors,
		suiteResult.FrameworkSummary.Total,
		suiteResult.FrameworkSummary.Passed,
		suiteResult.FrameworkSummary.Failed,
		suiteResult.FrameworkSummary.Skipped,
		suiteResult.FrameworkSummary.Errors,
	)

	// Add failed test details
	if suiteResult.TotalFailed() > 0 || suiteResult.TotalErrors() > 0 {
		output += "\n=== Failed/Error Tests ===\n"
		for _, result := range suiteResult.Results {
			if result.Status == runner.TestStatusFail || result.Status == runner.TestStatusError {
				output += fmt.Sprintf("\n[%s] %s (Req %s)\n",
					result.Status,
					result.TestName,
					result.RequirementID,
				)
				output += fmt.Sprintf("  Message: %s\n", result.Message)
				if result.Error != nil {
					output += fmt.Sprintf("  Error: %v\n", result.Error)
				}
			}
		}
	}

	return output
}

// formatJSONOutput creates JSON output
func formatJSONOutput(suiteResult *runner.SuiteResult) string {
	// Create a simplified structure for JSON output
	jsonData := map[string]any{
		"mode":           suiteResult.Mode,
		"start_time":     suiteResult.StartTime,
		"end_time":       suiteResult.EndTime,
		"duration":       suiteResult.Duration().String(),
		"overall_status": suiteResult.OverallStatus,
		"summary": map[string]any{
			"total":   suiteResult.TotalTests(),
			"passed":  suiteResult.TotalPassed(),
			"failed":  suiteResult.TotalFailed(),
			"skipped": suiteResult.TotalSkipped(),
			"errors":  suiteResult.TotalErrors(),
		},
		"sdk_summary":       suiteResult.SDKSummary,
		"framework_summary": suiteResult.FrameworkSummary,
		"results":           suiteResult.Results,
	}

	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}

	return string(jsonBytes)
}

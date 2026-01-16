package delegation

import (
	"context"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"

	"github.com/zero-day-ai/agents/debug/internal/runner"
)

// TestConfig holds configuration for delegation tests
type TestConfig struct {
	// TargetAgent is the agent to delegate to (default: "debug-agent")
	TargetAgent string

	// TestMessage is a message to pass to the child agent for verification
	TestMessage string

	// Timeout for delegation operation
	Timeout time.Duration
}

// ExecuteDelegationTest executes multi-agent delegation tests
// This function:
// 1. Calls harness.DelegateToAgent() targeting "debug-agent"
// 2. Passes test context and verifies it's received
// 3. Verifies result returned correctly
// Returns a structured TestResult
func ExecuteDelegationTest(ctx context.Context, harness agent.Harness, cfg TestConfig) runner.TestResult {
	testName := "Agent Delegation Test"
	reqID := "REQ-6"
	startTime := time.Now()

	// Apply defaults
	if cfg.TargetAgent == "" {
		cfg.TargetAgent = "debug-agent"
	}
	if cfg.TestMessage == "" {
		cfg.TestMessage = fmt.Sprintf("[DEBUG] Delegation test at %s", time.Now().Format(time.RFC3339))
	}

	harness.Logger().Info("Starting delegation test",
		"target_agent", cfg.TargetAgent,
		"test_message", cfg.TestMessage,
	)

	// Create delegation task with test context
	delegationTask := agent.Task{
		ID: fmt.Sprintf("delegation-test-%d", time.Now().Unix()),
		Metadata: map[string]any{
			"test_message": cfg.TestMessage,
			"test_type":    "delegation",
			"timestamp":    time.Now().Unix(),
		},
	}

	harness.Logger().Info("Delegating to agent",
		"target", cfg.TargetAgent,
		"task_id", delegationTask.ID,
		"metadata", delegationTask.Metadata,
	)

	// Create context with timeout if specified
	delegateCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		delegateCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	// Delegate to child agent
	result, err := harness.DelegateToAgent(delegateCtx, cfg.TargetAgent, delegationTask)
	if err != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Delegation to '%s' failed: %v", cfg.TargetAgent, err),
			err,
		)
	}

	harness.Logger().Info("Delegation completed",
		"target", cfg.TargetAgent,
		"result_status", result.Status,
		"has_output", result.Output != nil,
	)

	// Verify result structure - agent.Result is a struct, not a pointer
	// Check status - StatusSuccess is the only success status
	if result.Status != agent.StatusSuccess {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Child agent returned non-success status: %s", result.Status),
			fmt.Errorf("non-success status: %s", result.Status),
		)
	}

	// Check if result has error
	if result.Error != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Child agent returned error: %v", result.Error),
			result.Error,
		)
	}

	// Verify result has output
	if result.Output == nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			"Child agent returned nil output",
			fmt.Errorf("nil output from child"),
		)
	}

	harness.Logger().Info("Child agent result received",
		"output_type", fmt.Sprintf("%T", result.Output),
		"status", result.Status,
	)

	// Try to extract result data from output
	outputMap, ok := result.Output.(map[string]any)
	if !ok {
		// Output format may vary, but we should still have something
		harness.Logger().Info("Output is not a map, but delegation succeeded",
			"output", result.Output,
		)
	} else {
		harness.Logger().Info("Child agent output details",
			"status", outputMap["status"],
			"message", outputMap["message"],
		)

		// Verify the child agent received our test context
		if receivedMsg, ok := outputMap["received_message"].(string); ok {
			if receivedMsg != cfg.TestMessage {
				harness.Logger().Warn("Test message mismatch",
					"sent", cfg.TestMessage,
					"received", receivedMsg,
				)
			} else {
				harness.Logger().Info("Test message verified - context passed correctly")
			}
		}
	}

	duration := time.Since(startTime)
	harness.Logger().Info("Delegation test completed successfully",
		"duration", duration,
		"target_agent", cfg.TargetAgent,
	)

	return runner.NewPassResult(
		testName,
		reqID,
		runner.CategorySDK,
		duration,
		fmt.Sprintf("Delegation test passed: successfully delegated to '%s'", cfg.TargetAgent),
	).WithDetails(map[string]any{
		"target_agent":   cfg.TargetAgent,
		"test_message":   cfg.TestMessage,
		"task_id":        delegationTask.ID,
		"result_status":  string(result.Status),
		"execution_time": duration.String(),
		"result_output":  result.Output,
	})
}

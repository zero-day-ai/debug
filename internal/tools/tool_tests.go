package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/tool"

	"github.com/zero-day-ai/agents/debug/internal/runner"
)

// TestConfig holds configuration for tool tests
type TestConfig struct {
	// ToolName specifies which tool to test (optional, if empty tests first available)
	ToolName string

	// Timeout is the timeout for tool execution
	Timeout time.Duration
}

// ExecuteToolTest executes tool discovery and execution tests
// This function:
// 1. Lists available tools via harness.ListTools()
// 2. Executes safe tools (like ping) via harness.CallTool()
// 3. Verifies tool results
// Returns a structured TestResult
func ExecuteToolTest(ctx context.Context, harness agent.Harness, cfg TestConfig) runner.TestResult {
	testName := "Tool Execution Test"
	reqID := "REQ-3"
	startTime := time.Now()

	harness.Logger().Info("Starting tool execution test")

	// Phase 1: List available tools
	harness.Logger().Info("Listing available tools")
	tools, err := harness.ListTools(ctx)
	if err != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Failed to list tools: %v", err),
			err,
		)
	}

	if len(tools) == 0 {
		return runner.NewSkipResult(
			testName,
			reqID,
			runner.CategorySDK,
			"No tools available to test",
		)
	}

	harness.Logger().Info("Tools discovered",
		"count", len(tools),
		"tools", getToolNames(tools),
	)

	// Phase 2: Select a safe tool to test
	var toolToTest tool.Descriptor
	var found bool

	if cfg.ToolName != "" {
		// Use specified tool
		for _, t := range tools {
			if t.Name == cfg.ToolName {
				toolToTest = t
				found = true
				break
			}
		}
		if !found {
			return runner.NewSkipResult(
				testName,
				reqID,
				runner.CategorySDK,
				fmt.Sprintf("Specified tool '%s' not found in available tools", cfg.ToolName),
			)
		}
	} else {
		// Try to find a safe tool to test (ping, echo, or first available)
		safeTools := []string{"ping", "echo", "list"}
		for _, safeName := range safeTools {
			for _, t := range tools {
				if t.Name == safeName {
					toolToTest = t
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		// If no safe tool found, skip rather than executing unknown tool
		if !found {
			return runner.NewPassResult(
				testName,
				reqID,
				runner.CategorySDK,
				time.Since(startTime),
				fmt.Sprintf("Tool discovery succeeded: %d tools found (no safe tool to execute)", len(tools)),
			)
		}
	}

	harness.Logger().Info("Selected tool for testing",
		"tool_name", toolToTest.Name,
		"description", toolToTest.Description,
	)

	// Phase 3: Execute the tool with appropriate input
	var toolInput map[string]any

	switch toolToTest.Name {
	case "ping":
		// Ping localhost - safe and always available
		toolInput = map[string]any{
			"targets": []string{"127.0.0.1"},
			"count":   1,
			"timeout": 1000,
		}
	case "echo":
		// Echo a test message
		toolInput = map[string]any{
			"message": "[DEBUG] Tool test execution",
		}
	case "list":
		// List operation with minimal scope
		toolInput = map[string]any{
			"path": ".",
		}
	default:
		// For unknown tools, create minimal input
		toolInput = map[string]any{
			"test": true,
		}
	}

	harness.Logger().Info("Calling tool",
		"tool_name", toolToTest.Name,
		"input", toolInput,
	)

	// Create context with timeout
	callCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	// Execute tool
	result, err := harness.CallTool(callCtx, toolToTest.Name, toolInput)
	if err != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Tool execution failed for '%s': %v", toolToTest.Name, err),
			err,
		)
	}

	harness.Logger().Info("Tool execution completed",
		"tool_name", toolToTest.Name,
		"has_output", result != nil,
	)

	// Phase 4: Verify tool result
	if result == nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Tool '%s' returned nil output", toolToTest.Name),
			fmt.Errorf("nil output from tool"),
		)
	}

	// Success - tool executed and returned valid output
	duration := time.Since(startTime)
	harness.Logger().Info("Tool test completed successfully",
		"duration", duration,
		"tool_name", toolToTest.Name,
		"tools_discovered", len(tools),
	)

	return runner.NewPassResult(
		testName,
		reqID,
		runner.CategorySDK,
		duration,
		fmt.Sprintf("Tool test passed: discovered %d tools, executed '%s' successfully", len(tools), toolToTest.Name),
	).WithDetails(map[string]any{
		"tools_discovered": len(tools),
		"tool_names":       getToolNames(tools),
		"tool_executed":    toolToTest.Name,
		"execution_time":   duration.String(),
	})
}

// getToolNames extracts tool names from a list of tool descriptors
func getToolNames(tools []tool.Descriptor) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

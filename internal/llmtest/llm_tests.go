package llmtest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/llm"
	"github.com/zero-day-ai/sdk/tool"

	"github.com/zero-day-ai/agents/debug/internal/runner"
)

// TestConfig holds configuration for LLM tests
type TestConfig struct {
	// Method specifies which LLM method to test: "complete", "structured", "with_tools"
	Method string

	// Provider specifies which LLM provider to use (default: "primary")
	Provider string

	// Model optionally specifies a specific model to use
	Model string
}

// TestStructuredOutput is the schema for structured output tests
type TestStructuredOutput struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Value   int    `json:"value"`
}

// ExecuteLLMTest executes LLM completion tests based on the configured method
// This function tests:
// - "complete": harness.Complete() with a simple prompt
// - "structured": harness.CompleteStructured() with schema validation
// - "with_tools": harness.CompleteWithTools() with tool availability
// Uses minimal prompts to reduce cost
// Returns a structured TestResult
func ExecuteLLMTest(ctx context.Context, harness agent.Harness, cfg TestConfig) runner.TestResult {
	testName := fmt.Sprintf("LLM Test (%s)", cfg.Method)
	reqID := "REQ-5"
	startTime := time.Now()

	// Apply defaults
	if cfg.Provider == "" {
		cfg.Provider = "primary"
	}
	if cfg.Method == "" {
		cfg.Method = "complete"
	}

	harness.Logger().Info("Starting LLM test",
		"method", cfg.Method,
		"provider", cfg.Provider,
	)

	// Route to appropriate test method
	switch cfg.Method {
	case "complete":
		return testComplete(ctx, harness, cfg, testName, reqID, startTime)
	case "structured":
		return testStructured(ctx, harness, cfg, testName, reqID, startTime)
	case "with_tools":
		return testWithTools(ctx, harness, cfg, testName, reqID, startTime)
	default:
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Unknown test method: %s (must be 'complete', 'structured', or 'with_tools')", cfg.Method),
			fmt.Errorf("invalid method"),
		)
	}
}

// testComplete tests harness.Complete() with a minimal prompt
func testComplete(ctx context.Context, harness agent.Harness, cfg TestConfig, testName, reqID string, startTime time.Time) runner.TestResult {
	harness.Logger().Info("Testing Complete() method")

	// Use minimal prompt to reduce cost
	prompt := "Reply with exactly: OK"

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	harness.Logger().Info("Calling Complete()",
		"provider", cfg.Provider,
		"prompt_length", len(prompt),
	)

	response, err := harness.Complete(ctx, cfg.Provider, messages)
	if err != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Complete() failed: %v", err),
			err,
		)
	}

	harness.Logger().Info("Complete() response received",
		"response_length", len(response.Content),
		"response_preview", truncate(response.Content, 50),
		"finish_reason", response.FinishReason,
	)

	// Verify response is not empty
	if response.Content == "" {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			"Complete() returned empty response content",
			fmt.Errorf("empty response"),
		)
	}

	duration := time.Since(startTime)
	harness.Logger().Info("Complete() test passed",
		"duration", duration,
		"response_length", len(response.Content),
		"tokens_input", response.Usage.InputTokens,
		"tokens_output", response.Usage.OutputTokens,
	)

	return runner.NewPassResult(
		testName,
		reqID,
		runner.CategorySDK,
		duration,
		fmt.Sprintf("Complete() test passed: received %d character response", len(response.Content)),
	).WithDetails(map[string]any{
		"method":          "complete",
		"provider":        cfg.Provider,
		"prompt_length":   len(prompt),
		"response_length": len(response.Content),
		"input_tokens":    response.Usage.InputTokens,
		"output_tokens":   response.Usage.OutputTokens,
		"execution_time":  duration.String(),
	})
}

// testStructured tests harness.CompleteStructured() with schema validation
func testStructured(ctx context.Context, harness agent.Harness, cfg TestConfig, testName, reqID string, startTime time.Time) runner.TestResult {
	harness.Logger().Info("Testing CompleteStructured() method")

	// Use minimal prompt that requests structured output
	prompt := "Return a status of 'success', a message of 'test', and a value of 42."

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	harness.Logger().Info("Calling CompleteStructured()",
		"provider", cfg.Provider,
		"schema", "TestStructuredOutput",
	)

	result, err := harness.CompleteStructured(ctx, cfg.Provider, messages, TestStructuredOutput{})
	if err != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("CompleteStructured() failed: %v", err),
			err,
		)
	}

	// Type assert the result
	output, ok := result.(*TestStructuredOutput)
	if !ok {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("CompleteStructured() returned unexpected type: %T", result),
			fmt.Errorf("type assertion failed"),
		)
	}

	harness.Logger().Info("CompleteStructured() response received",
		"status", output.Status,
		"message", output.Message,
		"value", output.Value,
	)

	// Verify structured output has expected fields populated
	if output.Status == "" {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			"CompleteStructured() returned empty status field",
			fmt.Errorf("empty status"),
		)
	}

	duration := time.Since(startTime)
	harness.Logger().Info("CompleteStructured() test passed",
		"duration", duration,
		"status", output.Status,
	)

	return runner.NewPassResult(
		testName,
		reqID,
		runner.CategorySDK,
		duration,
		fmt.Sprintf("CompleteStructured() test passed: status=%s, message=%s, value=%d",
			output.Status, output.Message, output.Value),
	).WithDetails(map[string]any{
		"method":         "structured",
		"provider":       cfg.Provider,
		"status":         output.Status,
		"message":        output.Message,
		"value":          output.Value,
		"execution_time": duration.String(),
	})
}

// testWithTools tests harness.CompleteWithTools() with tool availability
func testWithTools(ctx context.Context, harness agent.Harness, cfg TestConfig, testName, reqID string, startTime time.Time) runner.TestResult {
	harness.Logger().Info("Testing CompleteWithTools() method")

	// First, discover available tools
	toolDescriptors, err := harness.ListTools(ctx)
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

	if len(toolDescriptors) == 0 {
		return runner.NewSkipResult(
			testName,
			reqID,
			runner.CategorySDK,
			"No tools available for CompleteWithTools() test",
		)
	}

	// Convert tool descriptors to LLM tool definitions
	// The ToolDef structure directly matches the tool descriptor fields
	tools := make([]llm.ToolDef, 0, len(toolDescriptors))
	for _, td := range toolDescriptors {
		// Convert schema.JSON to map[string]any for Parameters field
		// We need to marshal the schema to JSON and unmarshal it to a map
		schemaBytes, err := json.Marshal(td.InputSchema)
		if err != nil {
			harness.Logger().Warn("Failed to marshal tool schema",
				"tool", td.Name,
				"error", err,
			)
			continue
		}

		var params map[string]any
		if err := json.Unmarshal(schemaBytes, &params); err != nil {
			harness.Logger().Warn("Failed to unmarshal tool schema",
				"tool", td.Name,
				"error", err,
			)
			continue
		}

		tools = append(tools, llm.ToolDef{
			Name:        td.Name,
			Description: td.Description,
			Parameters:  params,
		})
	}

	harness.Logger().Info("Tools available for LLM",
		"count", len(tools),
		"tool_names", getToolNames(toolDescriptors),
	)

	// Create a prompt that might trigger tool use (but keep it minimal)
	// We'll ask for something that could use a tool, but the LLM may choose not to
	prompt := "Reply with: 'No tools needed for this response.'"

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	harness.Logger().Info("Calling CompleteWithTools()",
		"provider", cfg.Provider,
		"available_tools", len(tools),
	)

	response, err := harness.CompleteWithTools(ctx, cfg.Provider, messages, tools)
	if err != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("CompleteWithTools() failed: %v", err),
			err,
		)
	}

	harness.Logger().Info("CompleteWithTools() response received",
		"response_length", len(response.Content),
		"tool_calls", len(response.ToolCalls),
		"finish_reason", response.FinishReason,
	)

	// The response should be valid even if no tools were called
	// This tests that the method works with tools available
	if response.Content == "" && len(response.ToolCalls) == 0 {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			"CompleteWithTools() returned empty response and no tool calls",
			fmt.Errorf("no response or tool calls"),
		)
	}

	duration := time.Since(startTime)

	var message string
	if len(response.ToolCalls) > 0 {
		harness.Logger().Info("CompleteWithTools() test passed with tool calls",
			"duration", duration,
			"tool_calls", len(response.ToolCalls),
		)
		message = fmt.Sprintf("CompleteWithTools() test passed: %d tools available, %d tool calls made",
			len(tools), len(response.ToolCalls))
	} else {
		harness.Logger().Info("CompleteWithTools() test passed without tool calls",
			"duration", duration,
			"response_length", len(response.Content),
		)
		message = fmt.Sprintf("CompleteWithTools() test passed: %d tools available, direct response received",
			len(tools))
	}

	return runner.NewPassResult(
		testName,
		reqID,
		runner.CategorySDK,
		duration,
		message,
	).WithDetails(map[string]any{
		"method":          "with_tools",
		"provider":        cfg.Provider,
		"tools_available": len(tools),
		"tool_calls":      len(response.ToolCalls),
		"response_length": len(response.Content),
		"input_tokens":    response.Usage.InputTokens,
		"output_tokens":   response.Usage.OutputTokens,
		"execution_time":  duration.String(),
	})
}

// truncate truncates a string to a maximum length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getToolNames extracts tool names from a list of tool descriptors
func getToolNames(tools []tool.Descriptor) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

package health

import (
	"context"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"
)

// TestResult represents the outcome of a health check test
type TestResult struct {
	Component string
	Success   bool
	Duration  time.Duration
	Message   string
	Error     error
	Details   map[string]any
}

// ExecuteHealthCheck performs health checks on Gibson infrastructure components
// based on the specified component in cfg.Component.
// Valid components: "graphrag", "tools", "memory", "plugins"
func ExecuteHealthCheck(ctx context.Context, h agent.Harness, component string) (*TestResult, error) {
	result := &TestResult{
		Component: component,
		Details:   make(map[string]any),
	}
	startTime := time.Now()

	logger := h.Logger()
	logger.Info("Starting health check", "component", component)

	switch component {
	case "graphrag":
		result = checkGraphRAG(ctx, h)
	case "tools":
		result = checkTools(ctx, h)
	case "memory":
		result = checkMemory(ctx, h)
	case "plugins":
		result = checkPlugins(ctx, h)
	default:
		result.Success = false
		result.Error = fmt.Errorf("unknown component: %s", component)
		result.Message = fmt.Sprintf("Unknown health check component: %s", component)
	}

	result.Duration = time.Since(startTime)
	logger.Info("Health check completed",
		"component", component,
		"success", result.Success,
		"duration", result.Duration,
	)

	return result, nil
}

// checkGraphRAG verifies GraphRAG and Neo4j connectivity
func checkGraphRAG(ctx context.Context, h agent.Harness) *TestResult {
	result := &TestResult{
		Component: "graphrag",
		Details:   make(map[string]any),
	}
	startTime := time.Now()

	logger := h.Logger()

	// Check GraphRAG health status
	health := h.GraphRAGHealth(ctx)
	result.Duration = time.Since(startTime)

	if health.Status == "healthy" {
		result.Success = true
		result.Message = "GraphRAG and Neo4j are healthy and accessible"
		result.Details["status"] = health.Status
		result.Details["details"] = health.Details
		logger.Info("GraphRAG health check passed", "details", health.Details)
	} else {
		result.Success = false
		result.Error = fmt.Errorf("graphrag health check failed: %s", health.Details)
		result.Message = fmt.Sprintf("GraphRAG health check failed: %s", health.Details)
		result.Details["status"] = health.Status
		result.Details["details"] = health.Details
		logger.Error("GraphRAG health check failed", "status", health.Status, "details", health.Details)
	}

	return result
}

// checkTools verifies tool discovery and availability
func checkTools(ctx context.Context, h agent.Harness) *TestResult {
	result := &TestResult{
		Component: "tools",
		Details:   make(map[string]any),
	}
	startTime := time.Now()

	logger := h.Logger()

	// List all available tools
	tools, err := h.ListTools(ctx)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Failed to list tools: %v", err)
		logger.Error("Tool discovery failed", "error", err)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully discovered %d tools", len(tools))
	result.Details["tool_count"] = len(tools)

	// Collect tool names
	toolNames := make([]string, 0, len(tools))
	for _, t := range tools {
		toolNames = append(toolNames, t.Name)
	}
	result.Details["tools"] = toolNames

	logger.Info("Tool discovery successful",
		"tool_count", len(tools),
		"tools", toolNames,
	)

	return result
}

// checkMemory verifies memory tier accessibility
func checkMemory(ctx context.Context, h agent.Harness) *TestResult {
	result := &TestResult{
		Component: "memory",
		Details:   make(map[string]any),
	}
	startTime := time.Now()

	logger := h.Logger()

	// Get memory store
	mem := h.Memory()
	if mem == nil {
		result.Duration = time.Since(startTime)
		result.Success = false
		result.Error = fmt.Errorf("memory store is nil")
		result.Message = "Memory store is not available"
		logger.Error("Memory store is nil")
		return result
	}

	// Test Working memory with a simple ping
	testKey := "health_check_test"
	testValue := map[string]any{"timestamp": time.Now().Unix(), "test": "working_memory"}

	// Try Working tier
	workingErr := mem.Working().Set(ctx, testKey, testValue)
	workingAccessible := workingErr == nil
	if workingAccessible {
		// Clean up test data
		_ = mem.Working().Delete(ctx, testKey)
	}

	// Try Mission tier
	missionErr := mem.Mission().Set(ctx, testKey, testValue, nil)
	missionAccessible := missionErr == nil
	if missionAccessible {
		// Clean up test data
		_ = mem.Mission().Delete(ctx, testKey)
	}

	// Try LongTerm tier (Store method)
	_, longTermErr := mem.LongTerm().Store(ctx, testKey, nil)
	longTermAccessible := longTermErr == nil

	result.Duration = time.Since(startTime)

	// Check results
	allTiersAccessible := workingAccessible && missionAccessible && longTermAccessible
	result.Success = allTiersAccessible

	result.Details["working_accessible"] = workingAccessible
	result.Details["mission_accessible"] = missionAccessible
	result.Details["long_term_accessible"] = longTermAccessible

	if allTiersAccessible {
		result.Message = "All memory tiers (Working, Mission, LongTerm) are accessible"
		logger.Info("Memory health check passed", "all_tiers", "accessible")
	} else {
		errors := make([]string, 0)
		if !workingAccessible {
			errors = append(errors, fmt.Sprintf("Working: %v", workingErr))
		}
		if !missionAccessible {
			errors = append(errors, fmt.Sprintf("Mission: %v", missionErr))
		}
		if !longTermAccessible {
			errors = append(errors, fmt.Sprintf("LongTerm: %v", longTermErr))
		}
		result.Message = fmt.Sprintf("Memory tier accessibility issues: %v", errors)
		result.Error = fmt.Errorf("memory tiers not fully accessible")
		result.Details["errors"] = errors
		logger.Error("Memory health check failed", "errors", errors)
	}

	return result
}

// checkPlugins verifies plugin discovery and availability
func checkPlugins(ctx context.Context, h agent.Harness) *TestResult {
	result := &TestResult{
		Component: "plugins",
		Details:   make(map[string]any),
	}
	startTime := time.Now()

	logger := h.Logger()

	// List all available plugins
	plugins, err := h.ListPlugins(ctx)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Failed to list plugins: %v", err)
		logger.Error("Plugin discovery failed", "error", err)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully discovered %d plugins", len(plugins))
	result.Details["plugin_count"] = len(plugins)

	// Collect plugin names
	pluginNames := make([]string, 0, len(plugins))
	for _, p := range plugins {
		pluginNames = append(pluginNames, p.Name)
	}
	result.Details["plugins"] = pluginNames

	logger.Info("Plugin discovery successful",
		"plugin_count", len(plugins),
		"plugins", pluginNames,
	)

	return result
}

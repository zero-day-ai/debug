package delegation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"
)

// ChildResult represents the result of a child execution
type ChildResult struct {
	Status          string         `json:"status"`
	ReceivedTask    string         `json:"received_task"`
	ReceivedContext map[string]any `json:"received_context"`
	ProcessedAt     time.Time      `json:"processed_at"`
	EchoMessage     string         `json:"echo_message"`
}

// ExecuteChild is the execution handler for ModeChild.
// This function is called when the debug agent is delegated to as a child agent.
// It performs a simple operation (echo back context) to verify delegation works.
func ExecuteChild(ctx context.Context, h agent.Harness, task agent.Task) (agent.Result, error) {
	logger := h.Logger()
	startTime := time.Now()

	logger.Info("Child execution started",
		"task_id", task.ID,
	)

	// Log received context and task details for debugging
	logger.Debug("Received task details",
		"task_id", task.ID,
		"context_keys", getMapKeys(task.Context),
		"metadata_keys", getMapKeys(task.Metadata),
	)

	// Perform simple operation: echo back the context
	echoMessage := fmt.Sprintf("Child agent received task: %s", task.ID)

	// Extract any specific data from context
	if len(task.Context) > 0 {
		logger.Info("Processing task context",
			"context_entries", len(task.Context),
		)

		// Log each context entry
		for key, value := range task.Context {
			logger.Debug("Context entry",
				"key", key,
				"value_type", fmt.Sprintf("%T", value),
			)
		}
	}

	// Create structured result for parent to verify
	result := ChildResult{
		Status:          "success",
		ReceivedTask:    task.ID,
		ReceivedContext: task.Context,
		ProcessedAt:     time.Now(),
		EchoMessage:     echoMessage,
	}

	// Marshal result to JSON for readable output
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal child result",
			"error", err,
		)
		return agent.Result{
			Status: agent.StatusFailed,
			Output: fmt.Sprintf("Failed to marshal result: %v", err),
			Error:  err,
		}, nil
	}

	duration := time.Since(startTime)
	logger.Info("Child execution completed",
		"duration", duration,
		"status", "success",
	)

	return agent.Result{
		Status: agent.StatusSuccess,
		Output: string(resultJSON),
		Metadata: map[string]any{
			"duration":         duration.String(),
			"received_task":    task.ID,
			"context_entries":  len(task.Context),
			"metadata_entries": len(task.Metadata),
			"processed_at":     result.ProcessedAt.Format(time.RFC3339),
		},
	}, nil
}

// getMapKeys is a helper function to extract keys from a map
func getMapKeys(m map[string]any) []string {
	if m == nil {
		return []string{}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

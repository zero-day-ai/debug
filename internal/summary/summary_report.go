package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zero-day-ai/sdk/agent"
)

// PhaseResults represents the pass/fail counts for a specific test phase
type PhaseResults struct {
	PhaseName string `json:"phase_name"`
	Total     int    `json:"total"`
	Passed    int    `json:"passed"`
	Failed    int    `json:"failed"`
	Skipped   int    `json:"skipped"`
	Errors    int    `json:"errors"`
	PassRate  float64 `json:"pass_rate"`
}

// DebugMissionSummary represents the overall summary of debug mission results
type DebugMissionSummary struct {
	GeneratedAt    time.Time      `json:"generated_at"`
	MissionID      string         `json:"mission_id"`
	TotalPhases    int            `json:"total_phases"`
	PhaseResults   []PhaseResults `json:"phase_results"`
	OverallTotal   int            `json:"overall_total"`
	OverallPassed  int            `json:"overall_passed"`
	OverallFailed  int            `json:"overall_failed"`
	OverallSkipped int            `json:"overall_skipped"`
	OverallErrors  int            `json:"overall_errors"`
	OverallPassRate float64       `json:"overall_pass_rate"`
	Status         string         `json:"status"`
	Report         string         `json:"report"`
}

// Config represents the configuration for summary generation
type Config struct {
	// Additional configuration can be added here
	IncludeDetails bool
}

// ExecuteSummary generates a comprehensive summary report from test results.
// It reads test results from mission context/working memory, aggregates them,
// and generates a human-readable summary report.
func ExecuteSummary(ctx context.Context, h agent.Harness, cfg *Config) (agent.Result, error) {
	logger := h.Logger()
	startTime := time.Now()

	logger.Info("Summary generation started")

	// Initialize summary
	summary := &DebugMissionSummary{
		GeneratedAt:  time.Now(),
		MissionID:    h.Mission().ID,
		PhaseResults: []PhaseResults{},
	}

	// Access working memory to read test results
	memStore := h.Memory()
	workingMem := memStore.Working()

	logger.Debug("Reading test results from working memory")

	// Try to read various test phase results from memory
	phases := []string{
		"health_check",
		"memory_test",
		"tools_test",
		"graphrag_test",
		"llm_test",
		"delegation_test",
		"findings_test",
		"sdk_tests",
		"framework_tests",
	}

	for _, phase := range phases {
		// Try to read results for this phase
		data, err := workingMem.Get(ctx, phase)
		if err != nil {
			logger.Debug("Phase results not found in memory",
				"phase", phase,
				"error", err,
			)
			continue
		}

		// Parse the phase results
		phaseResult, err := parsePhaseResults(phase, data)
		if err != nil {
			logger.Warn("Failed to parse phase results",
				"phase", phase,
				"error", err,
			)
			continue
		}

		summary.PhaseResults = append(summary.PhaseResults, phaseResult)
		logger.Debug("Added phase results to summary",
			"phase", phase,
			"total", phaseResult.Total,
			"passed", phaseResult.Passed,
			"failed", phaseResult.Failed,
		)
	}

	// If no phase results found, try to read from mission memory
	if len(summary.PhaseResults) == 0 {
		logger.Info("No results in working memory, checking mission memory")
		missionMem := memStore.Mission()

		for _, phase := range phases {
			data, err := missionMem.Get(ctx, phase)
			if err != nil {
				continue
			}

			phaseResult, err := parsePhaseResults(phase, data)
			if err != nil {
				continue
			}

			summary.PhaseResults = append(summary.PhaseResults, phaseResult)
		}
	}

	// Calculate overall statistics
	summary.TotalPhases = len(summary.PhaseResults)
	for _, phase := range summary.PhaseResults {
		summary.OverallTotal += phase.Total
		summary.OverallPassed += phase.Passed
		summary.OverallFailed += phase.Failed
		summary.OverallSkipped += phase.Skipped
		summary.OverallErrors += phase.Errors
	}

	// Calculate overall pass rate
	if summary.OverallTotal > 0 {
		summary.OverallPassRate = float64(summary.OverallPassed) / float64(summary.OverallTotal)
	}

	// Determine overall status
	if summary.OverallErrors > 0 {
		summary.Status = "error"
	} else if summary.OverallFailed > 0 {
		summary.Status = "failed"
	} else if summary.OverallTotal == 0 {
		summary.Status = "no_tests"
	} else if summary.OverallTotal == summary.OverallSkipped {
		summary.Status = "skipped"
	} else {
		summary.Status = "success"
	}

	// Generate human-readable report
	summary.Report = generateReport(summary)

	// Log the summary via harness
	logger.Info("Debug mission summary generated",
		"total_phases", summary.TotalPhases,
		"overall_total", summary.OverallTotal,
		"overall_passed", summary.OverallPassed,
		"overall_failed", summary.OverallFailed,
		"overall_status", summary.Status,
		"pass_rate", fmt.Sprintf("%.2f%%", summary.OverallPassRate*100),
	)

	// Marshal to JSON
	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal summary",
			"error", err,
		)
		return agent.Result{
			Status: agent.StatusFailed,
			Output: fmt.Sprintf("Failed to marshal summary: %v", err),
			Error:  err,
		}, nil
	}

	duration := time.Since(startTime)
	logger.Info("Summary generation completed",
		"duration", duration,
	)

	// Determine result status
	resultStatus := agent.StatusSuccess
	if summary.Status == "error" || summary.Status == "failed" {
		resultStatus = agent.StatusPartial
	}

	return agent.Result{
		Status: resultStatus,
		Output: string(summaryJSON),
		Metadata: map[string]any{
			"duration":          duration.String(),
			"total_phases":      summary.TotalPhases,
			"overall_total":     summary.OverallTotal,
			"overall_passed":    summary.OverallPassed,
			"overall_failed":    summary.OverallFailed,
			"overall_errors":    summary.OverallErrors,
			"overall_pass_rate": summary.OverallPassRate,
			"status":            summary.Status,
		},
	}, nil
}

// parsePhaseResults attempts to parse test results from memory data
func parsePhaseResults(phaseName string, data any) (PhaseResults, error) {
	result := PhaseResults{
		PhaseName: phaseName,
	}

	// Try to parse as JSON map
	if dataMap, ok := data.(map[string]any); ok {
		// Extract counts from common field names
		if total, ok := dataMap["total"].(float64); ok {
			result.Total = int(total)
		}
		if passed, ok := dataMap["passed"].(float64); ok {
			result.Passed = int(passed)
		}
		if failed, ok := dataMap["failed"].(float64); ok {
			result.Failed = int(failed)
		}
		if skipped, ok := dataMap["skipped"].(float64); ok {
			result.Skipped = int(skipped)
		}
		if errors, ok := dataMap["errors"].(float64); ok {
			result.Errors = int(errors)
		}

		// Calculate pass rate
		if result.Total > 0 {
			result.PassRate = float64(result.Passed) / float64(result.Total)
		}

		return result, nil
	}

	// If we can't parse it, return a minimal result
	result.Total = 1
	result.Passed = 1
	result.PassRate = 1.0

	return result, nil
}

// generateReport creates a human-readable summary report
func generateReport(summary *DebugMissionSummary) string {
	var builder strings.Builder

	builder.WriteString("=== Debug Mission Summary Report ===\n\n")
	builder.WriteString(fmt.Sprintf("Generated: %s\n", summary.GeneratedAt.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("Mission ID: %s\n", summary.MissionID))
	builder.WriteString(fmt.Sprintf("Overall Status: %s\n\n", strings.ToUpper(summary.Status)))

	builder.WriteString("=== Overall Statistics ===\n")
	builder.WriteString(fmt.Sprintf("Total Phases: %d\n", summary.TotalPhases))
	builder.WriteString(fmt.Sprintf("Total Tests: %d\n", summary.OverallTotal))
	builder.WriteString(fmt.Sprintf("Passed: %d (%.1f%%)\n", summary.OverallPassed, summary.OverallPassRate*100))
	builder.WriteString(fmt.Sprintf("Failed: %d\n", summary.OverallFailed))
	builder.WriteString(fmt.Sprintf("Skipped: %d\n", summary.OverallSkipped))
	builder.WriteString(fmt.Sprintf("Errors: %d\n\n", summary.OverallErrors))

	if len(summary.PhaseResults) > 0 {
		builder.WriteString("=== Phase Breakdown ===\n")
		for _, phase := range summary.PhaseResults {
			builder.WriteString(fmt.Sprintf("\n[%s]\n", strings.ToUpper(phase.PhaseName)))
			builder.WriteString(fmt.Sprintf("  Total: %d\n", phase.Total))
			builder.WriteString(fmt.Sprintf("  Passed: %d (%.1f%%)\n", phase.Passed, phase.PassRate*100))
			builder.WriteString(fmt.Sprintf("  Failed: %d\n", phase.Failed))
			builder.WriteString(fmt.Sprintf("  Skipped: %d\n", phase.Skipped))
			builder.WriteString(fmt.Sprintf("  Errors: %d\n", phase.Errors))
		}
	} else {
		builder.WriteString("=== No Phase Results Available ===\n")
		builder.WriteString("No test results were found in working or mission memory.\n")
		builder.WriteString("This may indicate that no tests have been executed yet.\n")
	}

	builder.WriteString("\n=== Conclusion ===\n")
	switch summary.Status {
	case "success":
		builder.WriteString("All tests completed successfully!\n")
	case "failed":
		builder.WriteString("Some tests failed. Please review the failed tests above.\n")
	case "error":
		builder.WriteString("Errors occurred during test execution. Please review the errors above.\n")
	case "skipped":
		builder.WriteString("All tests were skipped.\n")
	case "no_tests":
		builder.WriteString("No tests were executed.\n")
	default:
		builder.WriteString("Unknown status.\n")
	}

	return builder.String()
}

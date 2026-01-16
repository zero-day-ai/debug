package findings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/finding"
)

// TestResult represents the result of a findings test execution
type TestResult struct {
	Status       string    `json:"status"`
	FindingID    string    `json:"finding_id"`
	FindingTitle string    `json:"finding_title"`
	SubmittedAt  time.Time `json:"submitted_at"`
	VerifiedAt   time.Time `json:"verified_at"`
	Message      string    `json:"message"`
}

// Config represents the configuration for findings tests
type Config struct {
	// Prefix to use in finding titles (default: "[DEBUG]")
	Prefix string
}

// ExecuteFindingsTest executes findings submission tests.
// It submits a test finding via harness.SubmitFinding() and verifies it was stored.
func ExecuteFindingsTest(ctx context.Context, h agent.Harness, cfg *Config) (agent.Result, error) {
	logger := h.Logger()
	startTime := time.Now()

	logger.Info("Findings test execution started")

	// Use default prefix if not provided
	prefix := "[DEBUG]"
	if cfg != nil && cfg.Prefix != "" {
		prefix = cfg.Prefix
	}

	// Create a unique test finding with timestamp for uniqueness
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
	findingID := uuid.New().String()

	testFinding := &finding.Finding{
		ID:          findingID,
		MissionID:   h.Mission().ID,
		AgentName:   "debug-agent",
		Title:       fmt.Sprintf("%s Test Finding - %s", prefix, timestamp),
		Description: fmt.Sprintf("This is a test finding created by the debug agent at %s to verify the finding submission functionality.", timestamp),
		Category:    finding.CategoryInformationDisclosure,
		Subcategory: "test",
		Severity:    finding.SeverityInfo,
		Confidence:  0.99,
		RiskScore:   1.0,
		Technique:   "debug-test",
		Tags:        []string{"debug", "test", "verification"},
		Status:      finding.StatusConfirmed,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add evidence to the finding
	testFinding.Evidence = []finding.Evidence{
		{
			Type:      finding.EvidenceLog,
			Title:     "Debug agent test execution log",
			Content:   fmt.Sprintf("Test finding submitted at %s", timestamp),
			Timestamp: time.Now(),
		},
	}

	logger.Info("Submitting test finding",
		"finding_id", findingID,
		"title", testFinding.Title,
		"severity", testFinding.Severity,
	)

	// Submit the finding via harness
	submitTime := time.Now()
	err := h.SubmitFinding(ctx, testFinding)
	if err != nil {
		logger.Error("Failed to submit finding",
			"error", err,
			"finding_id", findingID,
		)

		result := TestResult{
			Status:       "failed",
			FindingID:    findingID,
			FindingTitle: testFinding.Title,
			SubmittedAt:  submitTime,
			Message:      fmt.Sprintf("Failed to submit finding: %v", err),
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return agent.Result{
			Status: agent.StatusFailed,
			Output: string(resultJSON),
			Error:  err,
			Metadata: map[string]any{
				"duration":    time.Since(startTime).String(),
				"finding_id":  findingID,
				"error":       err.Error(),
			},
		}, nil
	}

	logger.Info("Finding submitted successfully",
		"finding_id", findingID,
		"submit_duration", time.Since(submitTime),
	)

	// Verify the finding was stored by querying it back
	verifyTime := time.Now()
	filter := finding.Filter{
		MissionID: h.Mission().ID,
		AgentName: "debug-agent",
	}

	findings, err := h.GetFindings(ctx, filter)
	if err != nil {
		logger.Warn("Failed to verify finding storage",
			"error", err,
			"finding_id", findingID,
		)

		// Still consider this a success since submission worked
		result := TestResult{
			Status:       "success_unverified",
			FindingID:    findingID,
			FindingTitle: testFinding.Title,
			SubmittedAt:  submitTime,
			VerifiedAt:   verifyTime,
			Message:      "Finding submitted but verification failed",
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return agent.Result{
			Status: agent.StatusPartial,
			Output: string(resultJSON),
			Metadata: map[string]any{
				"duration":        time.Since(startTime).String(),
				"finding_id":      findingID,
				"verification_error": err.Error(),
			},
		}, nil
	}

	// Check if our finding is in the results
	foundOurFinding := false
	for _, f := range findings {
		if f.ID == findingID {
			foundOurFinding = true
			logger.Info("Finding verified in storage",
				"finding_id", findingID,
				"title", f.Title,
			)
			break
		}
	}

	duration := time.Since(startTime)

	if !foundOurFinding {
		logger.Warn("Finding not found in query results",
			"finding_id", findingID,
			"total_findings", len(findings),
		)

		result := TestResult{
			Status:       "success_not_found",
			FindingID:    findingID,
			FindingTitle: testFinding.Title,
			SubmittedAt:  submitTime,
			VerifiedAt:   verifyTime,
			Message:      fmt.Sprintf("Finding submitted but not found in query results (found %d other findings)", len(findings)),
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return agent.Result{
			Status: agent.StatusPartial,
			Output: string(resultJSON),
			Metadata: map[string]any{
				"duration":       duration.String(),
				"finding_id":     findingID,
				"total_findings": len(findings),
			},
		}, nil
	}

	// Success - finding submitted and verified
	result := TestResult{
		Status:       "success",
		FindingID:    findingID,
		FindingTitle: testFinding.Title,
		SubmittedAt:  submitTime,
		VerifiedAt:   verifyTime,
		Message:      "Finding successfully submitted and verified in storage",
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal result",
			"error", err,
		)
		return agent.Result{
			Status: agent.StatusFailed,
			Output: fmt.Sprintf("Failed to marshal result: %v", err),
			Error:  err,
		}, nil
	}

	logger.Info("Findings test execution completed",
		"duration", duration,
		"status", "success",
		"finding_id", findingID,
	)

	return agent.Result{
		Status: agent.StatusSuccess,
		Output: string(resultJSON),
		Findings: []string{findingID},
		Metadata: map[string]any{
			"duration":       duration.String(),
			"finding_id":     findingID,
			"finding_title":  testFinding.Title,
			"submitted_at":   submitTime.Format(time.RFC3339),
			"verified_at":    verifyTime.Format(time.RFC3339),
			"total_findings": len(findings),
		},
	}, nil
}

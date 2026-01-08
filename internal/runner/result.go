package runner

import (
	"fmt"
	"time"
)

// TestStatus represents the outcome of a test execution
type TestStatus string

const (
	// TestStatusPass indicates the test passed successfully
	TestStatusPass TestStatus = "pass"

	// TestStatusFail indicates the test failed (expected behavior not met)
	TestStatusFail TestStatus = "fail"

	// TestStatusSkip indicates the test was skipped
	TestStatusSkip TestStatus = "skip"

	// TestStatusError indicates the test encountered an error during execution
	TestStatusError TestStatus = "error"
)

// Category represents whether a test is for SDK or Framework
type Category string

const (
	// CategorySDK represents SDK tests (Requirements 1-16)
	CategorySDK Category = "sdk"

	// CategoryFramework represents Framework tests (Requirements 17-31)
	CategoryFramework Category = "framework"
)

// TestResult represents the outcome of a single test execution
type TestResult struct {
	// TestName is the human-readable name of the test
	TestName string

	// RequirementID identifies which requirement this test validates (e.g., "2.1")
	RequirementID string

	// Category indicates if this is an SDK or Framework test
	Category Category

	// Status is the test outcome (pass, fail, skip, error)
	Status TestStatus

	// Duration is how long the test took to execute
	Duration time.Duration

	// Message provides human-readable information about the result
	Message string

	// Error contains the error if Status is Fail or Error
	Error error

	// Details contains additional context (inputs, outputs, etc.)
	Details map[string]any

	// Timestamp is when the test completed
	Timestamp time.Time
}

// NewPassResult creates a TestResult for a passing test
func NewPassResult(testName, requirementID string, category Category, duration time.Duration, message string) TestResult {
	return TestResult{
		TestName:      testName,
		RequirementID: requirementID,
		Category:      category,
		Status:        TestStatusPass,
		Duration:      duration,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       make(map[string]any),
	}
}

// NewFailResult creates a TestResult for a failing test
func NewFailResult(testName, requirementID string, category Category, duration time.Duration, message string, err error) TestResult {
	return TestResult{
		TestName:      testName,
		RequirementID: requirementID,
		Category:      category,
		Status:        TestStatusFail,
		Duration:      duration,
		Message:       message,
		Error:         err,
		Timestamp:     time.Now(),
		Details:       make(map[string]any),
	}
}

// NewSkipResult creates a TestResult for a skipped test
func NewSkipResult(testName, requirementID string, category Category, message string) TestResult {
	return TestResult{
		TestName:      testName,
		RequirementID: requirementID,
		Category:      category,
		Status:        TestStatusSkip,
		Duration:      0,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       make(map[string]any),
	}
}

// NewErrorResult creates a TestResult for a test that encountered an error
func NewErrorResult(testName, requirementID string, category Category, duration time.Duration, err error) TestResult {
	return TestResult{
		TestName:      testName,
		RequirementID: requirementID,
		Category:      category,
		Status:        TestStatusError,
		Duration:      duration,
		Message:       fmt.Sprintf("Test error: %v", err),
		Error:         err,
		Timestamp:     time.Now(),
		Details:       make(map[string]any),
	}
}

// WithDetails adds details to a TestResult
func (tr TestResult) WithDetails(details map[string]any) TestResult {
	tr.Details = details
	return tr
}

// CategorySummary aggregates results for a test category
type CategorySummary struct {
	// Total is the total number of tests in this category
	Total int

	// Passed is the number of passing tests
	Passed int

	// Failed is the number of failing tests
	Failed int

	// Skipped is the number of skipped tests
	Skipped int

	// Errors is the number of tests with errors
	Errors int
}

// CalculateSummary computes a CategorySummary from test results
func CalculateSummary(results []TestResult) CategorySummary {
	summary := CategorySummary{
		Total: len(results),
	}

	for _, result := range results {
		switch result.Status {
		case TestStatusPass:
			summary.Passed++
		case TestStatusFail:
			summary.Failed++
		case TestStatusSkip:
			summary.Skipped++
		case TestStatusError:
			summary.Errors++
		}
	}

	return summary
}

// PassRate returns the percentage of tests that passed (0.0 - 1.0)
func (cs CategorySummary) PassRate() float64 {
	if cs.Total == 0 {
		return 0
	}
	return float64(cs.Passed) / float64(cs.Total)
}

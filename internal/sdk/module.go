package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"

	"github.com/zero-day-ai/agents/debug/internal/runner"
)

// BaseModule provides common functionality for SDK test modules
type BaseModule struct {
	name          string
	description   string
	requirementID string
}

// NewBaseModule creates a new base module
func NewBaseModule(name, description, requirementID string) BaseModule {
	return BaseModule{
		name:          name,
		description:   description,
		requirementID: requirementID,
	}
}

func (b BaseModule) Name() string {
	return b.name
}

func (b BaseModule) Description() string {
	return b.description
}

func (b BaseModule) Category() runner.Category {
	return runner.CategorySDK
}

func (b BaseModule) RequirementID() string {
	return b.requirementID
}

// TestFunc is a function that executes a single test
type TestFunc func(ctx context.Context, h agent.Harness) runner.TestResult

// RunTest executes a test function with timing and error recovery
func RunTest(testName, requirementID string, testFunc TestFunc) TestFunc {
	return func(ctx context.Context, h agent.Harness) runner.TestResult {
		startTime := time.Now()

		defer func() {
			if r := recover(); r != nil {
				// Panic recovery - return error result
				h.Logger().Error("Test panicked",
					"test", testName,
					"panic", r,
				)
			}
		}()

		result := testFunc(ctx, h)
		result.Duration = time.Since(startTime)
		return result
	}
}

// AssertEqual checks if two values are equal
func AssertEqual(testName, requirementID string, expected, actual any, description string) runner.TestResult {
	if expected != actual {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: expected %v, got %v", description, expected, actual),
			fmt.Errorf("assertion failed: %s", description),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: values match (%v)", description, actual),
	)
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(testName, requirementID string, value any, description string) runner.TestResult {
	if value == nil {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: expected non-nil value, got nil", description),
			fmt.Errorf("assertion failed: %s is nil", description),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: value is not nil", description),
	)
}

// AssertNil checks if a value is nil
func AssertNil(testName, requirementID string, value any, description string) runner.TestResult {
	if value != nil {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: expected nil, got %v", description, value),
			fmt.Errorf("assertion failed: %s is not nil", description),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: value is nil", description),
	)
}

// AssertNoError checks if an error is nil
func AssertNoError(testName, requirementID string, err error, description string) runner.TestResult {
	if err != nil {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: unexpected error: %v", description, err),
			err,
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: no error", description),
	)
}

// AssertError checks if an error is not nil
func AssertError(testName, requirementID string, err error, description string) runner.TestResult {
	if err == nil {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: expected error, got nil", description),
			fmt.Errorf("assertion failed: expected error but got nil"),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: error present: %v", description, err),
	)
}

// AssertTrue checks if a boolean value is true
func AssertTrue(testName, requirementID string, value bool, description string) runner.TestResult {
	if !value {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: expected true, got false", description),
			fmt.Errorf("assertion failed: %s", description),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: condition is true", description),
	)
}

// AssertFalse checks if a boolean value is false
func AssertFalse(testName, requirementID string, value bool, description string) runner.TestResult {
	if value {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: expected false, got true", description),
			fmt.Errorf("assertion failed: %s", description),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: condition is false", description),
	)
}

// AssertGreaterThan checks if a value is greater than another
func AssertGreaterThan(testName, requirementID string, value, threshold int, description string) runner.TestResult {
	if value <= threshold {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: expected > %d, got %d", description, threshold, value),
			fmt.Errorf("assertion failed: %s", description),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: %d > %d", description, value, threshold),
	)
}

// AssertContains checks if a string contains a substring
func AssertContains(testName, requirementID string, haystack, needle, description string) runner.TestResult {
	if !contains(haystack, needle) {
		return runner.NewFailResult(
			testName,
			requirementID,
			runner.CategorySDK,
			0,
			fmt.Sprintf("%s: '%s' does not contain '%s'", description, haystack, needle),
			fmt.Errorf("assertion failed: %s", description),
		)
	}
	return runner.NewPassResult(
		testName,
		requirementID,
		runner.CategorySDK,
		0,
		fmt.Sprintf("%s: string contains substring", description),
	)
}

// contains is a helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CreateTestResult is a helper to create test results with consistent formatting
func CreateTestResult(testName, requirementID string, success bool, message string, err error, duration time.Duration) runner.TestResult {
	if success {
		return runner.NewPassResult(testName, requirementID, runner.CategorySDK, duration, message)
	}
	return runner.NewFailResult(testName, requirementID, runner.CategorySDK, duration, message, err)
}

// SkipTest creates a skip result with a reason
func SkipTest(testName, requirementID string, reason string) runner.TestResult {
	return runner.NewSkipResult(testName, requirementID, runner.CategorySDK, reason)
}

// ErrorTest creates an error result
func ErrorTest(testName, requirementID string, err error, duration time.Duration) runner.TestResult {
	return runner.NewErrorResult(testName, requirementID, runner.CategorySDK, duration, err)
}

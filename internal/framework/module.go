package framework

import (
	"time"

	"github.com/zero-day-ai/agents/debug/internal/runner"
)

// BaseModule provides common functionality for Framework test modules
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
	return runner.CategoryFramework
}

func (b BaseModule) RequirementID() string {
	return b.requirementID
}

// SkipTest creates a skip result for framework tests
func SkipTest(testName, requirementID string, reason string) runner.TestResult {
	return runner.NewSkipResult(testName, requirementID, runner.CategoryFramework, reason)
}

// PassTest creates a pass result for framework tests
func PassTest(testName, requirementID string, message string) runner.TestResult {
	return runner.NewPassResult(testName, requirementID, runner.CategoryFramework, 0, message)
}

// FailTest creates a fail result for framework tests
func FailTest(testName, requirementID string, message string, err error) runner.TestResult {
	return runner.NewFailResult(testName, requirementID, runner.CategoryFramework, 0, message, err)
}

// ErrorTest creates an error result for framework tests
func ErrorTest(testName, requirementID string, err error, duration time.Duration) runner.TestResult {
	return runner.NewErrorResult(testName, requirementID, runner.CategoryFramework, duration, err)
}

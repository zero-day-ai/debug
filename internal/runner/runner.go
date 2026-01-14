package runner

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/zero-day-ai/sdk/agent"
)

// TestModule represents a collection of related tests
// Each module implements one or more requirements
type TestModule interface {
	// Name returns the unique identifier for this module (e.g., "llm-integration")
	Name() string

	// Description returns a human-readable description of what this module tests
	Description() string

	// Category returns whether this is an SDK or Framework test module
	Category() Category

	// RequirementID returns the requirement(s) this module validates (e.g., "2")
	RequirementID() string

	// Run executes all tests in this module and returns results
	// The harness provides access to SDK functionality
	Run(ctx context.Context, h agent.Harness) []TestResult
}

// Runner orchestrates test module execution
type Runner struct {
	modules     []TestModule
	harness     agent.Harness
	logger      *slog.Logger
	timeout     time.Duration
	testTimeout time.Duration
}

// NewRunner creates a new test runner
func NewRunner(harness agent.Harness, timeout, testTimeout time.Duration) *Runner {
	return &Runner{
		modules:     []TestModule{},
		harness:     harness,
		logger:      harness.Logger(),
		timeout:     timeout,
		testTimeout: testTimeout,
	}
}

// RegisterModule adds a test module to the runner
func (r *Runner) RegisterModule(module TestModule) {
	r.modules = append(r.modules, module)
}

// RegisterModules adds multiple test modules to the runner
func (r *Runner) RegisterModules(modules ...TestModule) {
	r.modules = append(r.modules, modules...)
}

// Run executes all registered test modules and returns aggregated results
func (r *Runner) Run(ctx context.Context) (*SuiteResult, error) {
	r.logger.Info("Starting test suite execution",
		"modules", len(r.modules),
		"timeout", r.timeout,
	)

	suite := NewSuiteResult()

	// Create a context with timeout for the entire suite
	suiteCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Execute modules sequentially
	// (Could be parallelized in the future, but sequential is safer for now)
	for _, module := range r.modules {
		// Check if suite context is still valid
		select {
		case <-suiteCtx.Done():
			r.logger.Warn("Suite execution timed out or cancelled",
				"executed_modules", len(suite.Results),
				"total_modules", len(r.modules),
			)
			suite.Finalize()
			return suite, fmt.Errorf("suite execution timed out or cancelled: %w", suiteCtx.Err())
		default:
		}

		// Execute the module
		results := r.runModule(suiteCtx, module)
		suite.AddResults(results)
	}

	suite.Finalize()

	r.logger.Info("Test suite execution completed",
		"duration", suite.Duration(),
		"total", suite.TotalTests(),
		"passed", suite.TotalPassed(),
		"failed", suite.TotalFailed(),
		"skipped", suite.TotalSkipped(),
		"errors", suite.TotalErrors(),
		"status", suite.OverallStatus,
	)

	return suite, nil
}

// RunCategory executes only modules in the specified category
func (r *Runner) RunCategory(ctx context.Context, category Category) (*SuiteResult, error) {
	r.logger.Info("Starting category execution",
		"category", category,
		"timeout", r.timeout,
	)

	suite := NewSuiteResult()

	// Create a context with timeout
	categoryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Execute modules in the specified category
	for _, module := range r.modules {
		if module.Category() != category {
			continue
		}

		// Check if context is still valid
		select {
		case <-categoryCtx.Done():
			r.logger.Warn("Category execution timed out or cancelled",
				"category", category,
			)
			suite.Finalize()
			return suite, fmt.Errorf("category execution timed out: %w", categoryCtx.Err())
		default:
		}

		results := r.runModule(categoryCtx, module)
		suite.AddResults(results)
	}

	suite.Finalize()

	r.logger.Info("Category execution completed",
		"category", category,
		"duration", suite.Duration(),
		"total", suite.TotalTests(),
		"passed", suite.TotalPassed(),
		"failed", suite.TotalFailed(),
	)

	return suite, nil
}

// RunSingle executes a single module by name
func (r *Runner) RunSingle(ctx context.Context, moduleName string) ([]TestResult, error) {
	r.logger.Info("Running single module",
		"module", moduleName,
	)

	// Find the module
	var targetModule TestModule
	for _, module := range r.modules {
		if module.Name() == moduleName {
			targetModule = module
			break
		}
	}

	if targetModule == nil {
		return nil, fmt.Errorf("module not found: %s", moduleName)
	}

	// Create a context with timeout
	moduleCtx, cancel := context.WithTimeout(ctx, r.testTimeout)
	defer cancel()

	// Execute the module
	results := r.runModule(moduleCtx, targetModule)

	r.logger.Info("Single module execution completed",
		"module", moduleName,
		"tests", len(results),
	)

	return results, nil
}

// runModule executes a single test module with panic recovery
func (r *Runner) runModule(ctx context.Context, module TestModule) []TestResult {
	moduleName := module.Name()
	startTime := time.Now()

	r.logger.Debug("Executing module",
		"module", moduleName,
		"category", module.Category(),
		"requirement", module.RequirementID(),
	)

	// Use a separate goroutine with panic recovery
	resultsChan := make(chan []TestResult, 1)
	panicChan := make(chan any, 1)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer func() {
			if panicErr := recover(); panicErr != nil {
				// Capture panic and stack trace
				stackTrace := string(debug.Stack())
				r.logger.Error("Module panicked",
					"module", moduleName,
					"panic", panicErr,
					"stack", stackTrace,
				)
				panicChan <- panicErr
			}
		}()

		// Execute the module
		results := module.Run(ctx, r.harness)
		resultsChan <- results
	}()

	// Wait for completion or timeout
	select {
	case results := <-resultsChan:
		duration := time.Since(startTime)
		r.logger.Debug("Module completed",
			"module", moduleName,
			"tests", len(results),
			"duration", duration,
		)
		return results

	case panicErr := <-panicChan:
		// Module panicked, return error result
		duration := time.Since(startTime)
		return []TestResult{
			NewErrorResult(
				moduleName,
				module.RequirementID(),
				module.Category(),
				duration,
				fmt.Errorf("module panicked: %v", panicErr),
			),
		}

	case <-ctx.Done():
		// Context cancelled or timed out
		duration := time.Since(startTime)
		r.logger.Warn("Module execution timed out or cancelled",
			"module", moduleName,
			"duration", duration,
		)
		return []TestResult{
			NewErrorResult(
				moduleName,
				module.RequirementID(),
				module.Category(),
				duration,
				fmt.Errorf("module timed out: %w", ctx.Err()),
			),
		}
	}
}

// GetModules returns all registered modules
func (r *Runner) GetModules() []TestModule {
	return r.modules
}

// GetModulesByCategory returns modules for a specific category
func (r *Runner) GetModulesByCategory(category Category) []TestModule {
	var filtered []TestModule
	for _, module := range r.modules {
		if module.Category() == category {
			filtered = append(filtered, module)
		}
	}
	return filtered
}

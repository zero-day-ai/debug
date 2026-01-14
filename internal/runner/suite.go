package runner

import (
	"time"
)

// SuiteResult aggregates results from all test modules in a test run
type SuiteResult struct {
	// StartTime is when the test suite began execution
	StartTime time.Time

	// EndTime is when the test suite finished execution
	EndTime time.Time

	// Results contains all test results from all modules
	Results []TestResult

	// SDKSummary aggregates SDK test results
	SDKSummary CategorySummary

	// FrameworkSummary aggregates Framework test results
	FrameworkSummary CategorySummary

	// OverallStatus is the overall suite status
	OverallStatus TestStatus
}

// NewSuiteResult creates a new SuiteResult
func NewSuiteResult() *SuiteResult {
	return &SuiteResult{
		StartTime:        time.Now(),
		Results:          []TestResult{},
		SDKSummary:       CategorySummary{},
		FrameworkSummary: CategorySummary{},
		OverallStatus:    TestStatusPass,
	}
}

// AddResult adds a test result to the suite
func (sr *SuiteResult) AddResult(result TestResult) {
	sr.Results = append(sr.Results, result)
}

// AddResults adds multiple test results to the suite
func (sr *SuiteResult) AddResults(results []TestResult) {
	sr.Results = append(sr.Results, results...)
}

// Finalize computes final statistics and determines overall status
func (sr *SuiteResult) Finalize() {
	sr.EndTime = time.Now()

	// Separate SDK and Framework results
	var sdkResults []TestResult
	var frameworkResults []TestResult

	for _, result := range sr.Results {
		switch result.Category {
		case CategorySDK:
			sdkResults = append(sdkResults, result)
		case CategoryFramework:
			frameworkResults = append(frameworkResults, result)
		}
	}

	// Calculate summaries
	sr.SDKSummary = CalculateSummary(sdkResults)
	sr.FrameworkSummary = CalculateSummary(frameworkResults)

	// Determine overall status
	// If any errors, overall is error
	// Else if any failures, overall is fail
	// Else if all skipped, overall is skip
	// Else overall is pass
	hasErrors := sr.SDKSummary.Errors > 0 || sr.FrameworkSummary.Errors > 0
	hasFailures := sr.SDKSummary.Failed > 0 || sr.FrameworkSummary.Failed > 0
	allSkipped := (sr.SDKSummary.Total == sr.SDKSummary.Skipped) &&
		(sr.FrameworkSummary.Total == sr.FrameworkSummary.Skipped)

	if hasErrors {
		sr.OverallStatus = TestStatusError
	} else if hasFailures {
		sr.OverallStatus = TestStatusFail
	} else if allSkipped && len(sr.Results) > 0 {
		sr.OverallStatus = TestStatusSkip
	} else {
		sr.OverallStatus = TestStatusPass
	}
}

// Duration returns the total execution time
func (sr *SuiteResult) Duration() time.Duration {
	return sr.EndTime.Sub(sr.StartTime)
}

// TotalTests returns the total number of tests executed
func (sr *SuiteResult) TotalTests() int {
	return len(sr.Results)
}

// TotalPassed returns the total number of passing tests
func (sr *SuiteResult) TotalPassed() int {
	return sr.SDKSummary.Passed + sr.FrameworkSummary.Passed
}

// TotalFailed returns the total number of failing tests
func (sr *SuiteResult) TotalFailed() int {
	return sr.SDKSummary.Failed + sr.FrameworkSummary.Failed
}

// TotalSkipped returns the total number of skipped tests
func (sr *SuiteResult) TotalSkipped() int {
	return sr.SDKSummary.Skipped + sr.FrameworkSummary.Skipped
}

// TotalErrors returns the total number of tests with errors
func (sr *SuiteResult) TotalErrors() int {
	return sr.SDKSummary.Errors + sr.FrameworkSummary.Errors
}

// OverallPassRate returns the percentage of tests that passed (0.0 - 1.0)
func (sr *SuiteResult) OverallPassRate() float64 {
	total := sr.TotalTests()
	if total == 0 {
		return 0
	}
	return float64(sr.TotalPassed()) / float64(total)
}

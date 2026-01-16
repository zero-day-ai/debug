package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"
	sdkmem "github.com/zero-day-ai/sdk/memory"
)

// TestResult represents the outcome of a memory test
type TestResult struct {
	Success  bool
	Duration time.Duration
	Message  string
	Error    error
	Details  map[string]any
}

// TierResults aggregates test results for all memory tiers
type TierResults struct {
	Working  *TierTestResult
	Mission  *TierTestResult
	LongTerm *TierTestResult
	Overall  *TestResult
}

// TierTestResult represents test results for a specific memory tier
type TierTestResult struct {
	TierName string
	Set      bool
	Get      bool
	Delete   bool
	Search   bool // Only applicable for Mission and LongTerm
	Error    error
	Details  map[string]any
}

// ExecuteMemoryTest tests all three memory tiers (Working, Mission, LongTerm)
// Tests Set/Get/Delete operations for Working, Set/Get/Search for Mission,
// and Store/Search operations for LongTerm memory.
func ExecuteMemoryTest(ctx context.Context, h agent.Harness) (*TierResults, error) {
	logger := h.Logger()
	logger.Info("Starting comprehensive memory tier tests")

	startTime := time.Now()

	// Get memory store
	mem := h.Memory()
	if mem == nil {
		return &TierResults{
			Overall: &TestResult{
				Success:  false,
				Duration: time.Since(startTime),
				Message:  "Memory store is not available",
				Error:    fmt.Errorf("memory store is nil"),
				Details:  make(map[string]any),
			},
		}, nil
	}

	results := &TierResults{
		Working:  testWorkingMemory(ctx, mem.Working(), logger),
		Mission:  testMissionMemory(ctx, mem.Mission(), logger),
		LongTerm: testLongTermMemory(ctx, mem.LongTerm(), logger),
	}

	// Calculate overall results
	duration := time.Since(startTime)
	allSuccess := results.Working.IsSuccess() &&
		results.Mission.IsSuccess() &&
		results.LongTerm.IsSuccess()

	results.Overall = &TestResult{
		Success:  allSuccess,
		Duration: duration,
		Details:  make(map[string]any),
	}

	if allSuccess {
		results.Overall.Message = "All memory tier tests passed successfully"
		logger.Info("Memory tests completed successfully", "duration", duration)
	} else {
		failures := make([]string, 0)
		if !results.Working.IsSuccess() {
			failures = append(failures, "Working")
		}
		if !results.Mission.IsSuccess() {
			failures = append(failures, "Mission")
		}
		if !results.LongTerm.IsSuccess() {
			failures = append(failures, "LongTerm")
		}
		results.Overall.Message = fmt.Sprintf("Memory tier tests failed: %v", failures)
		results.Overall.Error = fmt.Errorf("failed tiers: %v", failures)
		logger.Error("Memory tests failed", "failed_tiers", failures)
	}

	results.Overall.Details["working"] = results.Working
	results.Overall.Details["mission"] = results.Mission
	results.Overall.Details["long_term"] = results.LongTerm

	return results, nil
}

// testWorkingMemory tests Working memory tier (Set, Get, Delete)
func testWorkingMemory(ctx context.Context, working sdkmem.WorkingMemory, logger interface{ Info(msg string, args ...any) }) *TierTestResult {
	result := &TierTestResult{
		TierName: "Working",
		Details:  make(map[string]any),
	}

	testKey := "[DEBUG]_working_test"
	testValue := map[string]any{
		"timestamp": time.Now().Unix(),
		"test":      "working_memory",
		"data":      []int{1, 2, 3, 4, 5},
	}

	// Test Set
	err := working.Set(ctx, testKey, testValue)
	result.Set = (err == nil)
	if err != nil {
		result.Error = fmt.Errorf("Set failed: %w", err)
		result.Details["set_error"] = err.Error()
		return result
	}

	// Test Get
	retrieved, err := working.Get(ctx, testKey)
	result.Get = (err == nil && retrieved != nil)
	if err != nil {
		result.Error = fmt.Errorf("Get failed: %w", err)
		result.Details["get_error"] = err.Error()
		return result
	}

	// Verify retrieved value matches
	if retrievedMap, ok := retrieved.(map[string]any); ok {
		result.Details["retrieved"] = retrievedMap
		if testVal, exists := retrievedMap["test"]; exists && testVal == "working_memory" {
			result.Details["value_match"] = true
		} else {
			result.Details["value_match"] = false
			result.Error = fmt.Errorf("retrieved value does not match")
		}
	}

	// Test Delete
	err = working.Delete(ctx, testKey)
	result.Delete = (err == nil)
	if err != nil {
		result.Error = fmt.Errorf("Delete failed: %w", err)
		result.Details["delete_error"] = err.Error()
		return result
	}

	// Verify deletion
	_, err = working.Get(ctx, testKey)
	if err == sdkmem.ErrNotFound {
		result.Details["delete_verified"] = true
	} else {
		result.Details["delete_verified"] = false
		result.Error = fmt.Errorf("item still exists after delete")
	}

	logger.Info("Working memory test completed",
		"set", result.Set,
		"get", result.Get,
		"delete", result.Delete,
	)

	return result
}

// testMissionMemory tests Mission memory tier (Set, Get, Delete, Search)
func testMissionMemory(ctx context.Context, mission sdkmem.MissionMemory, logger interface{ Info(msg string, args ...any) }) *TierTestResult {
	result := &TierTestResult{
		TierName: "Mission",
		Details:  make(map[string]any),
	}

	testKey := "[DEBUG]_mission_test"
	testValue := map[string]any{
		"timestamp": time.Now().Unix(),
		"test":      "mission_memory",
		"content":   "This is a test for mission memory tier",
	}
	testMetadata := map[string]any{
		"category": "debug",
		"test":     true,
	}

	// Test Set
	err := mission.Set(ctx, testKey, testValue, testMetadata)
	result.Set = (err == nil)
	if err != nil {
		result.Error = fmt.Errorf("Set failed: %w", err)
		result.Details["set_error"] = err.Error()
		return result
	}

	// Test Get
	item, err := mission.Get(ctx, testKey)
	result.Get = (err == nil && item != nil)
	if err != nil {
		result.Error = fmt.Errorf("Get failed: %w", err)
		result.Details["get_error"] = err.Error()
	} else if item != nil {
		result.Details["item_created_at"] = item.CreatedAt
		result.Details["item_updated_at"] = item.UpdatedAt
	}

	// Test Search
	searchResults, err := mission.Search(ctx, "mission_memory", 10)
	result.Search = (err == nil)
	if err != nil {
		result.Error = fmt.Errorf("Search failed: %w", err)
		result.Details["search_error"] = err.Error()
	} else {
		result.Details["search_result_count"] = len(searchResults)
		// Check if our test item is in search results
		found := false
		for _, sr := range searchResults {
			if sr.Key == testKey {
				found = true
				break
			}
		}
		result.Details["search_found_item"] = found
	}

	// Test Delete (cleanup)
	err = mission.Delete(ctx, testKey)
	result.Delete = (err == nil)
	if err != nil {
		result.Details["delete_error"] = err.Error()
		// Don't set result.Error here as we want to complete the test
	}

	logger.Info("Mission memory test completed",
		"set", result.Set,
		"get", result.Get,
		"search", result.Search,
		"delete", result.Delete,
	)

	return result
}

// testLongTermMemory tests LongTerm memory tier (Store, Search, Delete)
func testLongTermMemory(ctx context.Context, longTerm sdkmem.LongTermMemory, logger interface{ Info(msg string, args ...any) }) *TierTestResult {
	result := &TierTestResult{
		TierName: "LongTerm",
		Details:  make(map[string]any),
	}

	testContent := "[DEBUG] Long-term memory test: Gibson framework uses three memory tiers for persistent agent knowledge"
	testMetadata := map[string]any{
		"category": "debug",
		"test":     true,
		"tier":     "long_term",
		"debug":    true,
	}

	// Test Store
	id, err := longTerm.Store(ctx, testContent, testMetadata)
	result.Set = (err == nil && id != "")
	if err != nil {
		result.Error = fmt.Errorf("Store failed: %w", err)
		result.Details["store_error"] = err.Error()
		return result
	}
	result.Details["stored_id"] = id

	// Test Search
	searchResults, err := longTerm.Search(ctx, "memory tiers Gibson framework", 5, map[string]any{"debug": true})
	result.Search = (err == nil)
	if err != nil {
		result.Error = fmt.Errorf("Search failed: %w", err)
		result.Details["search_error"] = err.Error()
	} else {
		result.Details["search_result_count"] = len(searchResults)
		// Check if our test item is in search results
		found := false
		for _, sr := range searchResults {
			if sr.Key == id {
				found = true
				result.Details["search_score"] = sr.Score
				break
			}
		}
		result.Details["search_found_item"] = found
	}

	// Test Get (retrieve what we stored)
	result.Get = result.Search // In LongTerm, Search serves as Get

	// Test Delete (cleanup)
	err = longTerm.Delete(ctx, id)
	result.Delete = (err == nil)
	if err != nil {
		result.Details["delete_error"] = err.Error()
		// Don't set result.Error here as we want to complete the test
	} else {
		result.Details["deleted_id"] = id
	}

	logger.Info("LongTerm memory test completed",
		"store", result.Set,
		"search", result.Search,
		"delete", result.Delete,
		"stored_id", id,
	)

	return result
}

// IsSuccess returns true if all operations in the tier test succeeded
func (tr *TierTestResult) IsSuccess() bool {
	// All tiers: Set, Get, Delete must all succeed (and Search for Mission/LongTerm)
	if tr.TierName == "LongTerm" {
		return tr.Set && tr.Search && tr.Delete
	}
	if tr.TierName == "Mission" {
		return tr.Set && tr.Get && tr.Delete && tr.Search
	}
	return tr.Set && tr.Get && tr.Delete
}

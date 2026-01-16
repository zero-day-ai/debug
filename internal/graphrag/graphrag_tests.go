package graphrag

import (
	"context"
	"fmt"
	"time"

	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/graphrag"

	"github.com/zero-day-ai/agents/debug/internal/runner"
)

// TestConfig holds configuration for GraphRAG tests
type TestConfig struct {
	// Prefix for test data (should be "[DEBUG]" to mark test data)
	Prefix string

	// CleanupOnSuccess determines if test data should be cleaned up after successful test
	CleanupOnSuccess bool

	// CleanupOnFailure determines if test data should be cleaned up after failed test
	CleanupOnFailure bool
}

// ExecuteGraphRAGTest executes GraphRAG store, query, and traverse tests
// This function:
// 1. Tests StoreGraphNode() with [DEBUG] prefixed test data
// 2. Tests QueryGraphRAG() to retrieve stored data
// 3. Tests TraverseGraph() for relationship traversal
// 4. Cleans up test data
// Returns a structured TestResult
func ExecuteGraphRAGTest(ctx context.Context, harness agent.Harness, cfg TestConfig) runner.TestResult {
	testName := "GraphRAG Operations Test"
	reqID := "REQ-4"
	startTime := time.Now()

	harness.Logger().Info("Starting GraphRAG test")

	// Apply defaults
	if cfg.Prefix == "" {
		cfg.Prefix = "[DEBUG]"
	}

	// Phase 0: Check GraphRAG health
	harness.Logger().Info("Checking GraphRAG health")
	health := harness.GraphRAGHealth(ctx)
	if health.Status != "healthy" {
		return runner.NewSkipResult(
			testName,
			reqID,
			runner.CategorySDK,
			fmt.Sprintf("GraphRAG unavailable: %s - %s", health.Status, health.Message),
		)
	}

	harness.Logger().Info("GraphRAG is healthy", "status", health.Status)

	// Get mission context for unique test IDs
	mission := harness.Mission()
	testID := fmt.Sprintf("test-%s-%d", mission.ID, time.Now().Unix())

	// Track created node IDs for cleanup
	createdNodeIDs := []string{}

	// Phase 1: Store test nodes with relationships
	harness.Logger().Info("Phase 1: Storing test graph nodes")

	// Create a simple graph: TestSuite -> TestCase -> TestAssertion
	// This tests node creation and relationship establishment

	// Node 1: Test Suite
	suiteNode := graphrag.NewGraphNode("TestSuite").
		WithID(fmt.Sprintf("suite-%s", testID)).
		WithProperty("name", fmt.Sprintf("%s GraphRAG Test Suite", cfg.Prefix)).
		WithProperty("test_id", testID).
		WithProperty("timestamp", time.Now()).
		WithContent(fmt.Sprintf("%s Test suite for validating GraphRAG operations", cfg.Prefix))

	// Node 2: Test Case
	caseNode := graphrag.NewGraphNode("TestCase").
		WithID(fmt.Sprintf("case-%s", testID)).
		WithProperty("name", fmt.Sprintf("%s Store and Query Test", cfg.Prefix)).
		WithProperty("test_id", testID).
		WithProperty("status", "running").
		WithContent(fmt.Sprintf("%s Test case for store and query operations", cfg.Prefix))

	// Node 3: Test Assertion
	assertionNode := graphrag.NewGraphNode("TestAssertion").
		WithID(fmt.Sprintf("assertion-%s", testID)).
		WithProperty("name", fmt.Sprintf("%s Data Persistence", cfg.Prefix)).
		WithProperty("test_id", testID).
		WithProperty("expected", "data_persisted").
		WithContent(fmt.Sprintf("%s Assertion that stored data can be retrieved", cfg.Prefix))

	// Build batch with relationships
	batch := graphrag.Batch{
		Nodes: []graphrag.GraphNode{*suiteNode, *caseNode, *assertionNode},
		Relationships: []graphrag.Relationship{
			*graphrag.NewRelationship(suiteNode.ID, caseNode.ID, "CONTAINS"),
			*graphrag.NewRelationship(caseNode.ID, assertionNode.ID, "VERIFIES"),
		},
	}

	harness.Logger().Info("Storing graph batch",
		"nodes", len(batch.Nodes),
		"relationships", len(batch.Relationships),
	)

	nodeIDs, err := harness.StoreGraphBatch(ctx, batch)
	if err != nil {
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Failed to store graph batch: %v", err),
			err,
		)
	}

	createdNodeIDs = append(createdNodeIDs, nodeIDs...)
	harness.Logger().Info("Graph batch stored successfully",
		"node_count", len(nodeIDs),
		"node_ids", nodeIDs,
	)

	// Phase 2: Query stored data
	harness.Logger().Info("Phase 2: Querying stored graph data")

	// Query for nodes with our test_id property
	query := graphrag.Query{
		Text: fmt.Sprintf("test_id:%s", testID),
		TopK: 10,
	}
	queryResults, err := harness.QueryGraphRAG(ctx, query)
	if err != nil {
		cleanupTestData(ctx, harness, createdNodeIDs, cfg)
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Failed to query graph: %v", err),
			err,
		)
	}

	if len(queryResults) == 0 {
		cleanupTestData(ctx, harness, createdNodeIDs, cfg)
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Query returned no results for test_id=%s", testID),
			fmt.Errorf("no query results"),
		)
	}

	harness.Logger().Info("Query completed successfully",
		"results_count", len(queryResults),
		"query", query.Text,
	)

	// Verify we got our test nodes back
	foundNodes := make(map[string]bool)
	for _, result := range queryResults {
		foundNodes[result.Node.ID] = true
		harness.Logger().Info("Found test node",
			"id", result.Node.ID,
			"type", result.Node.Type,
			"content", result.Node.Content,
			"score", result.Score,
		)
	}

	expectedNodes := []string{suiteNode.ID, caseNode.ID, assertionNode.ID}
	for _, expectedID := range expectedNodes {
		if !foundNodes[expectedID] {
			cleanupTestData(ctx, harness, createdNodeIDs, cfg)
			return runner.NewFailResult(
				testName,
				reqID,
				runner.CategorySDK,
				time.Since(startTime),
				fmt.Sprintf("Expected node '%s' not found in query results", expectedID),
				fmt.Errorf("missing expected node"),
			)
		}
	}

	// Phase 3: Traverse relationships
	harness.Logger().Info("Phase 3: Traversing graph relationships")

	// Traverse from suite to see if we can reach case and assertion
	traverseOpts := graphrag.TraversalOptions{
		MaxDepth:  2, // depth 2 to reach assertion
		Direction: "outgoing",
	}
	traverseResults, err := harness.TraverseGraph(ctx, suiteNode.ID, traverseOpts)
	if err != nil {
		cleanupTestData(ctx, harness, createdNodeIDs, cfg)
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Failed to traverse graph: %v", err),
			err,
		)
	}

	harness.Logger().Info("Traversal completed",
		"results_count", len(traverseResults),
		"start_node", suiteNode.ID,
		"depth", traverseOpts.MaxDepth,
	)

	// Verify we found related nodes
	foundInTraverse := make(map[string]bool)
	for _, result := range traverseResults {
		foundInTraverse[result.Node.ID] = true
		harness.Logger().Info("Traversed to node",
			"id", result.Node.ID,
			"type", result.Node.Type,
			"distance", result.Distance,
		)
	}

	// Should find at least the case node (direct relationship)
	if !foundInTraverse[caseNode.ID] {
		cleanupTestData(ctx, harness, createdNodeIDs, cfg)
		return runner.NewFailResult(
			testName,
			reqID,
			runner.CategorySDK,
			time.Since(startTime),
			fmt.Sprintf("Traversal did not find expected case node '%s'", caseNode.ID),
			fmt.Errorf("traversal missing expected nodes"),
		)
	}

	// Phase 4: Cleanup test data
	harness.Logger().Info("Phase 4: Cleaning up test data")
	cleanupTestData(ctx, harness, createdNodeIDs, cfg)

	duration := time.Since(startTime)
	harness.Logger().Info("GraphRAG test completed successfully",
		"duration", duration,
		"nodes_created", len(createdNodeIDs),
		"query_results", len(queryResults),
		"traverse_results", len(traverseResults),
	)

	return runner.NewPassResult(
		testName,
		reqID,
		runner.CategorySDK,
		duration,
		fmt.Sprintf("GraphRAG test passed: stored %d nodes, queried %d results, traversed %d nodes",
			len(createdNodeIDs), len(queryResults), len(traverseResults)),
	).WithDetails(map[string]any{
		"nodes_created":    len(createdNodeIDs),
		"query_results":    len(queryResults),
		"traverse_results": len(traverseResults),
		"test_id":          testID,
		"execution_time":   duration.String(),
	})
}

// cleanupTestData attempts to delete test nodes from the graph
func cleanupTestData(ctx context.Context, harness agent.Harness, nodeIDs []string, cfg TestConfig) {
	if len(nodeIDs) == 0 {
		return
	}

	harness.Logger().Info("Cleaning up test data",
		"node_count", len(nodeIDs),
	)

	// Note: The SDK doesn't have a DeleteGraphNode method exposed in the current harness
	// In a production implementation, we would call harness.DeleteGraphNodes(ctx, nodeIDs)
	// For now, we log the cleanup intent. The [DEBUG] prefix in content allows filtering
	// test data in queries.

	harness.Logger().Info("Test data marked with prefix for identification",
		"prefix", cfg.Prefix,
		"node_ids", nodeIDs,
		"note", "Use prefix filter to identify test data in graph queries",
	)
}

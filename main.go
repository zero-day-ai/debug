package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	sdk "github.com/zero-day-ai/sdk"
	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/llm"
	"github.com/zero-day-ai/sdk/serve"
	"github.com/zero-day-ai/sdk/types"
)

const (
	agentName    = "debug-agent"
	agentVersion = "1.0.0"
)

func main() {
	fmt.Printf("Gibson Debug Agent v%s\n\n", agentVersion)

	// Create the debug agent using SDK builder pattern
	debugAgent, err := sdk.NewAgent(
		// Basic metadata
		sdk.WithName(agentName),
		sdk.WithVersion(agentVersion),
		sdk.WithDescription("Comprehensive diagnostic and testing agent for Gibson SDK and Framework. "+
			"Systematically validates all SDK and Framework functionality including agents, tools, plugins, "+
			"memory systems, GraphRAG, findings, LLM integration, mission orchestration, workflow engine, "+
			"component registry, database layer, and observability stack."),

		// Target types - supports all types for testing
		sdk.WithTargetTypes(
			types.TargetTypeLLMChat,
			types.TargetTypeLLMAPI,
			types.TargetTypeRAG,
			types.TargetTypeAgent,
			types.TargetTypeCopilot,
		),

		// Technique types
		sdk.WithTechniqueTypes(
			types.TechniquePromptInjection,
			types.TechniqueJailbreak,
			types.TechniqueDataExtraction,
			types.TechniqueModelManipulation,
			types.TechniqueDOS,
		),

		// Agent capabilities
		sdk.WithCapabilities(
			agent.CapabilityPromptInjection,
			agent.CapabilityJailbreak,
			agent.CapabilityDataExtraction,
			agent.CapabilityModelManipulation,
			agent.CapabilityDOS,
		),

		// LLM Slot - minimal requirements for debug agent
		sdk.WithLLMSlot("primary", llm.SlotRequirements{
			MinContextWindow: 8000,
			RequiredFeatures: []string{},
			PreferredModels:  []string{"claude-3-haiku", "gpt-4o-mini"},
		}),

		// Execution function
		sdk.WithExecuteFunc(executeDebugAgent),
	)
	if err != nil {
		log.Fatalf("Failed to create debug agent: %v", err)
	}

	// Parse command line flags
	// Gibson CLI passes --port flag when starting agents
	portFlag := flag.Int("port", 0, "Port to listen on (passed by Gibson CLI)")
	flag.Parse()

	// Determine port: CLI flag > environment variable > default
	port := 50051
	if *portFlag > 0 {
		port = *portFlag
	} else if portEnv := os.Getenv("AGENT_PORT"); portEnv != "" {
		fmt.Sscanf(portEnv, "%d", &port)
	}

	// Build serve options
	opts := []serve.Option{
		serve.WithPort(port),
		serve.WithRegistryFromEnv(), // Auto-register with etcd if GIBSON_REGISTRY_ENDPOINTS is set
	}

	fmt.Printf("Starting debug-agent v%s on port %d...\n", agentVersion, port)

	// Serve the agent as a gRPC service
	if err := serve.Agent(debugAgent, opts...); err != nil {
		log.Fatalf("Failed to serve agent: %v", err)
	}
}

package main

import (
	"fmt"
	"log"

	"github.com/zero-day-ai/sdk/agent"
	"github.com/zero-day-ai/sdk/llm"
	"github.com/zero-day-ai/sdk/types"
)

const (
	agentName    = "debug-agent"
	agentVersion = "1.0.0"
)

func main() {
	// Create the debug agent using agent package directly (bypassing SDK wrapper due to serve package issues)
	cfg := agent.NewConfig().
		SetName(agentName).
		SetVersion(agentVersion).
		SetDescription("Comprehensive diagnostic and testing agent for Gibson SDK and Framework. "+
			"Systematically validates all SDK and Framework functionality including agents, tools, plugins, "+
			"memory systems, GraphRAG, findings, LLM integration, mission orchestration, workflow engine, "+
			"component registry, database layer, and observability stack.").
		AddCapability(agent.CapabilityPromptInjection).
		AddCapability(agent.CapabilityJailbreak).
		AddCapability(agent.CapabilityDataExtraction).
		AddCapability(agent.CapabilityModelManipulation).
		AddCapability(agent.CapabilityDOS).
		AddTargetType(types.TargetTypeLLMChat).
		AddTargetType(types.TargetTypeLLMAPI).
		AddTargetType(types.TargetTypeRAG).
		AddTargetType(types.TargetTypeAgent).
		AddTargetType(types.TargetTypeCopilot).
		AddLLMSlot("primary", llm.SlotRequirements{
			MinContextWindow: 8000,
			RequiredFeatures: []string{},
			PreferredModels:  []string{"claude-3-haiku", "gpt-4o-mini"},
		}).
		SetExecuteFunc(executeDebugAgent)

	debugAgent, err := agent.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create debug agent: %v", err)
	}

	fmt.Printf("Debug Agent initialized successfully!\n")
	fmt.Printf("  Name: %s\n", debugAgent.Name())
	fmt.Printf("  Version: %s\n", debugAgent.Version())
	fmt.Printf("  Description: %s\n", debugAgent.Description())
	fmt.Println("\nAgent is ready to run via Gibson framework:")
	fmt.Println("  gibson attack --agent debug-agent")
	fmt.Println("\nOr use as a component:")
	fmt.Println("  gibson agent install /path/to/debug-agent")

	// TODO: Add gRPC serving support once SDK serve package is updated
	// Check if we should serve as gRPC service
	// if len(os.Args) > 1 && os.Args[1] == "serve" {
	// 	fmt.Println("\nStarting agent gRPC server...")
	// 	if err := serve.ServeAgent(debugAgent); err != nil {
	// 		log.Fatalf("Failed to serve agent: %v", err)
	// 	}
	// }
}

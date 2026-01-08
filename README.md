# Gibson Debug Agent

A comprehensive diagnostic and testing agent for the Gibson SDK and Framework. This agent systematically validates SDK and Framework functionality to help developers verify their Gibson installation is working correctly.

## Overview

The Debug Agent tests:
- **SDK Components** (Requirements 1-16): Agents, tools, plugins, memory systems, GraphRAG, findings, LLM integration, schema validation, and gRPC infrastructure
- **Framework Components** (Requirements 17-31): Daemon service, mission orchestration, workflow engine, component registry, database layer, CLI operations, and observability stack

## Features

- ✅ Comprehensive SDK testing (agent lifecycle, LLM, tools, plugins, memory, findings)
- ✅ Framework integration testing (daemon, missions, workflows)
- ✅ Multiple execution modes (full, sdk-only, framework-only, single-test)
- ✅ Configurable timeouts and test filtering
- ✅ Text and JSON output formats
- ✅ Detailed test results with pass/fail/skip/error status
- ✅ Structured logging and error reporting

## Installation

### As a Gibson Component

```bash
# From the debug agent directory
gibson agent install .
```

### Building from Source

```bash
go build -o debug-agent .
```

## Usage

### Via Gibson Framework

```bash
# Run full test suite
gibson attack --agent debug-agent

# Run SDK tests only
gibson attack --agent debug-agent --context '{"mode": "sdk"}'

# Run Framework tests only
gibson attack --agent debug-agent --context '{"mode": "framework"}'

# Run with verbose output
gibson attack --agent debug-agent --context '{"verbose": true}'
```

### Direct Execution

```bash
# The agent can also be run directly
./debug-agent
```

## Configuration

Configure execution via task context metadata:

```json
{
  "mode": "full|sdk|framework|single",
  "verbose": true,
  "timeout": "10m",
  "category_timeout": "60s",
  "test_timeout": "10s",
  "output_format": "text|json|both",
  "skip_categories": ["framework"],
  "skip_tests": ["test-name"],
  "tests": ["test1", "test2"]
}
```

### Configuration Options

- **mode**: Execution mode
  - `full` - Run all SDK and Framework tests (default)
  - `sdk` - Run only SDK tests
  - `framework` - Run only Framework tests
  - `single` - Run specific tests
- **verbose**: Enable detailed output (default: false)
- **timeout**: Overall execution timeout (default: 10m)
- **output_format**: Report format
  - `text` - Human-readable output
  - `json` - Structured JSON output
  - `both` - Both formats (default)
- **skip_categories**: Categories to skip testing
- **skip_tests**: Individual tests to skip
- **tests**: Specific tests to run (single mode only)

## Architecture

```
debug-agent/
├── main.go              # Agent entry point
├── config.go            # Configuration handling
├── execute.go           # Execution orchestrator
├── component.yaml       # Gibson manifest
├── internal/
│   ├── runner/         # Test orchestration framework
│   │   ├── runner.go   # Test runner
│   │   ├── result.go   # Result types
│   │   └── suite.go    # Suite aggregation
│   ├── sdk/            # SDK test modules
│   │   ├── module.go   # Base module and helpers
│   │   └── comprehensive_tests.go  # SDK tests
│   └── framework/      # Framework test modules
│       ├── module.go   # Base module
│       └── comprehensive_tests.go  # Framework tests
└── testdata/           # Test fixtures
```

## Test Modules

### SDK Tests (Requirements 1-16)

- Agent lifecycle and metadata
- LLM integration (Complete, Stream, CompleteWithTools)
- Tool system (discovery and execution)
- Plugin system (discovery and queries)
- Agent delegation
- Three-tier memory (working, mission, long-term)
- Finding submission and retrieval
- GraphRAG operations
- Target system access
- Mission context access
- Planning integration
- Observability (logging, tracing)
- Streaming execution
- Schema validation
- Error handling
- gRPC services

### Framework Tests (Requirements 17-31)

- Daemon gRPC service
- Mission orchestration
- Workflow engine
- Component registry (etcd)
- Database layer (SQLite)
- Component lifecycle
- CLI commands
- TUI integration
- LLM provider registry
- Framework harness
- Prompt system
- Observability stack
- Configuration system
- Neo4j integration
- Finding deduplication

## Output

### Text Format

```
=== Debug Agent Test Report ===
Mode: full
Duration: 2.5s
Status: pass

=== Overall Summary ===
Total Tests: 45
Passed: 42 (93.3%)
Failed: 1
Skipped: 2
Errors: 0

=== SDK Tests ===
Total: 25
Passed: 24
Failed: 1
...
```

### JSON Format

```json
{
  "mode": "full",
  "duration": "2.5s",
  "overall_status": "pass",
  "summary": {
    "total": 45,
    "passed": 42,
    "failed": 1,
    "skipped": 2
  },
  "results": [...]
}
```

## Development

### Building

```bash
go build -o debug-agent .
```

### Testing

```bash
go test ./...
```

### Adding New Tests

1. Create test methods in the appropriate module (`sdk/` or `framework/`)
2. Add test to the Run() method
3. Use assertion helpers from `module.go`
4. Return TestResult with pass/fail/skip/error status

## Implementation Status

### Completed (Phase 1-2)
- ✅ Project structure and configuration
- ✅ Test runner infrastructure
- ✅ Execution orchestrator
- ✅ SDK test base and comprehensive tests
- ✅ Framework test base and comprehensive tests
- ✅ Report generation (text and JSON)
- ✅ Module registration and integration

### Future Enhancements
- Additional test coverage for individual requirements
- Report generation with findings submission
- Integration with CI/CD systems
- Performance benchmarking
- Test fixtures for workflow engine tests

## Requirements Coverage

The debug agent validates **32 requirements** across SDK and Framework:
- Requirements 1-16: SDK functionality
- Requirements 17-31: Framework functionality
- Requirement 32: Comprehensive diagnostic reporting

## License

MIT

## Contributing

The debug agent is part of the Gibson project. Contributions are welcome!

## Support

For issues or questions, consult the Gibson documentation or raise an issue in the Gibson repository.

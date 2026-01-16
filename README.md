# Gibson Debug Agent

A comprehensive diagnostic and testing agent for the Gibson SDK and Framework. This agent systematically validates SDK and Framework functionality to help developers verify their Gibson installation is working correctly.

> **Note:** As of the latest version, execution modes have been removed. The debug agent now always runs the full test suite (SDK + Framework tests). Mode-specific execution and network reconnaissance features have been simplified or removed.

## Overview

The Debug Agent tests:
- **SDK Components** (Requirements 1-16): Agents, tools, plugins, memory systems, GraphRAG, findings, LLM integration, schema validation, and gRPC infrastructure
- **Framework Components** (Requirements 17-31): Daemon service, mission orchestration, workflow engine, component registry, database layer, CLI operations, and observability stack

## Features

- ✅ Comprehensive SDK testing (agent lifecycle, LLM, tools, plugins, memory, findings)
- ✅ Framework integration testing (daemon, missions, workflows)
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
# Run full test suite (SDK + Framework)
gibson attack --agent debug-agent

# Run with verbose output
gibson attack --agent debug-agent --context '{"verbose": true}'
```

### Direct Execution

```bash
# The agent can also be run directly
./debug-agent
```

## Network Reconnaissance Mode

The debug agent includes a comprehensive network reconnaissance capability that performs automated security testing of local networks. This mode orchestrates security tools (nmap, masscan, httpx, nuclei, subfinder, amass) and generates LLM-powered intelligence analysis.

### Overview

Network reconnaissance mode enables:
- Auto-discovery of local subnet from network interfaces
- Host discovery and port scanning (nmap, masscan)
- HTTP endpoint probing and technology detection (httpx)
- Vulnerability scanning (nuclei)
- Domain enumeration from /etc/hosts (subfinder, amass)
- LLM-powered intelligence generation with risk assessment and attack paths
- Knowledge graph population with hosts, ports, endpoints, findings, and intelligence nodes

### Available Modes

The network reconnaissance feature provides six execution modes:

| Mode | Description | Tools Used | Output |
|------|-------------|------------|--------|
| `network-recon` | Full reconnaissance workflow | All tools | Complete mission with intelligence |
| `network-recon-discover` | Network discovery and host scanning | nmap, masscan | Host and port nodes |
| `network-recon-probe` | HTTP endpoint probing | httpx | Endpoint and technology nodes |
| `network-recon-scan` | Vulnerability scanning | nuclei | Finding nodes |
| `network-recon-domain` | Domain enumeration | subfinder, amass | Domain and subdomain nodes |
| `network-recon-analyze` | Intelligence generation | LLM (Claude) | Intelligence nodes with analysis |

### Running Network Reconnaissance

#### Option 1: Full Mission Workflow (Recommended)

Run the complete reconnaissance mission using the dev environment:

```bash
# 1. Start the dev environment (Neo4j, Qdrant, etc.)
cd /home/anthony/Code/zero-day.ai
./dev/start-dev.sh

# 2. Run the reconnaissance mission
gibson mission run dev/debug-recon-mission.yaml

# 3. View results in Neo4j browser
# Open http://localhost:7474 (connect to bolt://localhost:7687)
# Run Cypher queries to explore the knowledge graph:
#   MATCH (h:host) RETURN h LIMIT 25
#   MATCH (i:intelligence)-[r:ANALYZES]->(n) RETURN i, r, n LIMIT 50
#   MATCH path = (h:host)-[*]-(f:finding) RETURN path LIMIT 25
```

#### Option 2: Individual Phases

Run specific reconnaissance phases independently:

```bash
# Discovery only
gibson attack --agent debug \
  --context '{"mode": "network-recon-discover"}'

# HTTP probing only (requires discovered hosts)
gibson attack --agent debug \
  --context '{"mode": "network-recon-probe"}'

# Vulnerability scanning (requires probed endpoints)
gibson attack --agent debug \
  --context '{"mode": "network-recon-scan"}'

# Domain enumeration
gibson attack --agent debug \
  --context '{"mode": "network-recon-domain"}'

# Intelligence generation (requires discovered data)
gibson attack --agent debug \
  --context '{"mode": "network-recon-analyze", "generate_intelligence": true}'
```

#### Option 3: Direct Execution

Run the agent directly with custom configuration:

```bash
cd opensource/agents/debug
./bin/debug \
  --mode network-recon \
  --neo4j-uri bolt://localhost:7687 \
  --qdrant-uri http://localhost:6334
```

### Configuration Options

Network reconnaissance modes support additional configuration via task context:

```json
{
  "mode": "network-recon",
  "demo_mode": true,
  "verbosity": "high",
  "generate_intelligence": true,
  "intelligence_model": "claude-opus-4-5-20251101",
  "max_hosts": 50,
  "scan_timeout": "10m"
}
```

**Configuration Parameters:**
- `mode`: Execution mode (see table above)
- `demo_mode`: Enable detailed logging and demo output (default: false)
- `verbosity`: Logging level - "low", "medium", "high" (default: "medium")
- `generate_intelligence`: Enable LLM intelligence generation (default: true)
- `intelligence_model`: LLM model for analysis (default: claude-opus-4-5-20251101)
- `max_hosts`: Maximum hosts to scan (default: 256)
- `scan_timeout`: Timeout for scanning phases (default: 5m)

### Setting Up /etc/hosts for Domain Reconnaissance

To enable domain enumeration, add local domain mappings to `/etc/hosts`:

```bash
# Edit /etc/hosts (requires sudo)
sudo nano /etc/hosts

# Add local domain mappings:
192.168.1.1     router.local gateway.local
192.168.1.10    nas.local fileserver.local storage.home.local
192.168.1.20    server.local web.home.local api.home.local
192.168.1.100   dev.local test.local staging.home.local
```

**Domain Mapping Format:**
```
<IP_ADDRESS>    <hostname1> <hostname2> <hostname3>
```

**Best Practices:**
- Use `.local` suffix for local network domains
- Group related services on the same IP
- Use descriptive hostnames (nas, router, server, etc.)
- Avoid conflicts with public domain names

The domain enumeration phase will:
1. Parse `/etc/hosts` for hostname-to-IP mappings
2. Extract unique domain suffixes (e.g., `home.local`, `local`)
3. Run subfinder and amass to discover subdomains
4. Create domain and subdomain nodes in the knowledge graph
5. Link domains to their IP addresses via relationships

### Knowledge Graph Output

Network reconnaissance populates the Gibson knowledge graph with the following node types and relationships:

**Node Types Created:**

| Node Type | Properties | Description |
|-----------|------------|-------------|
| `host` | ip, hostname, os, status | Discovered network hosts |
| `port` | number, protocol, service, version, state | Open ports on hosts |
| `endpoint` | url, status_code, title, content_type | HTTP/HTTPS endpoints |
| `technology` | name, version, category | Detected web technologies |
| `finding` | title, severity, description, cvss_score | Vulnerabilities and issues |
| `domain` | name, type, source | Root domains |
| `subdomain` | name, type, source | Subdomains |
| `intelligence` | summary, risk_assessment, attack_paths, recommendations, confidence | LLM-generated analysis |

**Relationships Created:**

| Relationship | From → To | Description |
|--------------|-----------|-------------|
| `HAS_PORT` | host → port | Host has open port |
| `HAS_ENDPOINT` | port → endpoint | Port serves HTTP endpoint |
| `USES_TECHNOLOGY` | endpoint → technology | Endpoint uses technology |
| `HAS_FINDING` | endpoint → finding | Endpoint has vulnerability |
| `HAS_SUBDOMAIN` | domain → subdomain | Domain has subdomain |
| `RESOLVES_TO` | domain/subdomain → host | DNS resolution |
| `ANALYZES` | intelligence → * | Intelligence analyzes entity |
| `GENERATED_BY` | intelligence → llm_call | Intelligence created by LLM call |
| `PART_OF_MISSION` | intelligence → mission | Intelligence belongs to mission |

**Example Cypher Queries:**

```cypher
// View all discovered hosts with their ports
MATCH (h:host)-[r:HAS_PORT]->(p:port)
RETURN h.ip, h.hostname, collect(p.number) as open_ports
ORDER BY h.ip

// Find high-severity vulnerabilities
MATCH (h:host)-[:HAS_PORT]->(p:port)-[:HAS_ENDPOINT]->(e:endpoint)-[:HAS_FINDING]->(f:finding)
WHERE f.severity IN ['high', 'critical']
RETURN h.ip, e.url, f.title, f.severity, f.cvss_score
ORDER BY f.cvss_score DESC

// View intelligence analysis with linked entities
MATCH (i:intelligence)-[r:ANALYZES]->(n)
RETURN i.summary, i.risk_assessment, type(r), labels(n), n
LIMIT 50

// Trace attack paths from intelligence nodes
MATCH path = (i:intelligence)-[:ANALYZES]->(h:host)-[*]-(f:finding)
WHERE f.severity IN ['high', 'critical']
RETURN path
LIMIT 25

// View domain enumeration results
MATCH (d:domain)-[:HAS_SUBDOMAIN]->(s:subdomain)-[:RESOLVES_TO]->(h:host)
RETURN d.name, collect(s.name) as subdomains, collect(DISTINCT h.ip) as ips
ORDER BY d.name
```

### Prerequisites

**Required Tools:**
- nmap (host discovery, service detection)
- masscan (fast port scanning)
- httpx (HTTP probing)
- nuclei (vulnerability scanning)
- subfinder (subdomain discovery)
- amass (subdomain enumeration)

**Install Tools (Debian/Ubuntu):**
```bash
# Install via apt
sudo apt-get update
sudo apt-get install -y nmap

# Install Go tools (requires Go 1.21+)
go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest
go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest
go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest

# Install masscan from source
git clone https://github.com/robertdavidgraham/masscan
cd masscan && make && sudo make install

# Install amass
go install -v github.com/owasp-amass/amass/v4/...@master
```

**Required Infrastructure:**
- Neo4j (knowledge graph) - running on localhost:7687
- Qdrant (vector store) - running on localhost:6334
- Gibson daemon - running with `gibson daemon start`

**Start Infrastructure:**
```bash
cd /home/anthony/Code/zero-day.ai
./dev/start-dev.sh
```

### Safety and Best Practices

**WARNING:** Network reconnaissance tools perform active scanning that may:
- Generate network traffic that triggers security alerts
- Impact network performance
- Violate organizational security policies
- Be illegal if run against networks you don't own or have permission to test

**Safe Usage:**
- Only scan networks you own or have explicit permission to test
- Use the dev environment for learning and testing
- Avoid scanning production networks without approval
- Configure appropriate scan timeouts and rate limits
- Review organizational security policies before running

**Local Network Only:**
The debug agent is designed for local network reconnaissance:
- Auto-detects private IP ranges (10.x.x.x, 172.16-31.x.x, 192.168.x.x)
- Scans local subnet only (typically /24 network)
- Respects mission bounds (timeouts, max tool calls)
- Suitable for home labs, dev environments, internal testing

### Example Outputs

**Discovery Phase:**
```
[INFO] Discovering local subnet...
[INFO] Detected local IP: 192.168.1.100
[INFO] Scanning CIDR: 192.168.1.0/24
[INFO] Running nmap for host discovery...
[INFO] Found 12 live hosts
[INFO] Running masscan for port scanning...
[INFO] Discovered 45 open ports
[INFO] Created 12 host nodes, 45 port nodes in knowledge graph
```

**Intelligence Analysis:**
```json
{
  "summary": "Local network reconnaissance identified 12 active hosts with 45 open ports. Web services detected on 3 hosts, including development servers and home automation systems. 2 high-severity vulnerabilities discovered in exposed admin panels.",
  "risk_assessment": "MEDIUM - Development environment with some exposed services. Admin panels lack authentication. Recommend implementing access controls and network segmentation.",
  "attack_paths": [
    {
      "path": "192.168.1.20:8080 -> Unauth Admin Panel -> RCE via Plugin Upload",
      "likelihood": "high",
      "impact": "high"
    }
  ],
  "recommendations": [
    "Enable authentication on admin panels at 192.168.1.20:8080",
    "Update vulnerable nginx version on 192.168.1.10",
    "Implement network segmentation for IoT devices"
  ],
  "confidence": 0.85
}
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
  - `network-recon` - Full network reconnaissance workflow
  - `network-recon-discover` - Network discovery and host scanning
  - `network-recon-probe` - HTTP endpoint probing
  - `network-recon-scan` - Vulnerability scanning
  - `network-recon-domain` - Domain enumeration
  - `network-recon-analyze` - Intelligence generation
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
│   ├── framework/      # Framework test modules
│   │   ├── module.go   # Base module
│   │   └── comprehensive_tests.go  # Framework tests
│   ├── network/        # Network discovery
│   │   ├── interfaces.go  # Discovery interfaces
│   │   ├── discover.go    # Subnet detection
│   │   └── hosts.go       # /etc/hosts parsing
│   ├── recon/          # Reconnaissance orchestration
│   │   ├── interfaces.go  # Recon interfaces
│   │   ├── runner.go      # Phase execution
│   │   └── extractor.go   # Taxonomy extraction
│   └── intelligence/   # LLM intelligence generation
│       ├── interfaces.go  # Generator interfaces
│       ├── prompts.go     # LLM prompt templates
│       ├── generator.go   # Core generation logic
│       └── parser.go      # Response parsing
└── testdata/           # Test fixtures
    └── debug-recon-mission.yaml  # Network recon mission
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

### Completed (Phase 3 - Network Reconnaissance)
- ✅ Network discovery module (subnet detection, /etc/hosts parsing)
- ✅ Reconnaissance runner (phase orchestration, tool execution)
- ✅ Taxonomy extraction (knowledge graph population)
- ✅ Intelligence generator (LLM-powered analysis)
- ✅ Mission workflow YAML (five-phase reconnaissance)
- ✅ Integration with dev environment
- ✅ Intelligence node type in GraphRAG taxonomy
- ✅ Network reconnaissance execution modes

### Future Enhancements
- Additional test coverage for individual requirements
- Report generation with findings submission
- Integration with CI/CD systems
- Performance benchmarking
- Test fixtures for workflow engine tests
- Advanced correlation algorithms for intelligence generation
- Support for custom reconnaissance workflows

## Requirements Coverage

The debug agent provides two primary capabilities:

### Diagnostic Testing
Validates **32 requirements** across SDK and Framework:
- Requirements 1-16: SDK functionality
- Requirements 17-31: Framework functionality
- Requirement 32: Comprehensive diagnostic reporting

### Network Reconnaissance
Implements comprehensive local network reconnaissance:
- Auto-discovery of local subnet and /etc/hosts domains
- Five-phase reconnaissance workflow (discover, probe, scan, domain, analyze)
- Tool orchestration (nmap, masscan, httpx, nuclei, subfinder, amass)
- Knowledge graph population with reconnaissance entities
- LLM-powered intelligence generation with risk assessment and attack paths

## License

MIT

## Contributing

The debug agent is part of the Gibson project. Contributions are welcome!

## Support

For issues or questions, consult the Gibson documentation or raise an issue in the Gibson repository.

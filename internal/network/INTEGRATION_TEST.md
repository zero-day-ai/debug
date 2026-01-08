# Network Recon Module Integration Test Procedure

This document describes how to perform end-to-end integration testing of the Network Recon module to validate the complete pipeline from target configuration through finding submission.

## Prerequisites

1. **Gibson Framework Running**: Ensure the Gibson framework is operational with all subsystems:
   - Neo4j (GraphRAG storage)
   - Langfuse (LLM observability)
   - Finding store
   - Tool registry with ping and nmap tools installed

2. **Debug Agent Built**: Build the debug agent with the network-recon module:
   ```bash
   cd /home/anthony/Code/zero-day.ai/opensource/agents/debug
   go build -o debug-agent .
   ```

3. **Network Access**: Ensure you have permission to scan the target network.

## Test Procedure

### Step 1: Create a Network Target

Create a target with subnet configuration:

```bash
gibson target add test-network \
  --type network \
  --connection '{"subnet":"192.168.1.0/24"}'
```

Expected output: Target created successfully with ID.

### Step 2: Run Network Reconnaissance Attack

Execute the debug agent in network-recon mode:

```bash
gibson attack \
  --target test-network \
  --agent debug-agent \
  --metadata '{"mode":"network-recon"}'
```

Expected output: Attack initiated, returns attack ID.

### Step 3: Monitor Execution

Watch the attack execution logs to verify each phase completes:

```bash
gibson attack logs <attack-id> --follow
```

Expected phases:
1. **Parse Subnet from Target** - PASS (subnet validated)
2. **Ping Sweep** - PASS (live hosts discovered)
3. **Nmap Port Scan** - PASS (ports scanned, services identified)
4. **GraphRAG Storage** - PASS (nodes/relationships created)
5. **LLM Security Analysis** - PASS (analysis completed with risk level)
6. **Finding Submission** - PASS (findings submitted for high-risk services)

### Step 4: Verify Neo4j Graph Storage

Query Neo4j to verify scan data was stored correctly:

```cypher
// Find the attack correlation node
MATCH (a:AttackCorrelation {attack_id: "<attack-id>"})
RETURN a;

// Find all scan nodes for this attack
MATCH (a:AttackCorrelation {attack_id: "<attack-id>"})-[:PERFORMED_SCAN]->(s:NetworkScan)
RETURN s;

// Find all discovered hosts
MATCH (s:NetworkScan {attack_id: "<attack-id>"})-[:SCANNED_HOST]->(h:Host)
RETURN h;

// Find all open ports
MATCH (h:Host {attack_id: "<attack-id>"})-[:HAS_PORT]->(p:Port)
RETURN h.ip, p.port, p.protocol, p.service, p.version;

// Find LLM analysis
MATCH (a:AttackCorrelation {attack_id: "<attack-id>"})-[:ANALYZED_BY]->(llm:LLMAnalysis)
RETURN llm;
```

Expected results:
- AttackCorrelation node exists
- NetworkScan node with subnet and host/port counts
- Host nodes for each discovered IP
- Port nodes for each open port
- LLMAnalysis node with summary and risk level

### Step 5: Verify Langfuse LLM Traces

Access Langfuse dashboard to verify LLM call was logged:

1. Navigate to Langfuse dashboard
2. Search for traces with attack ID
3. Find the LLM completion for network analysis
4. Verify prompt contains:
   - Subnet information
   - List of discovered hosts
   - Open ports and services
   - Request for JSON-formatted analysis
5. Verify response contains:
   - Summary of network exposure
   - Risk level (low/medium/high/critical)
   - High-risk services list
   - Security recommendations

### Step 6: Verify Finding Submission

Query the finding store to verify findings were created:

```bash
gibson finding list --attack <attack-id>
```

Expected results:
- One finding per high-risk service identified by LLM
- Findings include:
  - Title: "High-Risk Network Service Detected: <service:port>"
  - Category: "information_disclosure"
  - Severity: Matches LLM risk level
  - Evidence: Network scan results + LLM analysis
  - Remediation: Recommendations from LLM

### Step 7: Validate Complete Traceability

Verify you can trace from attack to all artifacts:

```bash
# Get attack summary
gibson attack show <attack-id>

# Should show:
# - Target: test-network
# - Agent: debug-agent
# - Status: completed
# - Results: 6 tests passed
# - Findings: N findings (where N = number of high-risk services)
```

## Expected End-to-End Flow

```
Attack Initiated
    ↓
Parse Subnet (192.168.1.0/24)
    ↓
Ping Sweep (discover live hosts)
    ↓
Nmap Scan (identify open ports/services)
    ↓
Store in Neo4j (create graph nodes/relationships)
    ↓
LLM Analysis (Claude analyzes security posture)
    ↓
Submit Findings (create findings for high-risk services)
    ↓
Attack Complete
```

## Validation Checklist

- [ ] Target created with valid subnet CIDR
- [ ] Attack executed successfully
- [ ] All 6 test phases passed
- [ ] Neo4j contains AttackCorrelation node
- [ ] Neo4j contains NetworkScan node with correct subnet
- [ ] Neo4j contains Host nodes for discovered IPs
- [ ] Neo4j contains Port nodes for open ports
- [ ] Neo4j contains LLMAnalysis node with risk assessment
- [ ] Langfuse shows LLM completion trace
- [ ] LLM prompt includes scan results
- [ ] LLM response is valid JSON with required fields
- [ ] Findings submitted to finding store
- [ ] Finding evidence includes scan output
- [ ] Finding evidence includes LLM analysis
- [ ] Findings have appropriate severity levels
- [ ] All artifacts traceable via attack ID

## Troubleshooting

### Ping Sweep Skipped
**Symptom**: "Ping tool not available or failed"
**Solution**: Ensure ping tool is registered in tool registry and executable

### Nmap Scan Skipped
**Symptom**: "Nmap tool not available or failed"
**Solution**: Ensure nmap tool is registered in tool registry and nmap binary is installed

### GraphRAG Storage Skipped
**Symptom**: "GraphRAG unavailable"
**Solution**: Verify Neo4j is running and accessible, check GraphRAG health endpoint

### LLM Analysis Skipped
**Symptom**: "LLM completion failed"
**Solution**: Check Anthropic API credentials, verify LLM slot configuration

### Finding Submission Failed
**Symptom**: "Failed to submit any findings"
**Solution**: Check finding store availability, verify mission context is set

## Performance Benchmarks

For a /24 subnet (256 IPs):
- Ping sweep: < 30 seconds
- Nmap scan: < 10 minutes (depends on number of live hosts)
- GraphRAG storage: < 5 seconds
- LLM analysis: < 60 seconds
- Total execution: < 15 minutes

## Security Notes

- Only scan networks you have permission to test
- The module respects /24 subnet limit (max 256 IPs) for safety
- Network and broadcast addresses are excluded from scanning
- Tool execution goes through harness (no direct exec)
- Findings do not include raw credentials if discovered

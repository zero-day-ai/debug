package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// analysisPromptTemplate is the LLM prompt template for network scan analysis
const analysisPromptTemplate = `You are a network security analyst. Analyze the following network scan results and provide a security assessment.

## Scan Information
- Subnet: {{.Subnet}}
- Scan Time: {{.ScanTime}}
- Live Hosts: {{.HostCount}}
- Open Ports: {{.PortCount}}

## Discovered Hosts and Services
{{range .Hosts}}
### {{.IP}}{{if .Hostname}} ({{.Hostname}}){{end}}
{{range .Ports}}
- Port {{.Port}}/{{.Protocol}}: {{.Service}}{{if .Version}} v{{.Version}}{{end}}
{{end}}
{{end}}

## Analysis Required
1. Provide a brief summary of the network exposure
2. Identify any high-risk services (e.g., Telnet, FTP, unencrypted protocols)
3. Rate the overall risk level: low, medium, high, or critical
4. Provide 2-3 specific security recommendations

Respond in JSON format:
{
  "summary": "...",
  "risk_level": "low|medium|high|critical",
  "high_risk_services": ["service:port", ...],
  "recommendations": ["...", ...]
}`

// promptData holds the data for rendering the analysis prompt template
type promptData struct {
	Subnet    string
	ScanTime  string
	HostCount int
	PortCount int
	Hosts     []HostResult
}

// buildAnalysisPrompt constructs the LLM analysis prompt from scan results
func buildAnalysisPrompt(subnet string, liveHosts []string, scan *ScanResults) (string, error) {
	// Prepare template data
	data := promptData{
		Subnet:    subnet,
		HostCount: len(liveHosts),
	}

	if scan != nil {
		data.ScanTime = scan.ScanTime.Format("2006-01-02 15:04:05 MST")
		data.Hosts = scan.Hosts

		// Count total open ports
		for _, host := range scan.Hosts {
			for _, port := range host.Ports {
				if port.State == "open" {
					data.PortCount++
				}
			}
		}
	} else {
		data.ScanTime = "N/A"
		data.Hosts = []HostResult{}
	}

	// Parse template
	tmpl, err := template.New("analysis").Parse(analysisPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// parseLLMResponse extracts LLMAnalysis from the LLM's JSON response
func parseLLMResponse(response string) (*LLMAnalysis, error) {
	// Try to find JSON in the response (LLM might include extra text)
	jsonStart := -1
	jsonEnd := -1

	// Find first { and last }
	for i := 0; i < len(response); i++ {
		if response[i] == '{' && jsonStart == -1 {
			jsonStart = i
		}
		if response[i] == '}' {
			jsonEnd = i + 1
		}
	}

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart:jsonEnd]

	// Parse JSON
	var analysis LLMAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Store the raw response
	analysis.RawResponse = response

	// Validate required fields
	if analysis.Summary == "" {
		return nil, fmt.Errorf("missing required field: summary")
	}
	if analysis.RiskLevel == "" {
		return nil, fmt.Errorf("missing required field: risk_level")
	}

	// Normalize risk level to lowercase
	switch analysis.RiskLevel {
	case "low", "medium", "high", "critical":
		// Already normalized
	default:
		// Try to normalize
		riskLower := ""
		for _, c := range analysis.RiskLevel {
			if c >= 'A' && c <= 'Z' {
				riskLower += string(c + 32)
			} else {
				riskLower += string(c)
			}
		}
		analysis.RiskLevel = riskLower
	}

	return &analysis, nil
}

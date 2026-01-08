package network

import (
	"net"
	"testing"
)

// TestEnumerateIPs tests IP enumeration from CIDR notation
func TestEnumerateIPs(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantLen int
		wantErr bool
	}{
		{
			name:    "Valid /30 subnet",
			cidr:    "192.168.1.0/30",
			wantLen: 2, // Excludes network and broadcast
			wantErr: false,
		},
		{
			name:    "Valid /29 subnet",
			cidr:    "10.0.0.0/29",
			wantLen: 6, // Excludes network and broadcast
			wantErr: false,
		},
		{
			name:    "Valid /24 subnet",
			cidr:    "172.16.0.0/24",
			wantLen: 254, // Excludes network and broadcast
			wantErr: false,
		},
		{
			name:    "Invalid CIDR",
			cidr:    "not-a-cidr",
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "Invalid IP format",
			cidr:    "999.999.999.999/24",
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enumerateIPs(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("enumerateIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("enumerateIPs() got %d IPs, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// TestEnumerateIPsContent tests that enumerated IPs are valid
func TestEnumerateIPsContent(t *testing.T) {
	ips, err := enumerateIPs("192.168.1.0/30")
	if err != nil {
		t.Fatalf("enumerateIPs() unexpected error: %v", err)
	}

	// Check that we got expected IPs (excluding network .0 and broadcast .3)
	expected := []string{"192.168.1.1", "192.168.1.2"}
	if len(ips) != len(expected) {
		t.Fatalf("enumerateIPs() got %d IPs, want %d", len(ips), len(expected))
	}

	for i, ip := range ips {
		if ip != expected[i] {
			t.Errorf("enumerateIPs() ip[%d] = %s, want %s", i, ip, expected[i])
		}

		// Validate that each IP is valid
		if net.ParseIP(ip) == nil {
			t.Errorf("enumerateIPs() invalid IP: %s", ip)
		}
	}
}

// TestBuildAnalysisPrompt tests LLM prompt construction
func TestBuildAnalysisPrompt(t *testing.T) {
	scan := &ScanResults{
		Hosts: []HostResult{
			{
				IP:       "192.168.1.100",
				Hostname: "test-server",
				Status:   "up",
				Ports: []PortResult{
					{Port: 22, Protocol: "tcp", State: "open", Service: "ssh", Version: "7.4"},
					{Port: 80, Protocol: "tcp", State: "open", Service: "http"},
				},
			},
		},
	}

	prompt, err := buildAnalysisPrompt("192.168.1.0/24", []string{"192.168.1.100"}, scan)
	if err != nil {
		t.Fatalf("buildAnalysisPrompt() unexpected error: %v", err)
	}

	// Check that prompt contains expected elements
	expectedStrings := []string{
		"192.168.1.0/24",           // Subnet
		"192.168.1.100",            // IP
		"test-server",              // Hostname
		"Port 22/tcp: ssh v7.4",    // Service with version
		"Port 80/tcp: http",        // Service without version
		"risk_level",               // JSON field
		"high_risk_services",       // JSON field
		"recommendations",          // JSON field
	}

	for _, expected := range expectedStrings {
		found := false
		for i := 0; i < len(prompt)-len(expected)+1; i++ {
			if prompt[i:i+len(expected)] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("buildAnalysisPrompt() prompt missing expected string: %s", expected)
		}
	}
}

// TestParseLLMResponse tests JSON response parsing
func TestParseLLMResponse(t *testing.T) {
	tests := []struct {
		name    string
		response string
		wantErr bool
	}{
		{
			name: "Valid JSON response",
			response: `{
				"summary": "Test summary",
				"risk_level": "high",
				"high_risk_services": ["telnet:23", "ftp:21"],
				"recommendations": ["Disable telnet", "Use SFTP"]
			}`,
			wantErr: false,
		},
		{
			name: "JSON with extra text",
			response: `Here's the analysis:
			{
				"summary": "Test summary",
				"risk_level": "medium",
				"high_risk_services": [],
				"recommendations": ["Update services"]
			}
			Hope this helps!`,
			wantErr: false,
		},
		{
			name:     "Missing summary",
			response: `{"risk_level": "low", "high_risk_services": [], "recommendations": []}`,
			wantErr:  true,
		},
		{
			name:     "Missing risk_level",
			response: `{"summary": "Test", "high_risk_services": [], "recommendations": []}`,
			wantErr:  true,
		},
		{
			name:     "No JSON in response",
			response: `This is just plain text without JSON`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := parseLLMResponse(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLLMResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if analysis.Summary == "" {
					t.Error("parseLLMResponse() summary is empty")
				}
				if analysis.RiskLevel == "" {
					t.Error("parseLLMResponse() risk_level is empty")
				}
				if analysis.RawResponse != tt.response {
					t.Error("parseLLMResponse() raw_response not preserved")
				}
			}
		})
	}
}

// TestFormatScanResultsForEvidence tests evidence formatting
func TestFormatScanResultsForEvidence(t *testing.T) {
	scan := &ScanResults{
		Hosts: []HostResult{
			{
				IP:       "192.168.1.100",
				Hostname: "test-server",
				Status:   "up",
				Ports: []PortResult{
					{Port: 22, Protocol: "tcp", State: "open", Service: "ssh"},
				},
			},
		},
	}

	evidence := formatScanResultsForEvidence(scan)

	// Check that evidence contains expected elements
	expectedStrings := []string{
		"192.168.1.100",
		"test-server",
		"Port 22/tcp: ssh",
	}

	for _, expected := range expectedStrings {
		found := false
		for i := 0; i < len(evidence)-len(expected)+1; i++ {
			if evidence[i:i+len(expected)] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("formatScanResultsForEvidence() missing expected string: %s", expected)
		}
	}
}

// TestFormatRecommendations tests recommendation formatting
func TestFormatRecommendations(t *testing.T) {
	recommendations := []string{
		"Disable telnet service",
		"Update SSH to latest version",
		"Enable firewall",
	}

	result := formatRecommendations(recommendations)

	// Check for numbered list format
	expectedStrings := []string{
		"1. Disable telnet service",
		"2. Update SSH to latest version",
		"3. Enable firewall",
	}

	for _, expected := range expectedStrings {
		found := false
		for i := 0; i < len(result)-len(expected)+1; i++ {
			if result[i:i+len(expected)] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("formatRecommendations() missing expected string: %s", expected)
		}
	}
}

// TestIncIP tests IP increment function
func TestIncIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{
			name: "Simple increment",
			ip:   "192.168.1.1",
			want: "192.168.1.2",
		},
		{
			name: "Overflow to next octet",
			ip:   "192.168.1.255",
			want: "192.168.2.0",
		},
		{
			name: "Multiple octet overflow",
			ip:   "192.168.255.255",
			want: "192.169.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip).To4()
			incIP(ip)
			got := ip.String()
			if got != tt.want {
				t.Errorf("incIP() = %s, want %s", got, tt.want)
			}
		})
	}
}

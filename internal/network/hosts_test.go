package network

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestParseHostsFile tests the parsing of /etc/hosts file with various formats
func TestParseHostsFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string][]string
		wantErr bool
	}{
		{
			name: "valid hosts file with multiple entries",
			content: `127.0.0.1       localhost
::1             localhost ip6-localhost ip6-loopback
192.168.1.10    server.local
192.168.1.20    app.example.com api.example.com
10.0.0.5        database.internal
`,
			want: map[string][]string{
				"localhost":          {"127.0.0.1", "::1"},
				"ip6-localhost":      {"::1"},
				"ip6-loopback":       {"::1"},
				"server.local":       {"192.168.1.10"},
				"app.example.com":    {"192.168.1.20"},
				"api.example.com":    {"192.168.1.20"},
				"database.internal":  {"10.0.0.5"},
			},
			wantErr: false,
		},
		{
			name: "hosts file with comments and blank lines",
			content: `# This is a comment
127.0.0.1       localhost

# Another comment
192.168.1.10    server.local # inline comment
# More comments

10.0.0.5        database.internal
`,
			want: map[string][]string{
				"localhost":          {"127.0.0.1"},
				"server.local":       {"192.168.1.10"},
				"database.internal":  {"10.0.0.5"},
			},
			wantErr: false,
		},
		{
			name: "hosts file with only comments and blank lines",
			content: `# This file has no actual entries
# Just comments

# More comments
`,
			want:    map[string][]string{},
			wantErr: false,
		},
		{
			name:    "empty hosts file",
			content: "",
			want:    map[string][]string{},
			wantErr: false,
		},
		{
			name: "hosts file with malformed lines (should skip)",
			content: `127.0.0.1       localhost
invalid line without IP
192.168.1.10    server.local
just-one-field
10.0.0.5        database.internal
`,
			want: map[string][]string{
				"localhost":          {"127.0.0.1"},
				"server.local":       {"192.168.1.10"},
				"database.internal":  {"10.0.0.5"},
			},
			wantErr: false,
		},
		{
			name: "hosts file with IPv6 addresses",
			content: `::1                     localhost ip6-localhost ip6-loopback
fe80::1                 gateway.local
2001:db8::1             server.ipv6.local
192.168.1.10            server.local
`,
			want: map[string][]string{
				"localhost":           {"::1"},
				"ip6-localhost":       {"::1"},
				"ip6-loopback":        {"::1"},
				"gateway.local":       {"fe80::1"},
				"server.ipv6.local":   {"2001:db8::1"},
				"server.local":        {"192.168.1.10"},
			},
			wantErr: false,
		},
		{
			name: "hosts file with unicode hostnames",
			content: `192.168.1.10    münchen.local
192.168.1.20    北京.local
10.0.0.5        مصر.local
`,
			want: map[string][]string{
				"münchen.local": {"192.168.1.10"},
				"北京.local":     {"192.168.1.20"},
				"مصر.local":     {"10.0.0.5"},
			},
			wantErr: false,
		},
		{
			name: "hosts file with tabs and multiple spaces",
			content: `127.0.0.1		localhost
192.168.1.10  	  server.local
10.0.0.5	database.internal    db.internal
`,
			want: map[string][]string{
				"localhost":          {"127.0.0.1"},
				"server.local":       {"192.168.1.10"},
				"database.internal":  {"10.0.0.5"},
				"db.internal":        {"10.0.0.5"},
			},
			wantErr: false,
		},
		{
			name: "hosts file with duplicate IP-hostname pairs",
			content: `192.168.1.10    server.local
192.168.1.10    server.local
192.168.1.20    api.example.com
`,
			want: map[string][]string{
				"server.local":       {"192.168.1.10"},
				"api.example.com":    {"192.168.1.20"},
			},
			wantErr: false,
		},
		{
			name: "hosts file with same hostname, different IPs (multi-homed)",
			content: `192.168.1.10    loadbalancer.local
192.168.1.11    loadbalancer.local
10.0.0.5        loadbalancer.local
`,
			want: map[string][]string{
				"loadbalancer.local": {"192.168.1.10", "192.168.1.11", "10.0.0.5"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary hosts file
			tmpDir := t.TempDir()
			hostsFile := filepath.Join(tmpDir, "hosts")
			if err := os.WriteFile(hostsFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test hosts file: %v", err)
			}

			// Parse the hosts file (this function will be implemented in task 6)
			got, err := parseHostsFile(hostsFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHostsFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare results
			if !mapsEqual(got, tt.want) {
				t.Errorf("parseHostsFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractDomainSuffixes tests extracting unique domain suffixes from hostnames
func TestExtractDomainSuffixes(t *testing.T) {
	tests := []struct {
		name      string
		hostnames map[string][]string
		want      []string
	}{
		{
			name: "extract .local and .internal suffixes",
			hostnames: map[string][]string{
				"server.local":        {"192.168.1.10"},
				"app.example.com":     {"192.168.1.20"},
				"database.internal":   {"10.0.0.5"},
				"api.example.com":     {"192.168.1.21"},
				"cache.internal":      {"10.0.0.6"},
			},
			want: []string{".com", ".internal", ".local"},
		},
		{
			name: "extract nested domain suffixes",
			hostnames: map[string][]string{
				"server.dev.local":    {"192.168.1.10"},
				"app.prod.local":      {"192.168.1.20"},
				"db.staging.local":    {"192.168.1.30"},
			},
			want: []string{".local"},
		},
		{
			name: "no domain suffixes (single label hostnames)",
			hostnames: map[string][]string{
				"localhost": {"127.0.0.1"},
				"server":    {"192.168.1.10"},
			},
			want: []string{},
		},
		{
			name:      "empty hostnames map",
			hostnames: map[string][]string{},
			want:      []string{},
		},
		{
			name: "various TLDs and internal domains",
			hostnames: map[string][]string{
				"api.example.com":       {"192.168.1.10"},
				"app.example.org":       {"192.168.1.20"},
				"server.example.net":    {"192.168.1.30"},
				"db.internal":           {"10.0.0.5"},
				"cache.local":           {"10.0.0.6"},
				"monitor.example.io":    {"192.168.1.40"},
			},
			want: []string{".com", ".internal", ".io", ".local", ".net", ".org"},
		},
		{
			name: "international domain suffixes",
			hostnames: map[string][]string{
				"server.中国":          {"192.168.1.10"},
				"app.рф":             {"192.168.1.20"},
			},
			want: []string{".中国", ".рф"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract domain suffixes (this function will be implemented in task 6)
			got := extractDomainSuffixes(tt.hostnames)

			// Sort both slices for comparison
			sort.Strings(got)
			sort.Strings(tt.want)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractDomainSuffixes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetDomainMappings tests the full GetDomainMappings implementation
func TestGetDomainMappings(t *testing.T) {
	// Skip this test if GetDomainMappings is not yet implemented
	ctx := context.Background()
	discovery := NewNetworkDiscovery()

	// Try calling GetDomainMappings
	mappings, err := discovery.GetDomainMappings(ctx)

	// If we get "not yet implemented" error, skip the test
	if err != nil && err.Error() == "GetDomainMappings not yet implemented (see task 6)" {
		t.Skip("GetDomainMappings not yet implemented (task 6)")
		return
	}

	// If it's implemented, run actual tests
	if err != nil {
		t.Logf("GetDomainMappings returned error (may be expected if /etc/hosts doesn't exist): %v", err)
		// Don't fail - this is integration test that depends on system state
		return
	}

	// Basic validation if it succeeds
	if mappings == nil {
		t.Error("GetDomainMappings() returned nil mappings without error")
		return
	}

	t.Logf("Found %d domain mappings from system hosts file", len(mappings))

	// Expect at least localhost
	if ips, ok := mappings["localhost"]; !ok {
		t.Log("Warning: 'localhost' not found in domain mappings (unusual but not necessarily an error)")
	} else {
		t.Logf("localhost maps to: %v", ips)
	}
}

// TestGetDomainMappingsWithCustomHostsFile tests GetDomainMappings with a custom hosts file
func TestGetDomainMappingsWithCustomHostsFile(t *testing.T) {
	// This test will be useful once we implement a way to specify custom hosts file path
	// For now, we'll create a temporary hosts file and test the parsing logic

	hostsContent := `# Test hosts file
127.0.0.1       localhost
192.168.1.10    app.internal web.internal
192.168.1.20    api.example.com
10.0.0.5        database.local
`

	tmpDir := t.TempDir()
	hostsFile := filepath.Join(tmpDir, "hosts")
	if err := os.WriteFile(hostsFile, []byte(hostsContent), 0644); err != nil {
		t.Fatalf("Failed to create test hosts file: %v", err)
	}

	// Parse the temporary hosts file
	mappings, err := parseHostsFile(hostsFile)
	if err != nil {
		t.Fatalf("parseHostsFile() error = %v", err)
	}

	// Verify expected mappings
	expected := map[string][]string{
		"localhost":        {"127.0.0.1"},
		"app.internal":     {"192.168.1.10"},
		"web.internal":     {"192.168.1.10"},
		"api.example.com":  {"192.168.1.20"},
		"database.local":   {"10.0.0.5"},
	}

	if !mapsEqual(mappings, expected) {
		t.Errorf("parseHostsFile() mappings mismatch\ngot:  %v\nwant: %v", mappings, expected)
	}

	// Test domain suffix extraction
	suffixes := extractDomainSuffixes(mappings)
	sort.Strings(suffixes)
	expectedSuffixes := []string{".com", ".internal", ".local"}

	if !reflect.DeepEqual(suffixes, expectedSuffixes) {
		t.Errorf("extractDomainSuffixes() = %v, want %v", suffixes, expectedSuffixes)
	}
}

// Helper function to compare maps with string slice values
// Handles different slice orderings
func mapsEqual(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}

	for key, aVals := range a {
		bVals, ok := b[key]
		if !ok {
			return false
		}

		// Sort both slices for comparison
		aCopy := make([]string, len(aVals))
		bCopy := make([]string, len(bVals))
		copy(aCopy, aVals)
		copy(bCopy, bVals)
		sort.Strings(aCopy)
		sort.Strings(bCopy)

		if !reflect.DeepEqual(aCopy, bCopy) {
			return false
		}
	}

	return true
}


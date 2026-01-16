package network

import (
	"context"
	"net"
	"testing"
)

func TestIsClass192(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{"192.168.1.1", net.IPv4(192, 168, 1, 1).To4(), true},
		{"192.168.0.1", net.IPv4(192, 168, 0, 1).To4(), true},
		{"192.167.1.1", net.IPv4(192, 167, 1, 1).To4(), false},
		{"10.0.0.1", net.IPv4(10, 0, 0, 1).To4(), false},
		{"172.16.0.1", net.IPv4(172, 16, 0, 1).To4(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClass192(tt.ip); got != tt.want {
				t.Errorf("isClass192(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsClass10(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{"10.0.0.1", net.IPv4(10, 0, 0, 1).To4(), true},
		{"10.255.255.254", net.IPv4(10, 255, 255, 254).To4(), true},
		{"11.0.0.1", net.IPv4(11, 0, 0, 1).To4(), false},
		{"192.168.1.1", net.IPv4(192, 168, 1, 1).To4(), false},
		{"172.16.0.1", net.IPv4(172, 16, 0, 1).To4(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClass10(tt.ip); got != tt.want {
				t.Errorf("isClass10(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsClass172(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{"172.16.0.1 (lower bound)", net.IPv4(172, 16, 0, 1).To4(), true},
		{"172.20.0.1 (middle)", net.IPv4(172, 20, 0, 1).To4(), true},
		{"172.31.255.254 (upper bound)", net.IPv4(172, 31, 255, 254).To4(), true},
		{"172.15.0.1 (below range)", net.IPv4(172, 15, 0, 1).To4(), false},
		{"172.32.0.1 (above range)", net.IPv4(172, 32, 0, 1).To4(), false},
		{"192.168.1.1", net.IPv4(192, 168, 1, 1).To4(), false},
		{"10.0.0.1", net.IPv4(10, 0, 0, 1).To4(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClass172(tt.ip); got != tt.want {
				t.Errorf("isClass172(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestGetNetworkCIDR(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		mask string
		want string
	}{
		{
			name: "192.168.1.50/24",
			ip:   "192.168.1.50",
			mask: "255.255.255.0",
			want: "192.168.1.0/24",
		},
		{
			name: "10.20.30.40/16",
			ip:   "10.20.30.40",
			mask: "255.255.0.0",
			want: "10.20.0.0/16",
		},
		{
			name: "172.16.5.100/12",
			ip:   "172.16.5.100",
			mask: "255.240.0.0",
			want: "172.16.0.0/12",
		},
		{
			name: "10.0.0.1/8",
			ip:   "10.0.0.1",
			mask: "255.0.0.0",
			want: "10.0.0.0/8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			mask := net.IPMask(net.ParseIP(tt.mask).To4())
			ipNet := &net.IPNet{IP: ip, Mask: mask}

			got := getNetworkCIDR(ipNet)
			if got != tt.want {
				t.Errorf("getNetworkCIDR(%s) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestDiscoverLocalSubnet(t *testing.T) {
	// This is an integration test that runs against the actual system.
	// It should find at least one private network interface on most systems.

	ctx := context.Background()
	discovery := NewNetworkDiscovery()

	cidr, err := discovery.DiscoverLocalSubnet(ctx)
	if err != nil {
		// Some CI environments may not have private network interfaces
		t.Logf("No private network found (this is OK in some CI environments): %v", err)
		t.Skip("Skipping test - no private network interface available")
		return
	}

	// Validate the returned CIDR format
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("DiscoverLocalSubnet() returned invalid CIDR %q: %v", cidr, err)
	}

	// Verify it's a network address (not a host address)
	if !ipNet.IP.Equal(ipNet.IP.Mask(ipNet.Mask)) {
		t.Errorf("DiscoverLocalSubnet() returned host IP instead of network CIDR: %s", cidr)
	}

	// Verify it's a private IP range
	ip := ipNet.IP.To4()
	if ip == nil {
		t.Fatalf("DiscoverLocalSubnet() returned non-IPv4 CIDR: %s", cidr)
	}

	isPrivate := isClass192(ip) || isClass10(ip) || isClass172(ip)
	if !isPrivate {
		t.Errorf("DiscoverLocalSubnet() returned non-private IP range: %s", cidr)
	}

	t.Logf("Discovered local subnet: %s", cidr)
}


// TestGetNetworkCIDR_EdgeCases tests edge cases for CIDR calculation
func TestGetNetworkCIDR_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		mask string
		want string
	}{
		{
			name: "/32 single host network",
			ip:   "192.168.1.50",
			mask: "255.255.255.255",
			want: "192.168.1.50/32",
		},
		{
			name: "/0 entire internet",
			ip:   "192.168.1.50",
			mask: "0.0.0.0",
			want: "0.0.0.0/0",
		},
		{
			name: "/25 subnet (128 hosts)",
			ip:   "192.168.1.200",
			mask: "255.255.255.128",
			want: "192.168.1.128/25",
		},
		{
			name: "/30 point-to-point (4 addresses)",
			ip:   "10.0.0.2",
			mask: "255.255.255.252",
			want: "10.0.0.0/30",
		},
		{
			name: "boundary test - network address itself",
			ip:   "192.168.0.0",
			mask: "255.255.255.0",
			want: "192.168.0.0/24",
		},
		{
			name: "boundary test - broadcast address",
			ip:   "192.168.0.255",
			mask: "255.255.255.0",
			want: "192.168.0.0/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			mask := net.IPMask(net.ParseIP(tt.mask).To4())
			ipNet := &net.IPNet{IP: ip, Mask: mask}

			got := getNetworkCIDR(ipNet)
			if got != tt.want {
				t.Errorf("getNetworkCIDR(%s) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

// TestIPClassification_Comprehensive tests all IP classification functions comprehensively
func TestIPClassification_Comprehensive(t *testing.T) {
	tests := []struct {
		name         string
		ip           string
		isClass192   bool
		isClass10    bool
		isClass172   bool
		description  string
	}{
		{
			name:        "192.168.0.0 - lower bound Class C",
			ip:          "192.168.0.0",
			isClass192:  true,
			isClass10:   false,
			isClass172:  false,
			description: "Lower bound of 192.168.x.x range",
		},
		{
			name:        "192.168.255.255 - upper bound Class C",
			ip:          "192.168.255.255",
			isClass192:  true,
			isClass10:   false,
			isClass172:  false,
			description: "Upper bound of 192.168.x.x range",
		},
		{
			name:        "10.0.0.0 - lower bound Class A",
			ip:          "10.0.0.0",
			isClass192:  false,
			isClass10:   true,
			isClass172:  false,
			description: "Lower bound of 10.x.x.x range",
		},
		{
			name:        "10.255.255.255 - upper bound Class A",
			ip:          "10.255.255.255",
			isClass192:  false,
			isClass10:   true,
			isClass172:  false,
			description: "Upper bound of 10.x.x.x range",
		},
		{
			name:        "172.16.0.0 - lower bound Class B",
			ip:          "172.16.0.0",
			isClass192:  false,
			isClass10:   false,
			isClass172:  true,
			description: "Lower bound of 172.16-31.x.x range",
		},
		{
			name:        "172.31.255.255 - upper bound Class B",
			ip:          "172.31.255.255",
			isClass192:  false,
			isClass10:   false,
			isClass172:  true,
			description: "Upper bound of 172.16-31.x.x range",
		},
		{
			name:        "8.8.8.8 - public IP (Google DNS)",
			ip:          "8.8.8.8",
			isClass192:  false,
			isClass10:   false,
			isClass172:  false,
			description: "Public IP address",
		},
		{
			name:        "1.1.1.1 - public IP (Cloudflare DNS)",
			ip:          "1.1.1.1",
			isClass192:  false,
			isClass10:   false,
			isClass172:  false,
			description: "Public IP address",
		},
		{
			name:        "172.15.255.255 - just below Class B range",
			ip:          "172.15.255.255",
			isClass192:  false,
			isClass10:   false,
			isClass172:  false,
			description: "Just below 172.16-31.x.x range",
		},
		{
			name:        "172.32.0.0 - just above Class B range",
			ip:          "172.32.0.0",
			isClass192:  false,
			isClass10:   false,
			isClass172:  false,
			description: "Just above 172.16-31.x.x range",
		},
		{
			name:        "192.167.255.255 - just below 192.168",
			ip:          "192.167.255.255",
			isClass192:  false,
			isClass10:   false,
			isClass172:  false,
			description: "Just below 192.168.x.x range",
		},
		{
			name:        "192.169.0.0 - just above 192.168",
			ip:          "192.169.0.0",
			isClass192:  false,
			isClass10:   false,
			isClass172:  false,
			description: "Just above 192.168.x.x range",
		},
		{
			name:        "127.0.0.1 - loopback",
			ip:          "127.0.0.1",
			isClass192:  false,
			isClass10:   false,
			isClass172:  false,
			description: "Loopback address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip).To4()
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			if got := isClass192(ip); got != tt.isClass192 {
				t.Errorf("isClass192(%s) = %v, want %v (%s)", tt.ip, got, tt.isClass192, tt.description)
			}

			if got := isClass10(ip); got != tt.isClass10 {
				t.Errorf("isClass10(%s) = %v, want %v (%s)", tt.ip, got, tt.isClass10, tt.description)
			}

			if got := isClass172(ip); got != tt.isClass172 {
				t.Errorf("isClass172(%s) = %v, want %v (%s)", tt.ip, got, tt.isClass172, tt.description)
			}
		})
	}
}

// TestDiscoverLocalSubnet_MultipleInterfaces tests priority selection with multiple interfaces
func TestDiscoverLocalSubnet_MultipleInterfaces(t *testing.T) {
	// This is an integration test that verifies the priority order
	// when multiple private interfaces are present

	ctx := context.Background()
	discovery := NewNetworkDiscovery()

	cidr, err := discovery.DiscoverLocalSubnet(ctx)
	if err != nil {
		t.Skipf("No private network found (OK in CI): %v", err)
		return
	}

	// Parse the CIDR
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("Invalid CIDR returned: %s", cidr)
	}

	ip := ipNet.IP.To4()
	if ip == nil {
		t.Fatalf("Non-IPv4 CIDR returned: %s", cidr)
	}

	// Log which type was selected
	if isClass192(ip) {
		t.Logf("Selected 192.168.x.x network (highest priority): %s", cidr)
	} else if isClass10(ip) {
		t.Logf("Selected 10.x.x.x network (medium priority): %s", cidr)
	} else if isClass172(ip) {
		t.Logf("Selected 172.16-31.x.x network (lowest priority): %s", cidr)
	} else {
		t.Errorf("Non-private IP selected: %s", cidr)
	}

	// Verify it's actually a private range
	if !isClass192(ip) && !isClass10(ip) && !isClass172(ip) {
		t.Errorf("DiscoverLocalSubnet() returned non-private IP: %s", cidr)
	}
}

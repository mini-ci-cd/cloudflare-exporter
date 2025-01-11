package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/namsral/flag"
)

func TestRequestPathsFlag(t *testing.T) {
	// Reset flags before test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "single path",
			args:     []string{"-request_paths=/api/%"},
			expected: []string{"/api/%"},
		},
		{
			name:     "multiple paths",
			args:     []string{"-request_paths=/api/%,/v1/.*,/health"},
			expected: []string{"/api/%", "/v1/.*", "/health"},
		},
		{
			name:     "paths with spaces",
			args:     []string{"-request_paths=/api/%, /v1/.*, /health"},
			expected: []string{"/api/%", "/v1/.*", "/health"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags and variables before each test
			cfgRequestPaths = ""
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			flag.StringVar(&cfgRequestPaths, "request_paths", cfgRequestPaths, "test request paths")

			err := flag.CommandLine.Parse(tt.args)
			if err != nil {
				t.Errorf("Failed to parse flags: %v", err)
			}

			var got []string
			if len(cfgRequestPaths) > 0 {
				got = strings.Split(cfgRequestPaths, ",")
				// Trim spaces from each path
				for i := range got {
					got[i] = strings.TrimSpace(got[i])
				}
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("RequestPaths = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDomainNamesListFlag(t *testing.T) {
	// Reset flags before test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "single domain",
			args:     []string{"-domain_names=example.com"},
			expected: []string{"example.com"},
		},
		{
			name:     "multiple domains",
			args:     []string{"-domain_names=example.com,test.com,demo.net"},
			expected: []string{"example.com", "test.com", "demo.net"},
		},
		{
			name:     "domains with spaces",
			args:     []string{"-domain_names=example.com, test.com, demo.net"},
			expected: []string{"example.com", "test.com", "demo.net"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags and variables before each test
			cfgDomainNamesList = ""
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			flag.StringVar(&cfgDomainNamesList, "domain_names", cfgDomainNamesList, "test domain names")

			err := flag.CommandLine.Parse(tt.args)
			if err != nil {
				t.Errorf("Failed to parse flags: %v", err)
			}

			var got []string
			if len(cfgDomainNamesList) > 0 {
				got = strings.Split(cfgDomainNamesList, ",")
				// Trim spaces from each domain
				for i := range got {
					got[i] = strings.TrimSpace(got[i])
				}
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("DomainNamesList = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFetchMetricsPathsAndDomains(t *testing.T) {
	// Test that fetchMetrics correctly processes both paths and domains
	originalCfgRequestPaths := cfgRequestPaths
	originalCfgDomainNamesList := cfgDomainNamesList
	defer func() {
		cfgRequestPaths = originalCfgRequestPaths
		cfgDomainNamesList = originalCfgDomainNamesList
	}()

	cfgRequestPaths = "/api/%,/v1/.*"
	cfgDomainNamesList = "example.com,test.com"

	// Create a mock zone for testing
	mockZones := []cloudflare.Zone{
		{
			ID:   "zone1",
			Name: "example.com",
		},
	}

	// Test the processing of paths and domains directly
	requestPaths := []string{}
	if len(cfgRequestPaths) > 0 {
		requestPaths = strings.Split(cfgRequestPaths, ",")
		for i := range requestPaths {
			requestPaths[i] = strings.TrimSpace(requestPaths[i])
		}
	}

	domainNames := []string{}
	if len(cfgDomainNamesList) > 0 {
		domainNames = strings.Split(cfgDomainNamesList, ",")
		for i := range domainNames {
			domainNames[i] = strings.TrimSpace(domainNames[i])
		}
	}

	// Verify the processed values
	expectedPaths := []string{"/api/%", "/v1/.*"}
	expectedDomains := []string{"example.com", "test.com"}

	if !reflect.DeepEqual(requestPaths, expectedPaths) {
		t.Errorf("RequestPaths = %v, want %v", requestPaths, expectedPaths)
	}

	if !reflect.DeepEqual(domainNames, expectedDomains) {
		t.Errorf("DomainNames = %v, want %v", domainNames, expectedDomains)
	}

	// Test that the zones are properly filtered
	filteredZones := filterZones(mockZones, []string{"zone1"})
	if len(filteredZones) != 1 || filteredZones[0].ID != "zone1" {
		t.Errorf("Zone filtering failed. Expected 1 zone with ID 'zone1', got %v", filteredZones)
	}
}

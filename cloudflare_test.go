package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/machinebox/graphql"
	"github.com/stretchr/testify/mock"
)

type mockTransport struct {
	mock.Mock
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log raw request details
	log.Printf("GraphQL Request:\n"+
		"URL: %s\n"+
		"Method: %s\n"+
		"Headers: %v\n",
		req.URL, req.Method, req.Header)

	// Read and log request body
	body, _ := io.ReadAll(req.Body)
	log.Printf("Request Body: %s\n", string(body))

	// Restore body for further reads
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	// Verify auth header exists
	if auth := req.Header.Get("Authorization"); auth != "Bearer test-token" {
		return &http.Response{
			StatusCode: 401,
			Body:       io.NopCloser(strings.NewReader(`{"errors": ["Unable to authenticate request"]}`)),
		}, nil
	}

	var requestBody struct {
		Query     string `json:"query"`
		Variables struct {
			ZoneIDs     []string  `json:"zoneIDs"`
			DomainNames []string  `json:"domainNames"`
			RequestPath string    `json:"requestPath"`
			Mintime     time.Time `json:"mintime"`
			Maxtime     time.Time `json:"maxtime"`
			Limit       int       `json:"limit"`
		} `json:"variables"`
	}
	if err := json.Unmarshal(body, &requestBody); err != nil {
		return nil, fmt.Errorf("failed to parse request body: %v", err)
	}

	// Verify complete GraphQL query structure
	expectedQueryParts := []string{
		"query ($zoneIDs: [String!], $mintime: Time!, $maxtime: Time!, $limit: Int!)",
		"zones(filter: { zoneTag_in: $zoneIDs })",
		"clientRequestPath_like: $requestPath",
		"clientRequestHTTPHost_in: $domainNames",
	}
	for _, part := range expectedQueryParts {
		if !strings.Contains(requestBody.Query, part) {
			return nil, fmt.Errorf("GraphQL query missing expected part: %s", part)
		}
	}

	// Verify variables
	expectedPath := "/api/%"
	expectedDomains := []string{"example.com", "test.com"}
	expectedZoneIDs := []string{"zone1"}

	if requestBody.Variables.RequestPath != expectedPath {
		return nil, fmt.Errorf("RequestPath = %v, want %v", requestBody.Variables.RequestPath, expectedPath)
	}
	if !reflect.DeepEqual(requestBody.Variables.DomainNames, expectedDomains) {
		return nil, fmt.Errorf("DomainNames = %v, want %v", requestBody.Variables.DomainNames, expectedDomains)
	}
	if !reflect.DeepEqual(requestBody.Variables.ZoneIDs, expectedZoneIDs) {
		return nil, fmt.Errorf("ZoneIDs = %v, want %v", requestBody.Variables.ZoneIDs, expectedZoneIDs)
	}
	if requestBody.Variables.Limit != 9999 {
		return nil, fmt.Errorf("Limit = %v, want %v", requestBody.Variables.Limit, 9999)
	}

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	mockResponse := `{"data":{"viewer":{"zones":[{"zoneTag":"zone1","httpRequestsEdgeCountryHost":[]}]}}}`
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(mockResponse)),
		Header:     header,
	}, nil
}

func TestFetchZoneTotals(t *testing.T) {
	mockClient := graphql.NewClient(cfGraphQLEndpoint, graphql.WithHTTPClient(&http.Client{
		Transport: &mockTransport{},
	}))

	// Save original and restore after test
	originalNewClient := newGraphQLClient
	defer func() { newGraphQLClient = originalNewClient }()

	// Override with mock
	newGraphQLClient = func(endpoint string, opts ...graphql.ClientOption) *graphql.Client {
		return mockClient
	}

	// Test data
	zoneIDs := []string{"zone1"}
	domainNames := []string{"example.com", "test.com"}
	requestPath := "/api/%"

	// Set up test configuration
	cfgCfAPIToken = "test-token"
	cfgScrapeDelay = 300

	// Call the function
	resp, err := fetchZoneTotals(zoneIDs, domainNames, requestPath)
	if err != nil {
		t.Fatalf("fetchZoneTotals failed: %v", err)
	}

	// Verify the response
	if len(resp.Viewer.Zones) != 1 {
		t.Errorf("Expected 1 zone in response, got %d", len(resp.Viewer.Zones))
	}
	if resp.Viewer.Zones[0].ZoneTag != "zone1" {
		t.Errorf("Expected zone tag 'zone1', got %s", resp.Viewer.Zones[0].ZoneTag)
	}
}

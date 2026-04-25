package main

import (
	"fmt"
	"os"
	"testing"
)

func TestGetHost(t *testing.T) {
	tests := []struct {
		name         string
		resolvedFQDN string
		expectedHost string
		expectErr    bool
	}{
		{
			name:         "root challenge",
			resolvedFQDN: "_acme-challenge.example.com.",
			expectedHost: "_acme-challenge.example.com",
			expectErr:    false,
		},
		{
			name:         "delegated challenge",
			resolvedFQDN: "_acme-challenge.foo.example.com.",
			expectedHost: "_acme-challenge.foo.example.com",
			expectErr:    false,
		},
		{
			name:         "missing trailing dot",
			resolvedFQDN: "_acme-challenge.example.com",
			expectedHost: "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, err := getHost(tt.resolvedFQDN)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if host != tt.expectedHost {
				t.Fatalf("expected host %q, got %q", tt.expectedHost, host)
			}
		})
	}
}

func TestTXTRecordManagementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiKey := os.Getenv("BUNNY_API_KEY")
	if apiKey == "" {
		t.Skip("BUNNY_API_KEY not set")
	}

	zoneID := os.Getenv("BUNNY_ZONE_ID")
	if zoneID == "" {
		t.Skip("BUNNY_ZONE_ID not set")
	}

	cfg := &bunnyClientConfig{
		apiKey: apiKey,
		zoneID: mustParseZoneID(t, zoneID),
	}

	domain := os.Getenv("BUNNY_TEST_FQDN")
	if domain == "" {
		t.Skip("BUNNY_TEST_FQDN not set")
	}

	txtValue := "cert-manager-webhook-test-value"

	t.Run("Add TXT Record", func(t *testing.T) {
		err := addTxtRecord(cfg, domain, txtValue)
		if err != nil {
			t.Fatalf("Failed to add TXT record: %v", err)
		}
	})

	t.Run("Delete TXT Record", func(t *testing.T) {
		err := deleteTxtRecord(cfg, domain, txtValue)
		if err != nil {
			t.Fatalf("Failed to delete TXT record: %v", err)
		}
	})
}

func mustParseZoneID(t *testing.T, value string) int {
	t.Helper()

	var zoneID int
	_, err := fmt.Sscanf(value, "%d", &zoneID)
	if err != nil {
		t.Fatalf("invalid BUNNY_ZONE_ID %q: %v", value, err)
	}
	return zoneID
}

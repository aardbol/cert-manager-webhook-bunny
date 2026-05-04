package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestGetHostFromZone(t *testing.T) {
	tests := []struct {
		name         string
		resolvedFQDN string
		zoneName     string
		expectedHost string
		expectErr    bool
	}{
		{
			name:         "root challenge",
			resolvedFQDN: "_acme-challenge.example.com.",
			zoneName:     "example.com",
			expectedHost: "_acme-challenge",
			expectErr:    false,
		},
		{
			name:         "delegated challenge",
			resolvedFQDN: "_acme-challenge.foo.example.com.",
			zoneName:     "example.com",
			expectedHost: "_acme-challenge.foo",
			expectErr:    false,
		},
		{
			name:         "nested zone",
			resolvedFQDN: "_acme-challenge.archive.mainnet.qfnode.net.",
			zoneName:     "qfnode.net",
			expectedHost: "_acme-challenge.archive.mainnet",
			expectErr:    false,
		},
		{
			name:         "missing trailing dot is accepted",
			resolvedFQDN: "_acme-challenge.example.com",
			zoneName:     "example.com",
			expectedHost: "_acme-challenge",
			expectErr:    false,
		},
		{
			name:         "outside zone",
			resolvedFQDN: "_acme-challenge.example.org.",
			zoneName:     "example.com",
			expectedHost: "",
			expectErr:    true,
		},
		{
			name:         "zone apex",
			resolvedFQDN: "example.com.",
			zoneName:     "example.com",
			expectedHost: "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			host, err := getHostFromZone(tt.resolvedFQDN, tt.zoneName)
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

// TestTXTRecordManagementIntegration tests the full Present/CleanUp flow against Bunny DNS.
// Run with BUNNY_API_KEY and BUNNY_TEST_FQDN environment variables set.
func TestTXTRecordManagementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiKey := os.Getenv("BUNNY_API_KEY")
	if apiKey == "" {
		t.Skip("BUNNY_API_KEY not set")
	}

	fqdn := os.Getenv("BUNNY_TEST_FQDN")
	if fqdn == "" {
		t.Skip("BUNNY_TEST_FQDN not set")
	}

	client := newBunnyClient(apiKey)

	txtValue := "cert-manager-webhook-test-value-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Test Present flow: resolve zone, check for duplicate, add record.
	t.Run("Present", func(t *testing.T) {
		zone, host, err := client.resolveZone(fqdn)
		if err != nil {
			t.Fatalf("failed to resolve zone for %q: %v", fqdn, err)
		}

		// Check for existing record (should not exist).
		for _, r := range zone.Records {
			if r.Type == RecordTypeTXT && r.Name == host && r.Value == txtValue {
				t.Fatalf("TXT record unexpectedly already exists for %q", host)
			}
		}

		// Add the record.
		if err := client.addTxtRecord(zone.Id, host, txtValue); err != nil {
			t.Fatalf("failed to add TXT record: %v", err)
		}

		// Verify it was added by re-fetching the zone.
		zone, host, err = client.resolveZone(fqdn)
		if err != nil {
			t.Fatalf("failed to re-resolve zone after add: %v", err)
		}
		found := false
		for _, r := range zone.Records {
			if r.Type == RecordTypeTXT && r.Name == host && r.Value == txtValue {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("TXT record was not found after add for %q", host)
		}
	})

	// Test CleanUp flow: resolve zone, delete matching records.
	t.Run("CleanUp", func(t *testing.T) {
		zone, host, err := client.resolveZone(fqdn)
		if err != nil {
			t.Fatalf("failed to resolve zone for cleanup: %v", err)
		}

		deleted, err := client.deleteTxtRecord(zone.Id, zone.Records, host, txtValue)
		if err != nil {
			t.Fatalf("failed to delete TXT record: %v", err)
		}
		if deleted == 0 {
			t.Logf("no matching TXT record found for cleanup (this is OK)")
		} else {
			t.Logf("deleted %d TXT record(s)", deleted)
		}
	})
}

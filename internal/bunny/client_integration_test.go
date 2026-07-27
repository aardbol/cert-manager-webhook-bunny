//go:build integration

package bunny

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

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

	client := NewClient(apiKey)

	txtValue := "cert-manager-webhook-test-value-" + fmt.Sprintf("%d", time.Now().UnixNano())

	t.Run("Present", func(t *testing.T) {
		zone, host, err := client.ResolveZone(context.Background(), fqdn)
		if err != nil {
			t.Fatalf("failed to resolve zone for %q: %v", fqdn, err)
		}

		for _, r := range zone.Records {
			if r.Type == RecordTypeTXT && r.Name == host && r.Value == txtValue {
				t.Fatalf("TXT record unexpectedly already exists for %q", host)
			}
		}

		if err := client.AddTXTRecord(context.Background(), zone.ID, host, txtValue); err != nil {
			t.Fatalf("failed to add TXT record: %v", err)
		}

		zone, host, err = client.ResolveZone(context.Background(), fqdn)
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

	t.Run("CleanUp", func(t *testing.T) {
		zone, host, err := client.ResolveZone(context.Background(), fqdn)
		if err != nil {
			t.Fatalf("failed to resolve zone for cleanup: %v", err)
		}

		deleted, err := client.DeleteTXTRecord(context.Background(), zone.ID, zone.Records, host, txtValue)
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

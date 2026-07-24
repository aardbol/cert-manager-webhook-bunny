package bunny

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestResolveZone(t *testing.T) {
	tests := []struct {
		name     string
		fqdn     string
		wantZone string
		wantHost string
		wantErr  string
	}{
		{
			name:     "success",
			fqdn:     "_acme-challenge.example.com.",
			wantZone: "example.com",
			wantHost: "_acme-challenge",
		},
		{
			name:    "not found",
			fqdn:    "_acme-challenge.unknown.com.",
			wantErr: "could not dynamically find",
		},
		{
			name:    "FQDN too short",
			fqdn:    "_acme-challenge.nope.",
			wantErr: "too short to determine a zone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Query().Get("search") {
				case "example.com":
					w.Write([]byte(`{"Items":[{"Id":1,"Domain":"example.com"}]}`))
				default:
					w.Write([]byte(`{"Items":[]}`))
				}
			}))
			defer server.Close()

			client := &Client{
				apiKey:     "test-key",
				httpClient: &http.Client{},
				baseURL:    server.URL + "/dnszone",
			}
			zone, host, err := client.ResolveZone(context.Background(), tt.fqdn)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if zone.Domain != tt.wantZone {
				t.Fatalf("expected domain %q, got %q", tt.wantZone, zone.Domain)
			}
			if host != tt.wantHost {
				t.Fatalf("expected host %q, got %q", tt.wantHost, host)
			}
		})
	}
}

func TestAddTXTRecord(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    string
	}{
		{name: "success", statusCode: http.StatusNoContent},
		{name: "API error", statusCode: http.StatusBadRequest, wantErr: "API error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("expected PUT, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := &Client{
				apiKey:     "test-key",
				httpClient: &http.Client{},
				baseURL:    server.URL + "/dnszone",
			}
			err := client.AddTXTRecord(context.Background(), 1, "test-host", "test-val")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteTXTRecord(t *testing.T) {
	tests := []struct {
		name    string
		records []Record
		key     string
		host    string
		wantCnt int
		wantErr string
	}{
		{
			name: "success",
			records: []Record{
				{ID: 1, Type: RecordTypeTXT, Value: "val", Name: "host"},
			},
			key:     "val",
			host:    "host",
			wantCnt: 1,
		},
		{
			name: "no match",
			records: []Record{
				{ID: 1, Type: RecordTypeTXT, Value: "other", Name: "host"},
			},
			key:     "val",
			host:    "host",
			wantCnt: 0,
		},
		{
			name: "partial failure",
			records: []Record{
				{ID: 1, Type: RecordTypeTXT, Value: "val", Name: "host"},
				{ID: 999, Type: RecordTypeTXT, Value: "val", Name: "host"},
			},
			key:     "val",
			host:    "host",
			wantCnt: 1,
			wantErr: "failed to delete 1 record(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/999") {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client := &Client{
				apiKey:     "test-key",
				httpClient: &http.Client{},
				baseURL:    server.URL + "/dnszone",
			}
			got, err := client.DeleteTXTRecord(context.Background(), 1, tt.records, tt.host, tt.key)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantCnt {
				t.Fatalf("expected %d deleted, got %d", tt.wantCnt, got)
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

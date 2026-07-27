package bunny

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestDoRequestSendsHeaders(t *testing.T) {
	t.Parallel()

	var gotAccessKey, gotContentType, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccessKey = r.Header.Get("AccessKey")
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := &Client{apiKey: "test-key", httpClient: &http.Client{}}
	body, err := client.doRequest(context.Background(), server.URL, "GET", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", body)
	}
	if gotAccessKey != "test-key" {
		t.Errorf("expected AccessKey header %q, got %q", "test-key", gotAccessKey)
	}
	if gotContentType != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", gotContentType)
	}
	if gotAccept != "application/json" {
		t.Errorf("expected Accept %q, got %q", "application/json", gotAccept)
	}
}

func TestDoRequestTruncatesErrorBody(t *testing.T) {
	t.Parallel()

	longBody := strings.Repeat("x", 500)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(longBody))
	}))
	defer server.Close()

	client := &Client{apiKey: "test-key", httpClient: &http.Client{}}
	_, err := client.doRequest(context.Background(), server.URL, "GET", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "... (truncated)") {
		t.Fatalf("expected truncated error body, got: %v", err)
	}
	if strings.Contains(err.Error(), longBody) {
		t.Fatalf("error should not contain full long body, got: %v", err)
	}
}

func TestDoRequestContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &Client{apiKey: "test-key", httpClient: &http.Client{}}
	_, err := client.doRequest(ctx, "http://localhost:0/never", "GET", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestResolveZoneDelegatedZone(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("search") {
		case "example.com":
			w.Write([]byte(`{"Items":[]}`))
		case "foo.example.com":
			w.Write([]byte(`{"Items":[{"Id":5,"Domain":"foo.example.com"}]}`))
		default:
			w.Write([]byte(`{"Items":[]}`))
		}
	}))
	defer server.Close()

	client := &Client{apiKey: "test-key", httpClient: &http.Client{}, baseURL: server.URL + "/dnszone"}

	zone, host, err := client.ResolveZone(context.Background(), "_acme-challenge.foo.example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.Domain != "foo.example.com" {
		t.Fatalf("expected zone domain %q, got %q", "foo.example.com", zone.Domain)
	}
	if host != "_acme-challenge" {
		t.Fatalf("expected host %q, got %q", "_acme-challenge", host)
	}
}

func TestResolveZoneSelectsMatchingItem(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Items":[{"Id":1,"Domain":"other.com"},{"Id":2,"Domain":"example.com"}]}`))
	}))
	defer server.Close()

	client := &Client{apiKey: "test-key", httpClient: &http.Client{}, baseURL: server.URL + "/dnszone"}

	zone, _, err := client.ResolveZone(context.Background(), "_acme-challenge.example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.ID != 2 {
		t.Fatalf("expected to select item with ID 2, got %d", zone.ID)
	}
}

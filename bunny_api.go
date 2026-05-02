package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/aardbol/cert-manager-webhook-bunny/internal"
)

const (
	ApiUrl          = "https://api.bunny.io/dnszone/"
	RecordsEndpoint = "/records"
	RecordTypeTXT   = 3
)

// bunnyClientConfig contains the parameters required to interact with Bunny API, should be located in a Secret
type bunnyClientConfig struct {
	apiKey string
	zoneID int
}

// addTxtRecord creates a TXT record in the Bunny zone. The host argument
// must already be the relative name within the zone.
func addTxtRecord(cfg *bunnyClientConfig, host string, key string) error {
	payload := internal.CreateRecordRequest{Type: RecordTypeTXT, Ttl: 120, Value: key, Name: host}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal record payload: %w", err)
	}

	_, err = callDnsApi(RecordsEndpoint, "PUT", bytes.NewReader(data), cfg)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	return nil
}

// deleteTxtRecord removes all matching TXT records from the provided slice.
// It returns the number of records successfully deleted. If one or more deletions fail, it still attempts the rest and returns an aggregate error.
func deleteTxtRecord(cfg *bunnyClientConfig, records []internal.Record, host string, key string) (int, error) {
	deleted := 0
	var failed []int

	for _, record := range records {
		if record.Value == key && record.Type == 3 && record.Name == host {
			suffix := RecordsEndpoint + "/" + fmt.Sprintf("%d", record.Id)
			if _, err := callDnsApi(suffix, "DELETE", nil, cfg); err != nil {
				klog.Warningf("failed to delete record %d: %v", record.Id, err)
				failed = append(failed, record.Id)
				continue
			}
			deleted++
		}
	}

	if len(failed) > 0 {
		return deleted, fmt.Errorf("failed to delete %d record(s) with ID(s) %v", len(failed), failed)
	}
	return deleted, nil
}

// getHostFromZone derives the relative DNS host name from a fully-qualified
// domain name and a zone domain. It normalizes both inputs, validates that
// the FQDN is a descendant of the zone (not the apex), and returns the host
// portion without the zone suffix.
func getHostFromZone(resolvedFqdn string, zoneName string) (string, error) {
	fqdn := strings.TrimSuffix(strings.TrimSpace(strings.ToLower(resolvedFqdn)), ".")
	if fqdn == "" {
		return "", fmt.Errorf("unable to parse host out of resolved FQDN ('%s')", resolvedFqdn)
	}

	zoneName = strings.TrimSuffix(strings.TrimSpace(strings.ToLower(zoneName)), ".")
	if zoneName == "" {
		return "", fmt.Errorf("zone domain is empty")
	}

	if fqdn == zoneName {
		return "", fmt.Errorf("resolved FQDN ('%s') points to zone apex, expected challenge record below zone", resolvedFqdn)
	}

	suffix := "." + zoneName
	if !strings.HasSuffix(fqdn, suffix) {
		return "", fmt.Errorf("resolved FQDN ('%s') is not within zone '%s'", resolvedFqdn, zoneName)
	}

	host := strings.TrimSuffix(fqdn, suffix)
	if host == "" {
		return "", fmt.Errorf("unable to derive relative host from resolved FQDN ('%s')", resolvedFqdn)
	}

	return host, nil
}

// getZone fetches the full Bunny DNS zone object for the configured zoneID.
// The returned Zone contains both the authoritative Domain name and the
// current list of Records, allowing callers to derive the relative host and
// check for existing TXT records with a single API call.
func getZone(cfg *bunnyClientConfig) (*internal.Zone, error) {
	body, err := callDnsApi("", "GET", nil, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to request zone: %w", err)
	}
	zone := &internal.Zone{}
	if err := json.Unmarshal(body, zone); err != nil {
		return nil, fmt.Errorf("unable to unmarshal zone response: %w", err)
	}
	return zone, nil
}

// callDnsApi executes a Bunny DNS API request.
func callDnsApi(urlSuffix, method string, body io.Reader, cfg *bunnyClientConfig) ([]byte, error) {
	ctx := context.Background()
	url := ApiUrl + fmt.Sprintf("%d", cfg.zoneID) + urlSuffix
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("unable to build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("AccessKey", cfg.apiKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			klog.Warningf("failed to close response body for %s %s: %v", method, urlSuffix, cerr)
		}
	}()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("unable to read response body: %w", readErr)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}

	return nil, fmt.Errorf("API error: status=%s, url=%s, method=%s, body=%s",
		resp.Status, urlSuffix, method, string(respBody))
}

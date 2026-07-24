package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/aardbol/cert-manager-webhook-bunny/internal"
)

const (
	dnsAPIURL       = "https://api.bunny.net/dnszone"
	recordsEndpoint = "records"
)

type bunnyClient struct {
	apiKey     string
	httpClient *http.Client
}

func newBunnyClient(apiKey string) *bunnyClient {
	return &bunnyClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *bunnyClient) addTxtRecord(zoneID int, host string, key string) error {
	payload := internal.CreateRecordRequest{Type: internal.RecordTypeTXT, TTL: 120, Value: key, Name: host}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal record payload: %w", err)
	}

	reqURL, err := url.JoinPath(dnsAPIURL, fmt.Sprint(zoneID), recordsEndpoint)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	_, err = c.doRequest(reqURL, "PUT", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	return nil
}

func (c *bunnyClient) deleteTxtRecord(zoneID int, records []internal.Record, host string, key string) (int, error) {
	deleted := 0
	var failed []int

	for _, record := range records {
		if record.Value == key && record.Type == internal.RecordTypeTXT && record.Name == host {
			reqURL, err := url.JoinPath(dnsAPIURL, fmt.Sprint(zoneID), recordsEndpoint, fmt.Sprint(record.ID))
			if err != nil {
				klog.Warningf("failed to build URL for record %d: %v", record.ID, err)
				failed = append(failed, record.ID)
				continue
			}

			if _, err := c.doRequest(reqURL, "DELETE", nil); err != nil {
				klog.Warningf("failed to delete record %d: %v", record.ID, err)
				failed = append(failed, record.ID)
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

func (c *bunnyClient) resolveZone(fqdn string) (*internal.Zone, string, error) {
	challengeDomain := strings.TrimSuffix(strings.TrimSpace(strings.ToLower(fqdn)), ".")
	challengeDomain = strings.TrimPrefix(challengeDomain, "_acme-challenge.")

	parts := strings.Split(challengeDomain, ".")
	if len(parts) < 2 {
		return nil, "", fmt.Errorf("FQDN '%s' is too short to determine a zone", fqdn)
	}

	for i := len(parts) - 2; i >= 0; i-- {
		searchDomain := strings.Join(parts[i:], ".")

		reqURL := fmt.Sprintf("%s?search=%s", dnsAPIURL, url.QueryEscape(searchDomain))
		body, err := c.doRequest(reqURL, "GET", nil)
		if err != nil {
			return nil, "", fmt.Errorf("failed to search for zone '%s': %w", searchDomain, err)
		}

		var list internal.ZoneList
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, "", fmt.Errorf("unable to unmarshal zone search response: %w", err)
		}

		for _, z := range list.Items {
			if strings.ToLower(z.Domain) == searchDomain {
				host, err := getHostFromZone(fqdn, z.Domain)
				if err != nil {
					return nil, "", fmt.Errorf("failed to derive host: %w", err)
				}
				return &z, host, nil
			}
		}
	}
	return nil, "", fmt.Errorf("could not dynamically find a matching zone for FQDN '%s'", fqdn)
}

func (c *bunnyClient) doRequest(reqURL, method string, body io.Reader) ([]byte, error) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("unable to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("AccessKey", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			klog.Warningf("failed to close response body for %s %s: %v", method, reqURL, cerr)
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
		resp.Status, reqURL, method, string(respBody))
}

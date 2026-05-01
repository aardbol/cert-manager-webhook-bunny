package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/aardbol/cert-manager-webhook-bunny/internal"
)

// bunnyClientConfig contains the parameters required to interact with Bunny API, should be located in a Secret
type bunnyClientConfig struct {
	apiKey string
	zoneID int
}

func addTxtRecord(cfg *bunnyClientConfig, resolvedFqdn string, key string) error {
	host, err := getHost(cfg, resolvedFqdn)
	if err != nil {
		return err
	}

	urlOfRecords := "https://api.bunny.net/dnszone/" + fmt.Sprintf("%d", cfg.zoneID) + "/records"
	payload := strings.NewReader("{\"Type\":3,\"Ttl\":120,\"Value\":\"" + key + "\",\"Name\":\"" + host + "\"}")

	putResBody, putResErr := callDnsApi(urlOfRecords, "PUT", payload, cfg)
	if putResErr != nil {
		return fmt.Errorf("Failed to create record: %v", putResErr)
	}

	record := internal.Record{}
	recordReadErr := json.Unmarshal(putResBody, &record)
	if recordReadErr != nil {
		return fmt.Errorf("Unable to unmarshal response: %v", recordReadErr)
	}
	return nil
}

func deleteTxtRecord(cfg *bunnyClientConfig, resolvedFqdn string, key string) error {
	host, err := getHost(cfg, resolvedFqdn)
	if err != nil {
		return err
	}

	records, err := getRecords(cfg)
	if err != nil {
		return err
	}

	for _, record := range records {
		if record.Value == key && record.Type == 3 && record.Name == host { // Type 3 is TXT record
			urlOfRecords := "https://api.bunny.net/dnszone/" + fmt.Sprintf("%d", cfg.zoneID) + "/records/" + fmt.Sprintf("%d", record.Id)
			_, deleteResErr := callDnsApi(urlOfRecords, "DELETE", nil, cfg)
			if deleteResErr != nil {
				return fmt.Errorf("Failed to delete record: %v", deleteResErr)
			}
			break
		}
	}

	return nil
}

func getHost(cfg *bunnyClientConfig, resolvedFqdn string) (string, error) {
	zoneName, err := getZoneDomain(cfg)
	if err != nil {
		return "", err
	}

	return getHostFromZone(resolvedFqdn, zoneName)
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

func getZoneDomain(cfg *bunnyClientConfig) (string, error) {
	urlOfZone := "https://api.bunny.net/dnszone/" + fmt.Sprintf("%d", cfg.zoneID)

	getResBody, getResErr := callDnsApi(urlOfZone, "GET", nil, cfg)
	if getResErr != nil {
		return "", fmt.Errorf("Failed to request zone: %v", getResErr)
	}

	zone := internal.Zone{}
	readErr := json.Unmarshal(getResBody, &zone)
	if readErr != nil {
		return "", fmt.Errorf("Unable to unmarshal response: %v", readErr)
	}

	return zone.Domain, nil
}

func getRecords(cfg *bunnyClientConfig) ([]internal.Record, error) {
	urlOfRecords := "https://api.bunny.net/dnszone/" + fmt.Sprintf("%d", cfg.zoneID)

	getResBody, getResErr := callDnsApi(urlOfRecords, "GET", nil, cfg)
	if getResErr != nil {
		return nil, fmt.Errorf("Failed to request records: %v", getResErr)
	}

	zone := internal.Zone{}
	readErr := json.Unmarshal(getResBody, &zone)
	if readErr != nil {
		return nil, fmt.Errorf("Unable to unmarshal response: %v", readErr)
	}

	return zone.Records, nil
}

func callDnsApi(url, method string, body io.Reader, cfg *bunnyClientConfig) ([]byte, error) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to execute request %v", err)
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("AccessKey", cfg.apiKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			klog.Fatal(err)
		}
	}()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("unable to read response body: %v", readErr)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		return respBody, nil
	}

	text := "Error calling API status:" + resp.Status + " url: " + url + " method: " + method
	klog.Error(text)
	return nil, errors.New(text)
}

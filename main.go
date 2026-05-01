package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/aardbol/cert-manager-webhook-bunny/internal"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&bunnyDNSProviderSolver{},
	)
}

// These are the things required to interact with Bunny API, should be located
// in secret, referenced in config by it's name
type bunnyClientConfig struct {
	apiKey string
	zoneID int
}

type bunnyDNSProviderSolver struct {
	client *kubernetes.Clientset
}

type bunnyDNSProviderConfig struct {
	// name of the secret which contains Bunny credentials
	SecretRef string `json:"secretRef"`
	// optional namespace for the secret
	SecretNamespace string `json:"secretNamespace"`
	// Bunny DNS zone ID
	ZoneID int `json:"zoneId"`
}

func (n *bunnyDNSProviderSolver) Name() string {
	return "bunny"
}

func (n *bunnyDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := n.getConfig(ch)
	if err != nil {
		return err
	}
	if err := addTxtRecord(cfg, ch.ResolvedFQDN, ch.Key); err != nil {
		return err
	}
	klog.Infof("successfully presented challenge for domain '%s'", ch.DNSName)
	return nil
}

func (n *bunnyDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := n.getConfig(ch)
	if err != nil {
		return err
	}
	if err := deleteTxtRecord(cfg, ch.ResolvedFQDN, ch.Key); err != nil {
		return err
	}
	klog.Infof("successfully cleaned up challenge for domain '%s'", ch.DNSName)
	return nil
}

func (n *bunnyDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	n.client = cl
	return nil
}

func (n *bunnyDNSProviderSolver) getConfig(ch *v1alpha1.ChallengeRequest) (*bunnyClientConfig, error) {
	var secretNs string
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	if cfg.ZoneID <= 0 {
		return nil, fmt.Errorf("zoneId must be specified and greater than 0")
	}

	bunnyCfg := &bunnyClientConfig{
		zoneID: cfg.ZoneID,
	}

	if cfg.SecretNamespace != "" {
		secretNs = cfg.SecretNamespace
	} else {
		secretNs = ch.ResourceNamespace
	}

	sec, err := n.client.CoreV1().Secrets(secretNs).Get(context.TODO(), cfg.SecretRef, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get secret '%s/%s': %v", secretNs, cfg.SecretRef, err)
	}

	bunnyCfg.apiKey, err = stringFromSecretData(&sec.Data, "api-key")
	if err != nil {
		return nil, fmt.Errorf("unable to get 'api-key' from secret '%s/%s': %v", secretNs, cfg.SecretRef, err)
	}

	return bunnyCfg, nil
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

func stringFromSecretData(secretData *map[string][]byte, key string) (string, error) {
	data, ok := (*secretData)[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret data", key)
	}
	return string(data), nil
}

func loadConfig(cfgJSON *extapi.JSON) (bunnyDNSProviderConfig, error) {
	cfg := bunnyDNSProviderConfig{}

	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}
	return cfg, nil
}

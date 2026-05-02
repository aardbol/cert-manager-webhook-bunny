package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

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

type bunnyDNSProviderSolver struct {
	client *kubernetes.Clientset
}

// bunnyDNSProviderConfig is the user-facing configuration for this webhook solver,
// typically provided in the cert-manager Issuer or ClusterIssuer resource.
type bunnyDNSProviderConfig struct {
	// SecretRef is the name of the secret which contains Bunny credentials
	SecretRef string `json:"secretRef"`
	// SecretNamespace contains an optional namespace for the secret
	SecretNamespace string `json:"secretNamespace"`
	// SecretKey contains an optional name of the secret, defaults to "api-key"
	SecretKey string `json:"secretKey"`
	// ZoneID contains the Bunny DNS zone ID
	ZoneID int `json:"zoneId"`
}

func (n *bunnyDNSProviderSolver) Name() string {
	return "bunny"
}

// Present creates the ACME DNS-01 TXT record if it does not already exist.
// It is idempotent: if a matching TXT record is found in the zone, it returns
// immediately without issuing another Bunny API write.
func (n *bunnyDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := n.getConfig(ch)
	if err != nil {
		return err
	}

	zone, err := getZone(cfg)
	if err != nil {
		return err
	}

	host, err := getHostFromZone(ch.ResolvedFQDN, zone.Domain)
	if err != nil {
		return err
	}

	for _, r := range zone.Records {
		if r.Type == 3 && r.Name == host && r.Value == ch.Key {
			klog.Infof("TXT record already exists for domain '%s', skipping creation", ch.DNSName)
			return nil
		}
	}

	if err := addTxtRecord(cfg, host, ch.Key); err != nil {
		return err
	}
	klog.Infof("successfully presented challenge for domain '%s'", ch.DNSName)
	return nil
}

// CleanUp removes every Bunny TXT record that matches the challenge key,
// relative host name, and record type. If no matching record exists, it
// logs and returns successfully so that retries or stale cleanups do not fail.
func (n *bunnyDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := n.getConfig(ch)
	if err != nil {
		return err
	}

	zone, err := getZone(cfg)
	if err != nil {
		return err
	}

	host, err := getHostFromZone(ch.ResolvedFQDN, zone.Domain)
	if err != nil {
		return err
	}

	deleted, err := deleteTxtRecord(cfg, zone.Records, host, ch.Key)
	if err != nil {
		return fmt.Errorf("cleanup incomplete (%d record(s) already deleted): %w", deleted, err)
	}
	if deleted > 0 {
		klog.Infof("successfully cleaned up challenge for domain '%s' (%d record(s) removed)", ch.DNSName, deleted)
	} else {
		klog.Infof("no matching TXT record found for domain '%s', cleanup skipped", ch.DNSName)
	}
	return nil
}

func (n *bunnyDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, _ <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	n.client = cl
	return nil
}

// getConfig builds the runtime Bunny client configuration from the challenge request.
// It validates that zoneId and secretRef are present, resolves the secret namespace,
// and extracts the API key from the referenced Kubernetes secret.
func (n *bunnyDNSProviderSolver) getConfig(ch *v1alpha1.ChallengeRequest) (*bunnyClientConfig, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	if cfg.ZoneID <= 0 {
		return nil, fmt.Errorf("zoneId must be specified and greater than 0")
	}

	if cfg.SecretRef == "" {
		return nil, fmt.Errorf("secretRef must be specified")
	}

	secretNs := cfg.SecretNamespace
	if secretNs == "" {
		secretNs = ch.ResourceNamespace
	}

	key := cfg.SecretKey
	if key == "" {
		key = "api-key"
	}

	ctx := context.Background()
	sec, err := n.client.CoreV1().Secrets(secretNs).Get(ctx, cfg.SecretRef, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get secret %q/%q: %w", secretNs, cfg.SecretRef, err)
	}

	apiKey, err := stringFromSecretData(sec.Data, key)
	if err != nil {
		return nil, fmt.Errorf("unable to get key %q from secret %q/%q: %w", key, secretNs, cfg.SecretRef, err)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("key %q in secret %q/%q is empty", key, secretNs, cfg.SecretRef)
	}

	return &bunnyClientConfig{
		zoneID: cfg.ZoneID,
		apiKey: apiKey,
	}, nil
}

// stringFromSecretData extracts a string value from a Kubernetes secret data map by key.
// It returns an error if the key is absent.
func stringFromSecretData(secretData map[string][]byte, key string) (string, error) {
	data, ok := secretData[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret data", key)
	}
	return string(data), nil
}

// loadConfig unmarshals the webhook solver configuration from the JSON blob provided by cert-manager.
// A nil configJSON returns a zero-value config.
func loadConfig(cfgJSON *extapi.JSON) (bunnyDNSProviderConfig, error) {
	cfg := bunnyDNSProviderConfig{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %w", err)
	}
	return cfg, nil
}

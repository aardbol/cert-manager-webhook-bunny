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

	"github.com/aardbol/cert-manager-webhook-bunny/internal/bunny"
)

const defaultSecretKey = "api-key"

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

type bunnyDNSProviderConfig struct {
	SecretRef       string `json:"secretRef"`
	SecretNamespace string `json:"secretNamespace"`
	SecretKey       string `json:"secretKey"`
}

func (n *bunnyDNSProviderSolver) Name() string {
	return "bunny"
}

func (n *bunnyDNSProviderSolver) Present(cr *v1alpha1.ChallengeRequest) error {
	bunnyClient, err := n.getClient(cr)
	if err != nil {
		return err
	}

	zone, host, err := bunnyClient.ResolveZone(context.Background(), cr.ResolvedFQDN)
	if err != nil {
		return err
	}

	for _, r := range zone.Records {
		if r.Type == bunny.RecordTypeTXT && r.Name == host && r.Value == cr.Key {
			klog.Infof("TXT record already exists for domain '%s', skipping creation", cr.DNSName)
			return nil
		}
	}

	if err := bunnyClient.AddTXTRecord(context.Background(), zone.ID, host, cr.Key); err != nil {
		return err
	}
	klog.Infof("successfully presented challenge for domain '%s'", cr.DNSName)
	return nil
}

func (n *bunnyDNSProviderSolver) CleanUp(cr *v1alpha1.ChallengeRequest) error {
	bunnyClient, err := n.getClient(cr)
	if err != nil {
		return err
	}

	zone, host, err := bunnyClient.ResolveZone(context.Background(), cr.ResolvedFQDN)
	if err != nil {
		return err
	}

	deleted, err := bunnyClient.DeleteTXTRecord(context.Background(), zone.ID, zone.Records, host, cr.Key)
	if err != nil {
		return fmt.Errorf("cleanup incomplete (%d record(s) already deleted): %w", deleted, err)
	}
	if deleted > 0 {
		klog.Infof("successfully cleaned up challenge for domain '%s' (%d record(s) removed)", cr.DNSName, deleted)
	} else {
		klog.Infof("no matching TXT record found for domain '%s', cleanup skipped", cr.DNSName)
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

func (n *bunnyDNSProviderSolver) getClient(cr *v1alpha1.ChallengeRequest) (*bunny.Client, error) {
	cfg, err := loadConfig(cr.Config)
	if err != nil {
		return nil, err
	}

	if cfg.SecretRef == "" {
		return nil, fmt.Errorf("secretRef must be specified")
	}

	secretNs := cfg.SecretNamespace
	if secretNs == "" {
		secretNs = cr.ResourceNamespace
	}

	key := cfg.SecretKey
	if key == "" {
		key = defaultSecretKey
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
	return bunny.NewClient(apiKey), nil
}

func stringFromSecretData(secretData map[string][]byte, key string) (string, error) {
	data, ok := secretData[key]
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
		return cfg, fmt.Errorf("error decoding solver config: %w", err)
	}
	return cfg, nil
}

package main

import (
	"testing"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    bunnyDNSProviderConfig
		wantErr bool
	}{
		{
			name: "nil config returns empty",
			raw:  "",
			want: bunnyDNSProviderConfig{},
		},
		{
			name: "valid config",
			raw:  `{"secretRef":"my-secret","secretNamespace":"ns","secretKey":"key"}`,
			want: bunnyDNSProviderConfig{SecretRef: "my-secret", SecretNamespace: "ns", SecretKey: "key"},
		},
		{
			name:    "invalid json",
			raw:     `not-json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var cfgJSON *extapi.JSON
			if tt.raw != "" {
				cfgJSON = &extapi.JSON{Raw: []byte(tt.raw)}
			}
			got, err := loadConfig(cfgJSON)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestStringFromSecretData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    map[string][]byte
		key     string
		want    string
		wantErr bool
	}{
		{
			name: "key exists",
			data: map[string][]byte{defaultSecretKey: []byte("secret")},
			key:  defaultSecretKey,
			want: "secret",
		},
		{
			name:    "key missing",
			data:    map[string][]byte{"other": []byte("x")},
			key:     defaultSecretKey,
			wantErr: true,
		},
		{
			name: "empty value is returned without error",
			data: map[string][]byte{defaultSecretKey: []byte("")},
			key:  defaultSecretKey,
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := stringFromSecretData(tt.data, tt.key)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

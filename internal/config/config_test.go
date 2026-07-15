package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestConfigLoadsValuesFromFile(t *testing.T) {
	for name, value := range map[string]string{
		"TENCENTCLOUD_SECRET_ID":  "env-secret-id",
		"TENCENTCLOUD_SECRET_KEY": "env-secret-key",
		"TENCENTCLOUD_REGION":     "env-region",
		"TENCENTCLOUD_ENDPOINT":   "env.tencentcloud.example.com",
		"IMAGE_SYNC_ENDPOINT":     "https://env.example.com/sync",
		"REGISTRY_USERNAME":       "env-registry-user",
		"REGISTRY_PASSWORD":       "env-registry-password",
	} {
		t.Setenv(name, value)
	}

	const data = `Name: cube_sandbox_image_server-api
Host: 127.0.0.1
Port: 43999
Timeout: 30000

TencentCloud:
  SecretID: file-secret-id
  SecretKey: file-secret-key
  Region: file-region
  Endpoint: file.tencentcloud.example.com
  RequestTimeout: 10s
  StatusPollInterval: 5s
  StatusPollTimeout: 20s

ImageSync:
  Endpoint: https://file.example.com/sync
  RequestTimeout: 20s
  AttemptTimeout: 1s
  MaxResponseBytes: 1048576
  MaxAttempts: 3
  RetryInterval: 500ms

Registry:
  Username: file-registry-user
  Password: file-registry-password
  AllowedHosts:
    - registry.example.com
  RequestTimeout: 10s
`
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configFile, []byte(data), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var config Config
	if err := conf.Load(configFile, &config); err != nil {
		t.Fatalf("conf.Load() error = %v", err)
	}
	if config.TencentCloud.SecretID != "file-secret-id" || config.TencentCloud.SecretKey != "file-secret-key" {
		t.Fatalf("TencentCloud credentials were not loaded from file: %#v", config.TencentCloud)
	}
	if config.TencentCloud.Region != "file-region" || config.TencentCloud.Endpoint != "file.tencentcloud.example.com" {
		t.Fatalf("TencentCloud settings were not loaded from file: %#v", config.TencentCloud)
	}
	if config.Timeout != 30000 {
		t.Fatalf("REST Timeout = %d", config.Timeout)
	}
	if config.TencentCloud.RequestTimeout != 10*time.Second ||
		config.TencentCloud.StatusPollInterval != 5*time.Second ||
		config.TencentCloud.StatusPollTimeout != 20*time.Second {
		t.Fatalf("unexpected TencentCloud timeouts: %#v", config.TencentCloud)
	}
	if config.ImageSync.Endpoint != "https://file.example.com/sync" || config.ImageSync.MaxAttempts != 3 || config.ImageSync.AttemptTimeout != time.Second {
		t.Fatalf("unexpected ImageSync config: %#v", config.ImageSync)
	}
	if config.Registry.Username != "file-registry-user" || config.Registry.Password != "file-registry-password" {
		t.Fatalf("Registry credentials were not loaded from file: %#v", config.Registry)
	}
	if len(config.Registry.AllowedHosts) != 1 || config.Registry.AllowedHosts[0] != "registry.example.com" {
		t.Fatalf("unexpected Registry.AllowedHosts: %#v", config.Registry.AllowedHosts)
	}
	if config.Registry.RequestTimeout != 10*time.Second {
		t.Fatalf("Registry.RequestTimeout = %s", config.Registry.RequestTimeout)
	}
}

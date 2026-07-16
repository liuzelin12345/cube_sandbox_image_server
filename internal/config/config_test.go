package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

Auth:
  APIKeys:
    - file-api-key-1
    - file-api-key-2

TencentCloud:
  SecretID: file-secret-id
  SecretKey: file-secret-key
  Region: file-region
  Endpoint: file.tencentcloud.example.com
  RequestTimeout: 10s

ImageSync:
  Endpoint: https://file.example.com/sync
  AllowedImageNames:
    - image-one
    - team/image-two
  RequestTimeout: 20s
  AttemptTimeout: 1s
  MaxResponseBytes: 1048576
  MaxAttempts: 3
  RetryInterval: 500ms

Registry:
  Username: file-registry-user
  Password: file-registry-password
  RequestTimeout: 10s
`
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configFile, []byte(data), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
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
	if len(config.Auth.APIKeys) != 2 || config.Auth.APIKeys[0] != "file-api-key-1" || config.Auth.APIKeys[1] != "file-api-key-2" {
		t.Fatalf("Auth.APIKeys = %#v", config.Auth.APIKeys)
	}
	if config.TencentCloud.RequestTimeout != 10*time.Second {
		t.Fatalf("unexpected TencentCloud timeouts: %#v", config.TencentCloud)
	}
	if config.ImageSync.Endpoint != "https://file.example.com/sync" || config.ImageSync.MaxAttempts != 3 || config.ImageSync.AttemptTimeout != time.Second {
		t.Fatalf("unexpected ImageSync config: %#v", config.ImageSync)
	}
	if len(config.ImageSync.AllowedImageNames) != 2 || config.ImageSync.AllowedImageNames[0] != "image-one" || config.ImageSync.AllowedImageNames[1] != "team/image-two" {
		t.Fatalf("ImageSync.AllowedImageNames = %#v", config.ImageSync.AllowedImageNames)
	}
	if config.Registry.Username != "file-registry-user" || config.Registry.Password != "file-registry-password" {
		t.Fatalf("Registry credentials were not loaded from file: %#v", config.Registry)
	}
	if config.Registry.RequestTimeout != 10*time.Second {
		t.Fatalf("Registry.RequestTimeout = %s", config.Registry.RequestTimeout)
	}
}

func TestLoadAppliesCentralDefaults(t *testing.T) {
	const data = `Name: cube_sandbox_image_server-api
Host: 127.0.0.1
Port: 43999
Timeout: 30000

Auth:
  APIKeys:
    - file-api-key

TencentCloud:
  SecretID: file-secret-id
  SecretKey: file-secret-key

ImageSync:
  AllowedImageNames:
    - image-one

Registry:
  Username: file-registry-user
  Password: file-registry-password
`
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configFile, []byte(data), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	got, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.TencentCloud.Region != DefaultTencentCloudRegion || got.TencentCloud.Endpoint != DefaultTencentCloudEndpoint || got.TencentCloud.RequestTimeout != DefaultTencentCloudRequestTimeout {
		t.Fatalf("TencentCloud defaults = %#v", got.TencentCloud)
	}
	if got.ImageSync.Endpoint != DefaultImageSyncEndpoint || got.ImageSync.RequestTimeout != DefaultImageSyncRequestTimeout || got.ImageSync.AttemptTimeout != DefaultImageSyncAttemptTimeout {
		t.Fatalf("ImageSync defaults = %#v", got.ImageSync)
	}
	if got.ImageSync.MaxResponseBytes != DefaultImageSyncMaxResponseBytes || got.ImageSync.MaxAttempts != DefaultImageSyncMaxAttempts || got.ImageSync.RetryInterval != DefaultImageSyncRetryInterval {
		t.Fatalf("ImageSync retry defaults = %#v", got.ImageSync)
	}
	if len(got.ImageSync.AllowedImageNames) != 1 || got.ImageSync.AllowedImageNames[0] != "image-one" {
		t.Fatalf("ImageSync.AllowedImageNames = %#v", got.ImageSync.AllowedImageNames)
	}
	if got.Registry.RequestTimeout != DefaultRegistryRequestTimeout {
		t.Fatalf("Registry defaults = %#v", got.Registry)
	}
}

func TestDefaultSandboxCommandReturnsEmptyList(t *testing.T) {
	command := DefaultSandboxCommand()
	if command == nil || len(command) != 0 {
		t.Fatalf("DefaultSandboxCommand() = %#v, want non-nil empty list", command)
	}

	command = append(command, "override")
	if got := DefaultSandboxCommand(); got == nil || len(got) != 0 {
		t.Fatalf("DefaultSandboxCommand() after caller mutation = %#v", got)
	}
}

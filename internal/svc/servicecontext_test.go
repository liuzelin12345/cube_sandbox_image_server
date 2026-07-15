package svc

import (
	"testing"
	"time"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
)

func TestValidateConfig(t *testing.T) {
	valid := func() config.Config {
		return config.Config{
			TencentCloud: config.TencentCloudConfig{
				SecretID:           "secret-id",
				SecretKey:          "secret-key",
				Region:             "ap-shanghai",
				Endpoint:           "ags.tencentcloudapi.com",
				RequestTimeout:     10 * time.Second,
				StatusPollInterval: 5 * time.Second,
				StatusPollTimeout:  20 * time.Second,
			},
			ImageSync: config.ImageSyncConfig{
				RequestTimeout: 20 * time.Second,
				AttemptTimeout: time.Second,
			},
			Registry: config.RegistryConfig{
				Username:       "registry-user",
				Password:       "registry-password",
				AllowedHosts:   []string{"registry.example.com"},
				RequestTimeout: 10 * time.Second,
			},
		}
	}

	tests := []struct {
		name      string
		mutate    func(*config.Config)
		wantError string
	}{
		{name: "missing secret ID", mutate: func(c *config.Config) { c.TencentCloud.SecretID = "" }, wantError: "TencentCloud.SecretID is required"},
		{name: "missing secret key", mutate: func(c *config.Config) { c.TencentCloud.SecretKey = " " }, wantError: "TencentCloud.SecretKey is required"},
		{name: "missing username", mutate: func(c *config.Config) { c.Registry.Username = "" }, wantError: "Registry.Username is required"},
		{name: "missing password", mutate: func(c *config.Config) { c.Registry.Password = "\t" }, wantError: "Registry.Password is required"},
		{name: "missing allowed hosts", mutate: func(c *config.Config) { c.Registry.AllowedHosts = nil }, wantError: "Registry.AllowedHosts must contain at least one host"},
		{name: "invalid timeout", mutate: func(c *config.Config) { c.Registry.RequestTimeout = 0 }, wantError: "Registry.RequestTimeout must be greater than zero"},
	}

	if err := validateConfig(valid()); err != nil {
		t.Fatalf("validateConfig(valid) error = %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := valid()
			test.mutate(&configuration)
			if err := validateConfig(configuration); err == nil || err.Error() != test.wantError {
				t.Fatalf("validateConfig() error = %v, want %q", err, test.wantError)
			}
		})
	}
}

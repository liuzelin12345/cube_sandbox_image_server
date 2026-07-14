package svc

import (
	"testing"
	"time"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
)

func TestValidateConfigRequiresRegistrySettings(t *testing.T) {
	valid := func() config.Config {
		return config.Config{
			TencentCloud: config.TencentCloudConfig{
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
		name   string
		mutate func(*config.Config)
	}{
		{name: "missing username", mutate: func(c *config.Config) { c.Registry.Username = "" }},
		{name: "missing password", mutate: func(c *config.Config) { c.Registry.Password = "" }},
		{name: "missing allowed hosts", mutate: func(c *config.Config) { c.Registry.AllowedHosts = nil }},
		{name: "invalid timeout", mutate: func(c *config.Config) { c.Registry.RequestTimeout = 0 }},
	}

	if err := validateConfig(valid()); err != nil {
		t.Fatalf("validateConfig(valid) error = %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := valid()
			test.mutate(&configuration)
			if err := validateConfig(configuration); err == nil {
				t.Fatal("validateConfig() error = nil")
			}
		})
	}
}

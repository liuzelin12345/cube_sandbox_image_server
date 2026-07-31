package svc

import (
	"testing"
	"time"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
)

func TestValidateConfig(t *testing.T) {
	valid := func() config.Config {
		return config.Config{
			Auth: config.AuthConfig{APIKeys: []string{"api-key-1", "api-key-2"}},
			TencentCloud: config.TencentCloudConfig{
				SecretID:       "secret-id",
				SecretKey:      "secret-key",
				Region:         "ap-shanghai",
				Endpoint:       "ags.tencentcloudapi.com",
				RequestTimeout: 10 * time.Second,
			},
			ImageSync: config.ImageSyncConfig{
				AllowedImageNames: []string{"image-one", "team/image-two"},
				RequestTimeout:    20 * time.Second,
				AttemptTimeout:    time.Second,
			},
			Registry: config.RegistryConfig{
				Username:       "registry-user",
				Password:       "registry-password",
				RequestTimeout: 10 * time.Second,
			},
		}
	}

	tests := []struct {
		name      string
		mutate    func(*config.Config)
		wantError string
	}{
		{name: "missing API keys", mutate: func(c *config.Config) { c.Auth.APIKeys = nil }, wantError: "Auth.APIKeys must contain at least one key"},
		{name: "blank API key", mutate: func(c *config.Config) { c.Auth.APIKeys = []string{" "} }, wantError: "Auth.APIKeys[0] is required"},
		{name: "API key with surrounding whitespace", mutate: func(c *config.Config) { c.Auth.APIKeys = []string{" api-key "} }, wantError: "Auth.APIKeys[0] must not contain surrounding whitespace"},
		{name: "missing secret ID", mutate: func(c *config.Config) { c.TencentCloud.SecretID = "" }, wantError: "TencentCloud.SecretID is required"},
		{name: "missing secret key", mutate: func(c *config.Config) { c.TencentCloud.SecretKey = " " }, wantError: "TencentCloud.SecretKey is required"},
		{name: "missing allowed image names", mutate: func(c *config.Config) { c.ImageSync.AllowedImageNames = nil }, wantError: "ImageSync.AllowedImageNames must contain at least one image name"},
		{name: "blank allowed image name", mutate: func(c *config.Config) { c.ImageSync.AllowedImageNames = []string{" "} }, wantError: "ImageSync.AllowedImageNames[0] is required"},
		{name: "allowed image name with surrounding whitespace", mutate: func(c *config.Config) { c.ImageSync.AllowedImageNames = []string{" image-one "} }, wantError: "ImageSync.AllowedImageNames[0] must not contain surrounding whitespace"},
		{name: "missing username", mutate: func(c *config.Config) { c.Registry.Username = "" }, wantError: "Registry.Username is required"},
		{name: "missing password", mutate: func(c *config.Config) { c.Registry.Password = "\t" }, wantError: "Registry.Password is required"},
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

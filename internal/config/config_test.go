package config

import (
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestConfigLoadsCredentialsFromEnvironment(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "secret-id")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "secret-key")
	t.Setenv("REGISTRY_USERNAME", "registry-user")
	t.Setenv("REGISTRY_PASSWORD", "registry-password")

	var config Config
	if err := conf.Load("../../etc/cubesandboximageserver-api.yaml", &config); err != nil {
		t.Fatalf("conf.Load() error = %v", err)
	}
	if config.TencentCloud.SecretID != "secret-id" || config.TencentCloud.SecretKey != "secret-key" {
		t.Fatalf("credentials were not loaded from environment")
	}
	if config.Timeout != 30000 {
		t.Fatalf("REST Timeout = %d", config.Timeout)
	}
	if config.TencentCloud.RequestTimeout != 10*time.Second ||
		config.TencentCloud.StatusPollInterval != 5*time.Second ||
		config.TencentCloud.StatusPollTimeout != 20*time.Second {
		t.Fatalf("unexpected TencentCloud timeouts: %#v", config.TencentCloud)
	}
	if config.ImageSync.MaxAttempts != 3 || config.ImageSync.AttemptTimeout != time.Second {
		t.Fatalf("unexpected ImageSync config: %#v", config.ImageSync)
	}
	if config.Registry.Username != "registry-user" || config.Registry.Password != "registry-password" {
		t.Fatalf("registry credentials were not loaded from environment")
	}
	if len(config.Registry.AllowedHosts) != 1 || config.Registry.AllowedHosts[0] != "ths-shanghai-tcr.tencentcloudcr.com" {
		t.Fatalf("unexpected Registry.AllowedHosts: %#v", config.Registry.AllowedHosts)
	}
	if config.Registry.RequestTimeout != 10*time.Second {
		t.Fatalf("Registry.RequestTimeout = %s", config.Registry.RequestTimeout)
	}
}

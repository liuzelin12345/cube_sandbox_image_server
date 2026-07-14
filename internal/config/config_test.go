package config

import (
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestConfigLoadsCredentialsFromEnvironment(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "secret-id")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "secret-key")

	var config Config
	if err := conf.Load("../../etc/cubesandboximageserver-api.yaml", &config); err != nil {
		t.Fatalf("conf.Load() error = %v", err)
	}
	if config.TencentCloud.SecretID != "secret-id" || config.TencentCloud.SecretKey != "secret-key" {
		t.Fatalf("credentials were not loaded from environment")
	}
	if config.TencentCloud.StatusPollTimeout != 45*time.Second {
		t.Fatalf("StatusPollTimeout = %s", config.TencentCloud.StatusPollTimeout)
	}
	if config.ImageSync.MaxAttempts != 3 || config.ImageSync.AttemptTimeout != 5*time.Second {
		t.Fatalf("unexpected ImageSync config: %#v", config.ImageSync)
	}
}

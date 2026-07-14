package types

import (
	"testing"

	"github.com/zeromicro/go-zero/core/mapping"
)

func TestCreateSandboxToolRequestDefaults(t *testing.T) {
	payload := []byte(`{
		"toolName":"sandbox-tool",
		"customConfiguration":{
			"image":"registry.example.com/team/sandbox:latest",
			"command":["/usr/local/bin/start-lightweight-code-interpreter.sh"]
		}
	}`)

	var request CreateSandboxToolRequest
	if err := mapping.UnmarshalJsonBytes(payload, &request); err != nil {
		t.Fatalf("UnmarshalJsonBytes() error = %v", err)
	}
	if request.DefaultTimeout != "5m" {
		t.Fatalf("DefaultTimeout = %q", request.DefaultTimeout)
	}
	if request.RoleArn != "qcs::cam::uin/100032159895:roleName/sandbox_test" {
		t.Fatalf("RoleArn = %q", request.RoleArn)
	}
	if request.CustomConfiguration.ImageRegistryType != "enterprise" {
		t.Fatalf("ImageRegistryType = %q", request.CustomConfiguration.ImageRegistryType)
	}
}

func TestCreateSandboxToolRequestRequiresCommand(t *testing.T) {
	payload := []byte(`{
		"toolName":"sandbox-tool",
		"customConfiguration":{"image":"registry.example.com/team/sandbox:latest"}
	}`)

	var request CreateSandboxToolRequest
	if err := mapping.UnmarshalJsonBytes(payload, &request); err == nil {
		t.Fatal("UnmarshalJsonBytes() error = nil")
	}
}

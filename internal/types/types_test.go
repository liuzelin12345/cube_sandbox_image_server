package types

import (
	"testing"

	"github.com/zeromicro/go-zero/core/mapping"
)

func TestCreateSandboxToolRequestDefaults(t *testing.T) {
	payload := []byte(`{
		"toolName":"sandbox-tool",
		"customConfiguration":{
			"image":"registry.example.com/team/sandbox:latest"
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

func TestCreateSandboxToolRequestAllowsMissingCommand(t *testing.T) {
	payload := []byte(`{
		"toolName":"sandbox-tool",
		"customConfiguration":{"image":"registry.example.com/team/sandbox:latest"}
	}`)

	var request CreateSandboxToolRequest
	if err := mapping.UnmarshalJsonBytes(payload, &request); err != nil {
		t.Fatalf("UnmarshalJsonBytes() error = %v", err)
	}
	if len(request.CustomConfiguration.Command) != 0 {
		t.Fatalf("Command = %#v, want empty", request.CustomConfiguration.Command)
	}
}

func TestCreateSandboxToolRequestStillRequiresToolNameAndImage(t *testing.T) {
	tests := []string{
		`{"customConfiguration":{"image":"registry.example.com/team/sandbox:latest"}}`,
		`{"toolName":"sandbox-tool","customConfiguration":{}}`,
	}
	for _, payload := range tests {
		var request CreateSandboxToolRequest
		if err := mapping.UnmarshalJsonBytes([]byte(payload), &request); err == nil {
			t.Fatalf("UnmarshalJsonBytes(%s) error = nil", payload)
		}
	}
}

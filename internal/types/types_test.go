package types

import (
	"testing"

	"github.com/zeromicro/go-zero/core/mapping"
)

func TestCreateSandboxToolRequestLeavesDefaultsToBusinessLogic(t *testing.T) {
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
	if request.DefaultTimeout != "" || request.RoleArn != "" || request.Persistent != nil || request.CustomConfiguration.ImageRegistryType != "" {
		t.Fatalf("generated types unexpectedly applied defaults: %#v", request)
	}
}

func TestCreateSandboxToolRequestPreservesExplicitFalse(t *testing.T) {
	payload := []byte(`{
		"toolName":"sandbox-tool",
		"persistent":false,
		"storageMounts":[{"readOnly":false}],
		"customConfiguration":{"image":"registry.example.com/team/sandbox:latest"}
	}`)

	var request CreateSandboxToolRequest
	if err := mapping.UnmarshalJsonBytes(payload, &request); err != nil {
		t.Fatalf("UnmarshalJsonBytes() error = %v", err)
	}
	if request.Persistent == nil || *request.Persistent {
		t.Fatalf("Persistent = %#v, want explicit false", request.Persistent)
	}
	if len(request.StorageMounts) != 1 || request.StorageMounts[0].ReadOnly == nil || *request.StorageMounts[0].ReadOnly {
		t.Fatalf("ReadOnly = %#v, want explicit false", request.StorageMounts)
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

func TestCreateSandboxToolRequestStorageMountLeavesDefaultsToBusinessLogic(t *testing.T) {
	payload := []byte(`{
		"toolName":"sandbox-tool",
		"storageMounts":[{"storageSource":{"cos":{}}}],
		"customConfiguration":{"image":"registry.example.com/team/sandbox:latest"}
	}`)

	var request CreateSandboxToolRequest
	if err := mapping.UnmarshalJsonBytes(payload, &request); err != nil {
		t.Fatalf("UnmarshalJsonBytes() error = %v", err)
	}
	if len(request.StorageMounts) != 1 {
		t.Fatalf("StorageMounts length = %d, want 1", len(request.StorageMounts))
	}
	mount := request.StorageMounts[0]
	if mount.Name != "" || mount.MountPath != "" || mount.ReadOnly != nil {
		t.Fatalf("generated types unexpectedly applied storage defaults: %#v", mount)
	}
	if mount.StorageSource.Cos.Endpoint != "" || mount.StorageSource.Cos.BucketName != "" || mount.StorageSource.Cos.BucketPath != "" {
		t.Fatalf("generated types unexpectedly applied COS defaults: %#v", mount.StorageSource.Cos)
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

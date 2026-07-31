package cubeSandboxImage

import (
	"testing"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"
)

func TestBuildCreateSandboxToolRequestMapsFullConfiguration(t *testing.T) {
	input := validCreateRequest()
	input.Description = "custom sandbox"
	input.DefaultTimeout = "10m"
	input.ClientToken = "token-1"
	input.Persistent = boolPtr(true)
	input.Tags = []types.Tag{{Key: "env", Value: "test"}}
	input.CustomConfiguration.Command = []string{"/usr/local/bin/start.sh"}
	input.CustomConfiguration.Args = []string{"--serve"}
	input.CustomConfiguration.Env = []types.EnvironmentVariable{{Name: "PORT", Value: "49999"}}
	input.CustomConfiguration.Ports = []types.PortConfiguration{
		{Name: "interpreter", Protocol: "TCP", Port: 49999},
		{Name: "management", Protocol: "TCP", Port: 49983},
	}
	input.CustomConfiguration.Resources = types.ResourceConfiguration{CPU: "2", Memory: "4Gi"}
	input.CustomConfiguration.Probe = types.ProbeConfiguration{
		HttpGet: types.HttpGetAction{Path: "/health", Port: 49999},
	}
	input.CustomConfiguration.DnsConfig = types.DnsConfiguration{Servers: []string{"10.0.0.1"}}

	request, err := buildCreateSandboxToolRequest(input)
	if err != nil {
		t.Fatalf("buildCreateSandboxToolRequest() error = %v", err)
	}
	if got := *request.ToolType; got != "custom" {
		t.Fatalf("ToolType = %q, want custom", got)
	}
	if request.Persistent == nil || !*request.Persistent {
		t.Fatalf("Persistent was not mapped")
	}
	if request.CustomConfiguration.ImageRegistryType == nil || *request.CustomConfiguration.ImageRegistryType != config.DefaultSandboxImageRegistryType {
		t.Fatalf("default ImageRegistryType was not mapped")
	}
	if len(request.CustomConfiguration.Ports) != 2 || *request.CustomConfiguration.Ports[1].Port != 49983 {
		t.Fatalf("ports were not mapped: %#v", request.CustomConfiguration.Ports)
	}
	probe := request.CustomConfiguration.Probe
	if probe == nil || probe.HttpGet == nil || *probe.HttpGet.Scheme != config.DefaultSandboxProbeScheme || *probe.ReadyTimeoutMs != config.DefaultSandboxReadyTimeoutMs {
		t.Fatalf("probe defaults were not mapped: %#v", probe)
	}
	if request.CustomConfiguration.DNSConfig == nil || len(request.CustomConfiguration.DNSConfig.Servers) != 1 {
		t.Fatalf("DNS config was not mapped")
	}
}

func TestBuildCreateSandboxToolRequestAppliesCentralDefaults(t *testing.T) {
	input := &types.CreateSandboxToolRequest{
		ToolName: "sandbox-tool-defaults",
		CustomConfiguration: types.CustomConfiguration{
			Image:   "registry.example.com/team/sandbox:latest",
			Command: []string{"/image-entrypoint"},
		},
	}

	request, err := buildCreateSandboxToolRequest(input)
	if err != nil {
		t.Fatalf("buildCreateSandboxToolRequest() error = %v", err)
	}
	if request.DefaultTimeout == nil || *request.DefaultTimeout != config.DefaultSandboxToolTimeout {
		t.Fatalf("DefaultTimeout = %#v", request.DefaultTimeout)
	}
	if request.Description == nil || *request.Description != config.DefaultSandboxDescription {
		t.Fatalf("Description = %#v", request.Description)
	}
	if request.ClientToken != nil {
		t.Fatalf("ClientToken = %#v, want nil", request.ClientToken)
	}
	if request.RoleArn == nil || *request.RoleArn != config.DefaultSandboxRoleARN {
		t.Fatalf("RoleArn = %#v", request.RoleArn)
	}
	if request.Persistent == nil || *request.Persistent != config.DefaultSandboxPersistent {
		t.Fatalf("Persistent = %#v", request.Persistent)
	}
	if request.NetworkConfiguration == nil || *request.NetworkConfiguration.NetworkMode != config.DefaultSandboxNetworkMode {
		t.Fatalf("NetworkMode was not defaulted")
	}
	vpc := request.NetworkConfiguration.VpcConfig
	if vpc == nil || len(vpc.SubnetIds) != 1 || *vpc.SubnetIds[0] != config.DefaultSandboxVPCSubnetID || len(vpc.SecurityGroupIds) != 1 || *vpc.SecurityGroupIds[0] != config.DefaultSandboxSecurityGroupID {
		t.Fatalf("VPC defaults = %#v", vpc)
	}
	configuration := request.CustomConfiguration
	if len(configuration.Command) != 1 || *configuration.Command[0] != "/image-entrypoint" {
		t.Fatalf("command = %#v", configuration.Command)
	}
	if len(configuration.Ports) != 1 || *configuration.Ports[0].Name != config.DefaultSandboxPortName || *configuration.Ports[0].Protocol != config.DefaultSandboxPortProtocol || *configuration.Ports[0].Port != config.DefaultSandboxPort {
		t.Fatalf("default ports = %#v", configuration.Ports)
	}
	if configuration.Resources == nil || *configuration.Resources.CPU != config.DefaultSandboxCPU || *configuration.Resources.Memory != config.DefaultSandboxMemory {
		t.Fatalf("default resources = %#v", configuration.Resources)
	}
	if configuration.Probe == nil || *configuration.Probe.HttpGet.Path != config.DefaultSandboxProbePath || *configuration.Probe.HttpGet.Port != config.DefaultSandboxProbePort || *configuration.Probe.ProbePeriodMs != config.DefaultSandboxProbePeriodMs {
		t.Fatalf("default probe = %#v", configuration.Probe)
	}
	if request.StorageMounts != nil {
		t.Fatalf("StorageMounts = %#v, want nil", request.StorageMounts)
	}
}

func TestBuildCreateSandboxToolRequestMapsStorageMountOverrides(t *testing.T) {
	input := validCreateRequest()
	input.StorageMounts = []types.StorageMount{
		{
			Name: "workspace",
			StorageSource: types.StorageSource{Cos: types.CosStorageSource{
				Endpoint:   "custom.cos.ap-shanghai.myqcloud.com",
				BucketName: "custom-bucket",
				BucketPath: "/projects/one",
			}},
			MountPath: "/workspace",
			ReadOnly:  boolPtr(true),
		},
		{
			Name:      "cache",
			MountPath: "/cache",
		},
	}

	request, err := buildCreateSandboxToolRequest(input)
	if err != nil {
		t.Fatalf("buildCreateSandboxToolRequest() error = %v", err)
	}
	if len(request.StorageMounts) != 2 {
		t.Fatalf("StorageMounts length = %d, want 2", len(request.StorageMounts))
	}
	first := request.StorageMounts[0]
	if *first.Name != "workspace" || *first.MountPath != "/workspace" || first.ReadOnly == nil || !*first.ReadOnly {
		t.Fatalf("first storage mount = %#v", first)
	}
	if cos := first.StorageSource.Cos; *cos.Endpoint != "custom.cos.ap-shanghai.myqcloud.com" || *cos.BucketName != "custom-bucket" || *cos.BucketPath != "/projects/one" {
		t.Fatalf("first COS configuration = %#v", cos)
	}
	second := request.StorageMounts[1]
	if *second.Name != "cache" || *second.MountPath != "/cache" || *second.StorageSource.Cos.Endpoint != config.DefaultSandboxCOSEndpoint {
		t.Fatalf("second storage mount defaults = %#v", second)
	}
}

func TestBuildStorageMountsLeavesMissingOrEmptyListUnmounted(t *testing.T) {
	tests := []struct {
		name  string
		input []types.StorageMount
	}{
		{name: "missing", input: nil},
		{name: "empty", input: []types.StorageMount{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mounts, err := buildStorageMounts(test.input)
			if err != nil {
				t.Fatalf("buildStorageMounts() error = %v", err)
			}
			if mounts != nil {
				t.Fatalf("buildStorageMounts() = %#v, want nil", mounts)
			}
		})
	}
}

func TestBuildCreateSandboxToolRequestValidatesCrossFieldConstraints(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.CreateSandboxToolRequest)
	}{
		{
			name: "invalid tool name",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.ToolName = "contains spaces"
			},
		},
		{
			name: "blank command",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.CustomConfiguration.Command = []string{" "}
			},
		},
		{
			name: "VPC with blank subnet",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.NetworkConfiguration.NetworkMode = "VPC"
				req.NetworkConfiguration.VpcConfig.SubnetIds = []string{" "}
			},
		},
		{
			name: "VPC config with public network",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.NetworkConfiguration.VpcConfig.SubnetIds = []string{"subnet-1"}
			},
		},
		{
			name: "duplicate port",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.CustomConfiguration.Ports = []types.PortConfiguration{
					{Name: "one", Protocol: "TCP", Port: 49999},
					{Name: "two", Protocol: "TCP", Port: 49999},
				}
			},
		},
		{
			name: "negative probe threshold",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.CustomConfiguration.Probe.FailureThreshold = -1
			},
		},
		{
			name: "invalid timeout",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.DefaultTimeout = "tomorrow"
			},
		},
		{
			name: "blank storage mount name",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.StorageMounts = []types.StorageMount{{Name: " "}}
			},
		},
		{
			name: "blank COS endpoint",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.StorageMounts = []types.StorageMount{{StorageSource: types.StorageSource{Cos: types.CosStorageSource{Endpoint: " "}}}}
			},
		},
		{
			name: "blank COS bucket name",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.StorageMounts = []types.StorageMount{{StorageSource: types.StorageSource{Cos: types.CosStorageSource{BucketName: " "}}}}
			},
		},
		{
			name: "relative COS bucket path",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.StorageMounts = []types.StorageMount{{StorageSource: types.StorageSource{Cos: types.CosStorageSource{BucketPath: "relative"}}}}
			},
		},
		{
			name: "relative storage mount path",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.StorageMounts = []types.StorageMount{{MountPath: "relative"}}
			},
		},
		{
			name: "duplicate storage mount name",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.StorageMounts = []types.StorageMount{
					{Name: "shared", MountPath: "/one"},
					{Name: "shared", MountPath: "/two"},
				}
			},
		},
		{
			name: "duplicate storage mount path",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.StorageMounts = []types.StorageMount{
					{Name: "one", MountPath: "/shared"},
					{Name: "two", MountPath: "/shared"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validCreateRequest()
			tt.mutate(request)
			if _, err := buildCreateSandboxToolRequest(request); err == nil {
				t.Fatalf("buildCreateSandboxToolRequest() error = nil")
			}
		})
	}
}

func validCreateRequest() *types.CreateSandboxToolRequest {
	return &types.CreateSandboxToolRequest{
		ToolName: "sandbox-tool-1",
		RoleArn:  "qcs::cam::uin/100000000001:roleName/AGSRole",
		NetworkConfiguration: types.NetworkConfiguration{
			NetworkMode: "PUBLIC",
		},
		CustomConfiguration: types.CustomConfiguration{
			Image:   "registry.example.com/team/sandbox:latest",
			Command: []string{"/image-entrypoint"},
		},
	}
}

package cubeSandboxImage

import (
	"testing"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"
)

func TestBuildCreateSandboxToolRequestMapsFullConfiguration(t *testing.T) {
	input := validCreateRequest()
	input.Description = "custom sandbox"
	input.DefaultTimeout = "10m"
	input.ClientToken = "token-1"
	input.Persistent = true
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
	if request.CustomConfiguration.ImageRegistryType == nil || *request.CustomConfiguration.ImageRegistryType != "enterprise" {
		t.Fatalf("default ImageRegistryType was not mapped")
	}
	if len(request.CustomConfiguration.Ports) != 2 || *request.CustomConfiguration.Ports[1].Port != 49983 {
		t.Fatalf("ports were not mapped: %#v", request.CustomConfiguration.Ports)
	}
	probe := request.CustomConfiguration.Probe
	if probe == nil || probe.HttpGet == nil || *probe.HttpGet.Scheme != "HTTP" || *probe.ReadyTimeoutMs != 30000 {
		t.Fatalf("probe defaults were not mapped: %#v", probe)
	}
	if request.CustomConfiguration.DNSConfig == nil || len(request.CustomConfiguration.DNSConfig.Servers) != 1 {
		t.Fatalf("DNS config was not mapped")
	}
}

func TestBuildCreateSandboxToolRequestAppliesEnvDefaults(t *testing.T) {
	input := &types.CreateSandboxToolRequest{
		ToolName: "sandbox-tool-defaults",
		CustomConfiguration: types.CustomConfiguration{
			Image:   "registry.example.com/team/sandbox:latest",
			Command: []string{"/usr/local/bin/start-lightweight-code-interpreter.sh"},
		},
	}

	request, err := buildCreateSandboxToolRequest(input)
	if err != nil {
		t.Fatalf("buildCreateSandboxToolRequest() error = %v", err)
	}
	if request.DefaultTimeout == nil || *request.DefaultTimeout != "5m" {
		t.Fatalf("DefaultTimeout = %#v", request.DefaultTimeout)
	}
	if request.RoleArn == nil || *request.RoleArn != defaultRoleARN {
		t.Fatalf("RoleArn = %#v", request.RoleArn)
	}
	if request.NetworkConfiguration == nil || *request.NetworkConfiguration.NetworkMode != "PUBLIC" {
		t.Fatalf("NetworkMode was not defaulted")
	}
	configuration := request.CustomConfiguration
	if len(configuration.Ports) != 1 || *configuration.Ports[0].Name != "http" || *configuration.Ports[0].Port != 49999 {
		t.Fatalf("default ports = %#v", configuration.Ports)
	}
	if configuration.Resources == nil || *configuration.Resources.CPU != "1" || *configuration.Resources.Memory != "2Gi" {
		t.Fatalf("default resources = %#v", configuration.Resources)
	}
	if configuration.Probe == nil || *configuration.Probe.HttpGet.Path != "/health" || *configuration.Probe.HttpGet.Port != 49999 {
		t.Fatalf("default probe = %#v", configuration.Probe)
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
			name: "missing command",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.CustomConfiguration.Command = nil
			},
		},
		{
			name: "VPC without subnet",
			mutate: func(req *types.CreateSandboxToolRequest) {
				req.NetworkConfiguration.NetworkMode = "VPC"
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
			Command: []string{"/usr/local/bin/start-lightweight-code-interpreter.sh"},
		},
	}
}

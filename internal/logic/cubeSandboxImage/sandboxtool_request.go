package cubeSandboxImage

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	tcags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"
)

var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,50}$`)

const (
	defaultToolTimeout = "5m"
	defaultRoleARN     = "qcs::cam::uin/100032159895:roleName/sandbox_test"
	defaultProbePath   = "/health"
	defaultProbePort   = int64(49999)
)

func buildCreateSandboxToolRequest(input *types.CreateSandboxToolRequest) (*tcags.CreateSandboxToolRequest, error) {
	if input == nil {
		return nil, fmt.Errorf("request body is required")
	}

	toolName := strings.TrimSpace(input.ToolName)
	if !toolNamePattern.MatchString(toolName) {
		return nil, fmt.Errorf("toolName must contain 1-50 letters, digits, underscores, or hyphens")
	}
	if utf8.RuneCountInString(input.Description) > 200 {
		return nil, fmt.Errorf("description cannot exceed 200 characters")
	}
	if utf8.RuneCountInString(input.ClientToken) > 64 {
		return nil, fmt.Errorf("clientToken cannot exceed 64 characters")
	}
	defaultTimeout := strings.TrimSpace(input.DefaultTimeout)
	if defaultTimeout == "" {
		defaultTimeout = defaultToolTimeout
	}
	if err := validateDefaultTimeout(defaultTimeout); err != nil {
		return nil, err
	}

	roleARN := strings.TrimSpace(input.RoleArn)
	if roleARN == "" {
		roleARN = defaultRoleARN
	}

	network, err := buildNetworkConfiguration(input.NetworkConfiguration)
	if err != nil {
		return nil, err
	}
	custom, err := buildCustomConfiguration(input.CustomConfiguration)
	if err != nil {
		return nil, err
	}
	tags, err := buildTags(input.Tags)
	if err != nil {
		return nil, err
	}

	request := tcags.NewCreateSandboxToolRequest()
	request.ToolName = stringPtr(toolName)
	request.ToolType = stringPtr("custom")
	request.RoleArn = stringPtr(roleARN)
	request.Persistent = boolPtr(input.Persistent)
	request.NetworkConfiguration = network
	request.CustomConfiguration = custom
	request.Tags = tags
	request.Description = optionalString(input.Description)
	request.DefaultTimeout = stringPtr(defaultTimeout)
	request.ClientToken = optionalString(input.ClientToken)

	return request, nil
}

func validateDefaultTimeout(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("defaultTimeout must be a valid Go duration such as 5m or 300s")
	}
	if duration <= 0 || duration > 24*time.Hour {
		return fmt.Errorf("defaultTimeout must be greater than zero and no more than 24h")
	}
	return nil
}

func buildNetworkConfiguration(input types.NetworkConfiguration) (*tcags.NetworkConfiguration, error) {
	mode := strings.ToUpper(strings.TrimSpace(input.NetworkMode))
	if mode == "" {
		mode = "PUBLIC"
	}
	if mode != "PUBLIC" && mode != "VPC" && mode != "SANDBOX" {
		return nil, fmt.Errorf("networkMode must be PUBLIC, VPC, or SANDBOX")
	}

	configuration := &tcags.NetworkConfiguration{NetworkMode: stringPtr(mode)}
	if mode != "VPC" {
		if len(input.VpcConfig.SubnetIds) > 0 || len(input.VpcConfig.SecurityGroupIds) > 0 {
			return nil, fmt.Errorf("vpcConfig can only be set when networkMode is VPC")
		}
		return configuration, nil
	}

	subnets, err := nonEmptyStringPointers("vpcConfig.subnetIds", input.VpcConfig.SubnetIds)
	if err != nil {
		return nil, err
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("vpcConfig.subnetIds is required when networkMode is VPC")
	}
	securityGroups, err := nonEmptyStringPointers("vpcConfig.securityGroupIds", input.VpcConfig.SecurityGroupIds)
	if err != nil {
		return nil, err
	}

	configuration.VpcConfig = &tcags.VPCConfig{
		SubnetIds:        subnets,
		SecurityGroupIds: securityGroups,
	}
	return configuration, nil
}

func buildCustomConfiguration(input types.CustomConfiguration) (*tcags.CustomConfiguration, error) {
	image := strings.TrimSpace(input.Image)
	if image == "" {
		return nil, fmt.Errorf("customConfiguration.image is required")
	}

	registryType := strings.ToLower(strings.TrimSpace(input.ImageRegistryType))
	if registryType == "" {
		registryType = "enterprise"
	}
	if registryType != "enterprise" && registryType != "personal" {
		return nil, fmt.Errorf("customConfiguration.imageRegistryType must be enterprise or personal")
	}

	environment, err := buildEnvironment(input.Env)
	if err != nil {
		return nil, err
	}
	ports, err := buildPorts(input.Ports)
	if err != nil {
		return nil, err
	}
	command, err := nonEmptyStringPointers("customConfiguration.command", input.Command)
	if err != nil {
		return nil, err
	}
	if len(command) == 0 {
		return nil, fmt.Errorf("customConfiguration.command is required")
	}
	probe, err := buildProbe(input.Probe)
	if err != nil {
		return nil, err
	}
	dns, err := buildDNS(input.DnsConfig)
	if err != nil {
		return nil, err
	}

	configuration := &tcags.CustomConfiguration{
		Image:             stringPtr(image),
		ImageRegistryType: stringPtr(registryType),
		Command:           command,
		Args:              stringPointers(input.Args),
		Env:               environment,
		Ports:             ports,
		Probe:             probe,
		DNSConfig:         dns,
	}
	cpu := strings.TrimSpace(input.Resources.CPU)
	if cpu == "" {
		cpu = "1"
	}
	memory := strings.TrimSpace(input.Resources.Memory)
	if memory == "" {
		memory = "2Gi"
	}
	configuration.Resources = &tcags.ResourceConfiguration{
		CPU:    stringPtr(cpu),
		Memory: stringPtr(memory),
	}

	return configuration, nil
}

func buildTags(input []types.Tag) ([]*tcags.Tag, error) {
	result := make([]*tcags.Tag, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for index, tag := range input {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			return nil, fmt.Errorf("tags[%d].key is required", index)
		}
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("tags contains duplicate key %q", key)
		}
		seen[key] = struct{}{}
		value := tag.Value
		result = append(result, &tcags.Tag{Key: stringPtr(key), Value: &value})
	}
	return result, nil
}

func buildEnvironment(input []types.EnvironmentVariable) ([]*tcags.EnvVar, error) {
	result := make([]*tcags.EnvVar, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for index, variable := range input {
		name := strings.TrimSpace(variable.Name)
		if name == "" {
			return nil, fmt.Errorf("customConfiguration.env[%d].name is required", index)
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("customConfiguration.env contains duplicate name %q", name)
		}
		seen[name] = struct{}{}
		value := variable.Value
		result = append(result, &tcags.EnvVar{Name: stringPtr(name), Value: &value})
	}
	return result, nil
}

func buildPorts(input []types.PortConfiguration) ([]*tcags.PortConfiguration, error) {
	if len(input) == 0 {
		input = []types.PortConfiguration{{
			Name:     "http",
			Protocol: "TCP",
			Port:     defaultProbePort,
		}}
	}
	result := make([]*tcags.PortConfiguration, 0, len(input))
	names := make(map[string]struct{}, len(input))
	ports := make(map[int64]struct{}, len(input))
	for index, port := range input {
		name := strings.TrimSpace(port.Name)
		if name == "" {
			return nil, fmt.Errorf("customConfiguration.ports[%d].name is required", index)
		}
		protocol := strings.ToUpper(strings.TrimSpace(port.Protocol))
		if protocol != "TCP" {
			return nil, fmt.Errorf("customConfiguration.ports[%d].protocol must be TCP", index)
		}
		if err := validatePort(port.Port, fmt.Sprintf("customConfiguration.ports[%d].port", index)); err != nil {
			return nil, err
		}
		if _, ok := names[name]; ok {
			return nil, fmt.Errorf("customConfiguration.ports contains duplicate name %q", name)
		}
		if _, ok := ports[port.Port]; ok {
			return nil, fmt.Errorf("customConfiguration.ports contains duplicate port %d", port.Port)
		}
		names[name] = struct{}{}
		ports[port.Port] = struct{}{}
		portNumber := port.Port
		result = append(result, &tcags.PortConfiguration{
			Name:     stringPtr(name),
			Protocol: stringPtr(protocol),
			Port:     &portNumber,
		})
	}
	return result, nil
}

func buildProbe(input types.ProbeConfiguration) (*tcags.ProbeConfiguration, error) {
	path := strings.TrimSpace(input.HttpGet.Path)
	if path == "" {
		path = defaultProbePath
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("customConfiguration.probe.httpGet.path must start with /")
	}
	port := input.HttpGet.Port
	if port == 0 {
		port = defaultProbePort
	}
	if err := validatePort(port, "customConfiguration.probe.httpGet.port"); err != nil {
		return nil, err
	}
	scheme := strings.ToUpper(strings.TrimSpace(input.HttpGet.Scheme))
	if scheme == "" {
		scheme = "HTTP"
	}
	if scheme != "HTTP" && scheme != "HTTPS" {
		return nil, fmt.Errorf("customConfiguration.probe.httpGet.scheme must be HTTP or HTTPS")
	}

	readyTimeout, err := positiveOrDefault(input.ReadyTimeoutMs, 30000, "customConfiguration.probe.readyTimeoutMs")
	if err != nil {
		return nil, err
	}
	probeTimeout, err := positiveOrDefault(input.ProbeTimeoutMs, 1000, "customConfiguration.probe.probeTimeoutMs")
	if err != nil {
		return nil, err
	}
	period, err := positiveOrDefault(input.ProbePeriodMs, 1000, "customConfiguration.probe.probePeriodMs")
	if err != nil {
		return nil, err
	}
	successThreshold, err := positiveOrDefault(input.SuccessThreshold, 1, "customConfiguration.probe.successThreshold")
	if err != nil {
		return nil, err
	}
	failureThreshold, err := positiveOrDefault(input.FailureThreshold, 100, "customConfiguration.probe.failureThreshold")
	if err != nil {
		return nil, err
	}

	return &tcags.ProbeConfiguration{
		HttpGet: &tcags.HttpGetAction{
			Path:   stringPtr(path),
			Port:   &port,
			Scheme: stringPtr(scheme),
		},
		ReadyTimeoutMs:   &readyTimeout,
		ProbeTimeoutMs:   &probeTimeout,
		ProbePeriodMs:    &period,
		SuccessThreshold: &successThreshold,
		FailureThreshold: &failureThreshold,
	}, nil
}

func buildDNS(input types.DnsConfiguration) (*tcags.DNSConfig, error) {
	if len(input.Servers) == 0 && len(input.Searches) == 0 && len(input.Options) == 0 {
		return nil, nil
	}
	servers, err := nonEmptyStringPointers("customConfiguration.dnsConfig.servers", input.Servers)
	if err != nil {
		return nil, err
	}
	for _, server := range servers {
		if net.ParseIP(*server) == nil {
			return nil, fmt.Errorf("customConfiguration.dnsConfig.servers contains invalid IP %q", *server)
		}
	}
	searches, err := nonEmptyStringPointers("customConfiguration.dnsConfig.searches", input.Searches)
	if err != nil {
		return nil, err
	}
	options, err := nonEmptyStringPointers("customConfiguration.dnsConfig.options", input.Options)
	if err != nil {
		return nil, err
	}
	return &tcags.DNSConfig{Servers: servers, Searches: searches, Options: options}, nil
}

func validatePort(port int64, field string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", field)
	}
	return nil
}

func positiveOrDefault(value, fallback int64, field string) (int64, error) {
	if value == 0 {
		return fallback, nil
	}
	if value < 0 {
		return 0, fmt.Errorf("%s must be greater than zero", field)
	}
	return value, nil
}

func nonEmptyStringPointers(field string, values []string) ([]*string, error) {
	result := make([]*string, 0, len(values))
	for index, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("%s[%d] cannot be empty", field, index)
		}
		result = append(result, stringPtr(trimmed))
	}
	return result, nil
}

func stringPointers(values []string) []*string {
	result := make([]*string, 0, len(values))
	for index := range values {
		value := values[index]
		result = append(result, &value)
	}
	return result
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringPtr(value string) *string { return &value }
func boolPtr(value bool) *bool       { return &value }

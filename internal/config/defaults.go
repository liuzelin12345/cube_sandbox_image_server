package config

import (
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
)

// Project configuration defaults.
// Credentials are required and intentionally have no defaults.
const (
	// Default config file used by the service entrypoint.
	DefaultConfigFile = "etc/cubesandboximageserver-api.yaml"

	// Tencent Cloud client defaults.
	DefaultTencentCloudRegion         = "ap-shanghai"
	DefaultTencentCloudEndpoint       = "ags.tencentcloudapi.com"
	DefaultTencentCloudRequestTimeout = 10 * time.Second

	// Image synchronization client defaults.
	DefaultImageSyncEndpoint         = "http://172.20.208.115/sync/image"
	DefaultImageSyncRequestTimeout   = 20 * time.Second
	DefaultImageSyncAttemptTimeout   = 5 * time.Second
	DefaultImageSyncMaxResponseBytes = int64(1 << 20)
	DefaultImageSyncMaxAttempts      = 3
	DefaultImageSyncRetryInterval    = 500 * time.Millisecond

	// Registry client defaults.
	DefaultRegistryAllowedHost    = "aime-agent-tcr.tencentcloudcr.com"
	DefaultRegistryRequestTimeout = 10 * time.Second
)

// API parameter defaults, ordered like the full tool request example.
// toolName and customConfiguration.image are required and have no defaults.
const (
	DefaultSandboxDescription = "custom sandbox tool"
	DefaultSandboxToolTimeout = "5m"
	DefaultSandboxClientToken = ""
	DefaultSandboxRoleARN     = "qcs::cam::uin/100012162316:roleName/aime-ags"
	DefaultSandboxPersistent  = false
	// tags defaults to an empty list.
	DefaultSandboxNetworkMode     = "VPC"
	DefaultSandboxVPCSubnetID     = "subnet-21t3qpcr"
	DefaultSandboxSecurityGroupID = "sg-nkwdhmgy"
	// storageMounts defaults to an empty list. These values apply only to
	// explicitly provided mount entries with omitted fields.
	DefaultSandboxStorageMountName = "aime-agent-harness"
	DefaultSandboxCOSEndpoint      = "aime-agent-cos-1300730068.cos.ap-shanghai.myqcloud.com"
	DefaultSandboxCOSBucketName    = "aime-agent-cos-1300730068"
	DefaultSandboxCOSBucketPath    = "/"
	DefaultSandboxStorageMountPath = "/mnt/cos"
	DefaultSandboxStorageReadOnly  = false
	// customConfiguration.image is required and has no default.
	DefaultSandboxImageRegistryType = "enterprise"
)

// DefaultSandboxCommand returns the example's empty command list.
// Creation fills it from the image Entrypoint, then the image Cmd.
func DefaultSandboxCommand() []string {
	return []string{}
}

const (
	// args and env default to empty lists.
	DefaultSandboxPortName         = "envd"
	DefaultSandboxPortProtocol     = "TCP"
	DefaultSandboxPort             = int64(49983)
	DefaultSandboxCPU              = "0.5"
	DefaultSandboxMemory           = "1Gi"
	DefaultSandboxProbePath        = "/health"
	DefaultSandboxProbePort        = int64(49999)
	DefaultSandboxProbeScheme      = "HTTP"
	DefaultSandboxReadyTimeoutMs   = int64(30000)
	DefaultSandboxProbeTimeoutMs   = int64(1000)
	DefaultSandboxProbePeriodMs    = int64(300)
	DefaultSandboxSuccessThreshold = int64(1)
	DefaultSandboxFailureThreshold = int64(100)
	// dnsConfig defaults to an empty object.
)

// DefaultConfig returns a new config value so callers cannot mutate shared
// slices such as Registry.AllowedHosts.
func DefaultConfig() Config {
	return Config{
		TencentCloud: TencentCloudConfig{
			Region:         DefaultTencentCloudRegion,
			Endpoint:       DefaultTencentCloudEndpoint,
			RequestTimeout: DefaultTencentCloudRequestTimeout,
		},
		ImageSync: ImageSyncConfig{
			Endpoint:         DefaultImageSyncEndpoint,
			RequestTimeout:   DefaultImageSyncRequestTimeout,
			AttemptTimeout:   DefaultImageSyncAttemptTimeout,
			MaxResponseBytes: DefaultImageSyncMaxResponseBytes,
			MaxAttempts:      DefaultImageSyncMaxAttempts,
			RetryInterval:    DefaultImageSyncRetryInterval,
		},
		Registry: RegistryConfig{
			AllowedHosts:   []string{DefaultRegistryAllowedHost},
			RequestTimeout: DefaultRegistryRequestTimeout,
		},
	}
}

// Load starts from the centralized defaults and then applies explicit values
// from the selected runtime config file.
func Load(path string) (Config, error) {
	c := DefaultConfig()
	if err := conf.Load(path, &c); err != nil {
		return Config{}, fmt.Errorf("load config %q: %w", path, err)
	}
	return c, nil
}

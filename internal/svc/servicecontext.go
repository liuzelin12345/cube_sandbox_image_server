package svc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	tcags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/client/imagesync"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/client/registrycommand"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
)

type SandboxToolClient interface {
	CreateSandboxToolWithContext(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error)
	DescribeSandboxToolListWithContext(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error)
}

type ImageSyncClient interface {
	Sync(context.Context, imagesync.Request) (*imagesync.Response, error)
}

type ImageCommandResolver interface {
	ResolveCommand(context.Context, string) ([]string, error)
}

type ServiceContext struct {
	Config               config.Config
	SandboxToolClient    SandboxToolClient
	ImageSyncClient      ImageSyncClient
	ImageCommandResolver ImageCommandResolver
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	if err := validateConfig(c); err != nil {
		return nil, err
	}
	secretID := strings.TrimSpace(c.TencentCloud.SecretID)
	secretKey := strings.TrimSpace(c.TencentCloud.SecretKey)

	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.Endpoint = strings.TrimSpace(c.TencentCloud.Endpoint)
	sandboxToolClient, err := tcags.NewClient(
		common.NewCredential(secretID, secretKey),
		strings.TrimSpace(c.TencentCloud.Region),
		clientProfile,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize TencentCloud AGS client: %w", err)
	}

	imageSyncClient, err := imagesync.NewClient(
		c.ImageSync.Endpoint,
		&http.Client{Timeout: c.ImageSync.AttemptTimeout},
		c.ImageSync.MaxResponseBytes,
		c.ImageSync.MaxAttempts,
		c.ImageSync.RetryInterval,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize image sync client: %w", err)
	}
	imageCommandResolver, err := registrycommand.NewClient(
		c.Registry.AllowedHosts,
		c.Registry.Username,
		c.Registry.Password,
		c.Registry.RequestTimeout,
		http.DefaultTransport,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize image command resolver: %w", err)
	}

	return &ServiceContext{
		Config:               c,
		SandboxToolClient:    sandboxToolClient,
		ImageSyncClient:      imageSyncClient,
		ImageCommandResolver: imageCommandResolver,
	}, nil
}

func validateConfig(c config.Config) error {
	if strings.TrimSpace(c.TencentCloud.SecretID) == "" {
		return fmt.Errorf("TencentCloud.SecretID is required")
	}
	if strings.TrimSpace(c.TencentCloud.SecretKey) == "" {
		return fmt.Errorf("TencentCloud.SecretKey is required")
	}
	if strings.TrimSpace(c.TencentCloud.Region) == "" {
		return fmt.Errorf("TencentCloud.Region is required")
	}
	if strings.TrimSpace(c.TencentCloud.Endpoint) == "" {
		return fmt.Errorf("TencentCloud.Endpoint is required")
	}
	if c.TencentCloud.RequestTimeout <= 0 {
		return fmt.Errorf("TencentCloud.RequestTimeout must be greater than zero")
	}
	if c.ImageSync.RequestTimeout <= 0 {
		return fmt.Errorf("ImageSync.RequestTimeout must be greater than zero")
	}
	if c.ImageSync.AttemptTimeout <= 0 {
		return fmt.Errorf("ImageSync.AttemptTimeout must be greater than zero")
	}
	if strings.TrimSpace(c.Registry.Username) == "" {
		return fmt.Errorf("Registry.Username is required")
	}
	if strings.TrimSpace(c.Registry.Password) == "" {
		return fmt.Errorf("Registry.Password is required")
	}
	if len(c.Registry.AllowedHosts) == 0 {
		return fmt.Errorf("Registry.AllowedHosts must contain at least one host")
	}
	if c.Registry.RequestTimeout <= 0 {
		return fmt.Errorf("Registry.RequestTimeout must be greater than zero")
	}
	return nil
}

package cubeSandboxImage

import (
	"context"
	"errors"
	"testing"
	"time"

	tcags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/apperror"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"
)

func TestGetSandboxToolStatusReturnsMatchingTool(t *testing.T) {
	client := statusSandboxToolClient(func(_ context.Context, req *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
		if len(req.ToolIds) != 1 || req.ToolIds[0] == nil || *req.ToolIds[0] != "sdt-1" {
			t.Fatalf("unexpected ToolIds: %#v", req.ToolIds)
		}
		return describeStatusResponse("sdt-1", "ACTIVE", "", "request-1"), nil
	})

	response, err := newStatusLogic(client, time.Second).GetSandboxToolStatus(&types.GetSandboxToolStatusRequest{ToolId: " sdt-1 "})
	if err != nil {
		t.Fatalf("GetSandboxToolStatus() error = %v", err)
	}
	if response.ToolId != "sdt-1" || response.Status != "ACTIVE" || response.RequestId != "request-1" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestGetSandboxToolStatusReturnsFailureReason(t *testing.T) {
	client := statusSandboxToolClient(func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
		return describeStatusResponse("sdt-failed", "failed", "image cannot be pulled", "request-2"), nil
	})

	response, err := newStatusLogic(client, time.Second).GetSandboxToolStatus(&types.GetSandboxToolStatusRequest{ToolId: "sdt-failed"})
	if err != nil {
		t.Fatalf("GetSandboxToolStatus() error = %v", err)
	}
	if response.Status != "FAILED" || response.StatusReason != "image cannot be pulled" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestGetSandboxToolStatusReturnsNotFound(t *testing.T) {
	client := statusSandboxToolClient(func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
		return describeStatusResponse("sdt-other", "ACTIVE", "", "request-3"), nil
	})

	_, err := newStatusLogic(client, time.Second).GetSandboxToolStatus(&types.GetSandboxToolStatusRequest{ToolId: "sdt-missing"})
	assertAppError(t, err, 404, "TOOL_NOT_FOUND")
}

func TestGetSandboxToolStatusReturnsGatewayTimeout(t *testing.T) {
	client := statusSandboxToolClient(func(ctx context.Context, _ *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	_, err := newStatusLogic(client, 5*time.Millisecond).GetSandboxToolStatus(&types.GetSandboxToolStatusRequest{ToolId: "sdt-timeout"})
	assertAppError(t, err, 504, "TENCENTCLOUD_TIMEOUT")
}

func statusSandboxToolClient(describe func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error)) *sandboxToolClientMock {
	return &sandboxToolClientMock{
		create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			panic("CreateSandboxTool must not be called by the status endpoint")
		},
		describe: describe,
	}
}

func newStatusLogic(client svc.SandboxToolClient, timeout time.Duration) *GetSandboxToolStatusLogic {
	return NewGetSandboxToolStatusLogic(context.Background(), &svc.ServiceContext{
		Config:            config.Config{TencentCloud: config.TencentCloudConfig{RequestTimeout: timeout}},
		SandboxToolClient: client,
	})
}

func describeStatusResponse(toolID, status, reason, requestID string) *tcags.DescribeSandboxToolListResponse {
	tool := &tcags.SandboxTool{
		ToolId: stringPtr(toolID),
		Status: stringPtr(status),
	}
	if reason != "" {
		tool.StatusReason = stringPtr(reason)
	}
	return &tcags.DescribeSandboxToolListResponse{Response: &tcags.DescribeSandboxToolListResponseParams{
		SandboxToolSet: []*tcags.SandboxTool{tool},
		RequestId:      stringPtr(requestID),
	}}
}

func assertAppError(t *testing.T, err error, statusCode int, code string) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %#v, want *apperror.Error", err)
	}
	if appErr.StatusCode != statusCode || appErr.Code != code {
		t.Fatalf("app error = %#v, want status=%d code=%s", appErr, statusCode, code)
	}
}

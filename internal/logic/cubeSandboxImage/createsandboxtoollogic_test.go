package cubeSandboxImage

import (
	"context"
	"errors"
	"testing"
	"time"

	tcags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
)

type sandboxToolClientMock struct {
	create   func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error)
	describe func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error)
}

func (m *sandboxToolClientMock) CreateSandboxToolWithContext(ctx context.Context, req *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
	return m.create(ctx, req)
}

func (m *sandboxToolClientMock) DescribeSandboxToolListWithContext(ctx context.Context, req *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
	return m.describe(ctx, req)
}

func TestCreateSandboxToolWaitsForActive(t *testing.T) {
	describeCalls := 0
	client := &sandboxToolClientMock{
		create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			return createSDKResponse("sdt-1", "request-1"), nil
		},
		describe: func(_ context.Context, req *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
			describeCalls++
			if len(req.ToolIds) != 1 || *req.ToolIds[0] != "sdt-1" {
				t.Fatalf("unexpected ToolIds: %#v", req.ToolIds)
			}
			status := "CREATING"
			if describeCalls == 2 {
				status = "ACTIVE"
			}
			return describeSDKResponse(status, ""), nil
		},
	}
	logic := newCreateLogic(client, 100*time.Millisecond)

	response, err := logic.CreateSandboxTool(validCreateRequest())
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if !response.Accepted || response.Status != "ACTIVE" || response.ToolId != "sdt-1" || response.RequestId != "request-1" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if describeCalls != 2 {
		t.Fatalf("describe calls = %d, want 2", describeCalls)
	}
}

func TestCreateSandboxToolReportsAsynchronousFailure(t *testing.T) {
	client := &sandboxToolClientMock{
		create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			return createSDKResponse("sdt-failed", "request-2"), nil
		},
		describe: func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
			return describeSDKResponse("FAILED", "image cannot be pulled"), nil
		},
	}
	logic := newCreateLogic(client, 100*time.Millisecond)

	response, err := logic.CreateSandboxTool(validCreateRequest())
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if !response.Accepted || response.Status != "FAILED" || response.StatusReason != "image cannot be pulled" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response.ErrorCode != "CREATE_SANDBOX_TOOL_FAILED" {
		t.Fatalf("ErrorCode = %q", response.ErrorCode)
	}
}

func TestCreateSandboxToolReturnsTencentCloudErrorDetails(t *testing.T) {
	client := &sandboxToolClientMock{
		create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			return nil, tcerr.NewTencentCloudSDKError("InvalidParameter", "invalid image", "request-error")
		},
		describe: func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
			t.Fatal("Describe should not be called")
			return nil, nil
		},
	}
	logic := newCreateLogic(client, 100*time.Millisecond)

	response, err := logic.CreateSandboxTool(validCreateRequest())
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if response.Accepted || response.Status != "REJECTED" || response.ErrorCode != "InvalidParameter" || response.RequestId != "request-error" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestCreateSandboxToolToleratesTransientDescribeError(t *testing.T) {
	describeCalls := 0
	client := &sandboxToolClientMock{
		create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			return createSDKResponse("sdt-2", "request-3"), nil
		},
		describe: func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
			describeCalls++
			if describeCalls == 1 {
				return nil, errors.New("temporary network error")
			}
			return describeSDKResponse("ACTIVE", ""), nil
		},
	}
	logic := newCreateLogic(client, 100*time.Millisecond)

	response, err := logic.CreateSandboxTool(validCreateRequest())
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if response.Status != "ACTIVE" || response.ErrorCode != "" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestCreateSandboxToolReturnsCreatingWhenStatusQueryKeepsFailing(t *testing.T) {
	client := &sandboxToolClientMock{
		create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			return createSDKResponse("sdt-3", "request-4"), nil
		},
		describe: func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
			return nil, errors.New("network unavailable")
		},
	}
	logic := newCreateLogic(client, 10*time.Millisecond)

	response, err := logic.CreateSandboxTool(validCreateRequest())
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if !response.Accepted || response.Status != "CREATING" || response.ErrorCode != "STATUS_QUERY_FAILED" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func newCreateLogic(client svc.SandboxToolClient, pollTimeout time.Duration) *CreateSandboxToolLogic {
	return NewCreateSandboxToolLogic(context.Background(), &svc.ServiceContext{
		Config: config.Config{TencentCloud: config.TencentCloudConfig{
			RequestTimeout:     time.Second,
			StatusPollInterval: time.Millisecond,
			StatusPollTimeout:  pollTimeout,
		}},
		SandboxToolClient: client,
	})
}

func createSDKResponse(toolID, requestID string) *tcags.CreateSandboxToolResponse {
	return &tcags.CreateSandboxToolResponse{Response: &tcags.CreateSandboxToolResponseParams{
		ToolId:    stringPtr(toolID),
		RequestId: stringPtr(requestID),
	}}
}

func describeSDKResponse(status, reason string) *tcags.DescribeSandboxToolListResponse {
	tool := &tcags.SandboxTool{Status: stringPtr(status)}
	if reason != "" {
		tool.StatusReason = stringPtr(reason)
	}
	return &tcags.DescribeSandboxToolListResponse{Response: &tcags.DescribeSandboxToolListResponseParams{
		SandboxToolSet: []*tcags.SandboxTool{tool},
	}}
}

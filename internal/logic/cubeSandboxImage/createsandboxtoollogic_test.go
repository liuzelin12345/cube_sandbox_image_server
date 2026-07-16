package cubeSandboxImage

import (
	"context"
	"errors"
	"reflect"
	"strings"
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

type imageCommandResolverMock struct {
	resolve func(context.Context, string) ([]string, error)
}

func (m *imageCommandResolverMock) ResolveCommand(ctx context.Context, image string) ([]string, error) {
	return m.resolve(ctx, image)
}

func TestCreateSandboxToolReturnsCreateResultWithoutStatusQuery(t *testing.T) {
	client := &sandboxToolClientMock{
		create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			return createSDKResponse("sdt-1", "request-1"), nil
		},
		describe: func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
			t.Fatal("DescribeSandboxToolList must not be called by the create endpoint")
			return nil, nil
		},
	}
	logic := newCreateLogic(client)

	response, err := logic.CreateSandboxTool(validCreateRequest())
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if !response.Accepted || response.Status != "CREATING" || response.ToolId != "sdt-1" || response.RequestId != "request-1" {
		t.Fatalf("unexpected response: %#v", response)
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
	logic := newCreateLogic(client)

	response, err := logic.CreateSandboxTool(validCreateRequest())
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if response.Accepted || response.Status != "REJECTED" || response.ErrorCode != "InvalidParameter" || response.RequestId != "request-error" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestCreateSandboxToolUsesRegistryCommandWhenRequestOmitsCommand(t *testing.T) {
	resolveCalls := 0
	resolver := &imageCommandResolverMock{resolve: func(_ context.Context, image string) ([]string, error) {
		resolveCalls++
		if image != "registry.example.com/team/sandbox:latest" {
			t.Fatalf("ResolveCommand() image = %q", image)
		}
		return []string{"/image-entrypoint", "--serve"}, nil
	}}
	var createdCommand []string
	client := successfulSandboxToolClient(func(request *tcags.CreateSandboxToolRequest) {
		for _, value := range request.CustomConfiguration.Command {
			createdCommand = append(createdCommand, *value)
		}
	})
	request := validCreateRequest()
	request.CustomConfiguration.Command = nil
	logic := newCreateLogicWithResolver(client, resolver)

	response, err := logic.CreateSandboxTool(request)
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if !response.Accepted || response.Status != "CREATING" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if resolveCalls != 1 {
		t.Fatalf("ResolveCommand() calls = %d, want 1", resolveCalls)
	}
	if want := []string{"/image-entrypoint", "--serve"}; !reflect.DeepEqual(createdCommand, want) {
		t.Fatalf("created command = %#v, want %#v", createdCommand, want)
	}
}

func TestCreateSandboxToolStillResolvesRegistryWhenRequestOverridesCommand(t *testing.T) {
	resolveCalls := 0
	resolver := &imageCommandResolverMock{resolve: func(context.Context, string) ([]string, error) {
		resolveCalls++
		return []string{"/image-entrypoint"}, nil
	}}
	var createdCommand []string
	client := successfulSandboxToolClient(func(request *tcags.CreateSandboxToolRequest) {
		for _, value := range request.CustomConfiguration.Command {
			createdCommand = append(createdCommand, *value)
		}
	})
	request := validCreateRequest()
	request.CustomConfiguration.Command = []string{"/caller-command", "--override"}
	logic := newCreateLogicWithResolver(client, resolver)

	response, err := logic.CreateSandboxTool(request)
	if err != nil {
		t.Fatalf("CreateSandboxTool() error = %v", err)
	}
	if !response.Accepted || response.Status != "CREATING" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if resolveCalls != 1 {
		t.Fatalf("ResolveCommand() calls = %d, want 1", resolveCalls)
	}
	if want := []string{"/caller-command", "--override"}; !reflect.DeepEqual(createdCommand, want) {
		t.Fatalf("created command = %#v, want %#v", createdCommand, want)
	}
}

func TestCreateSandboxToolDoesNotCallTencentCloudWhenRegistryResolveFails(t *testing.T) {
	tests := []struct {
		name           string
		requestCommand []string
		resolveError   error
		wantCode       string
	}{
		{
			name:         "missing caller command",
			resolveError: errors.New("registry failed with registry-password"),
			wantCode:     "IMAGE_COMMAND_RESOLVE_FAILED",
		},
		{
			name:           "caller command provided",
			requestCommand: []string{"/caller-command"},
			resolveError:   errors.New("registry timeout"),
			wantCode:       "IMAGE_COMMAND_RESOLVE_FAILED",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			createCalls := 0
			client := &sandboxToolClientMock{
				create: func(context.Context, *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
					createCalls++
					return nil, errors.New("must not be called")
				},
				describe: func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
					t.Fatal("DescribeSandboxToolList must not be called")
					return nil, nil
				},
			}
			resolver := &imageCommandResolverMock{resolve: func(context.Context, string) ([]string, error) {
				return nil, test.resolveError
			}}
			request := validCreateRequest()
			request.CustomConfiguration.Command = test.requestCommand
			logic := newCreateLogicWithResolver(client, resolver)

			response, err := logic.CreateSandboxTool(request)
			if err != nil {
				t.Fatalf("CreateSandboxTool() error = %v", err)
			}
			if response.Accepted || response.Status != "REJECTED" || response.ErrorCode != test.wantCode {
				t.Fatalf("unexpected response: %#v", response)
			}
			if response.ErrorMessage == "" || strings.Contains(response.ErrorMessage, "registry-password") {
				t.Fatalf("unsafe response error message: %q", response.ErrorMessage)
			}
			if createCalls != 0 {
				t.Fatalf("CreateSandboxTool calls = %d, want 0", createCalls)
			}
		})
	}
}

func newCreateLogic(client svc.SandboxToolClient) *CreateSandboxToolLogic {
	resolver := &imageCommandResolverMock{resolve: func(context.Context, string) ([]string, error) {
		return []string{"/image-entrypoint"}, nil
	}}
	return newCreateLogicWithResolver(client, resolver)
}

func newCreateLogicWithResolver(client svc.SandboxToolClient, resolver svc.ImageCommandResolver) *CreateSandboxToolLogic {
	return NewCreateSandboxToolLogic(context.Background(), &svc.ServiceContext{
		Config: config.Config{TencentCloud: config.TencentCloudConfig{
			RequestTimeout: time.Second,
		}},
		SandboxToolClient:    client,
		ImageCommandResolver: resolver,
	})
}

func successfulSandboxToolClient(inspect func(*tcags.CreateSandboxToolRequest)) *sandboxToolClientMock {
	return &sandboxToolClientMock{
		create: func(_ context.Context, request *tcags.CreateSandboxToolRequest) (*tcags.CreateSandboxToolResponse, error) {
			if inspect != nil {
				inspect(request)
			}
			return createSDKResponse("sdt-success", "request-success"), nil
		},
		describe: func(context.Context, *tcags.DescribeSandboxToolListRequest) (*tcags.DescribeSandboxToolListResponse, error) {
			panic("DescribeSandboxToolList must not be called by the create endpoint")
		},
	}
}

func createSDKResponse(toolID, requestID string) *tcags.CreateSandboxToolResponse {
	return &tcags.CreateSandboxToolResponse{Response: &tcags.CreateSandboxToolResponseParams{
		ToolId:    stringPtr(toolID),
		RequestId: stringPtr(requestID),
	}}
}

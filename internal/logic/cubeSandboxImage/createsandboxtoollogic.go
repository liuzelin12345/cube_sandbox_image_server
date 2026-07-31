// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package cubeSandboxImage

import (
	"context"
	"errors"
	"strings"
	"time"

	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/apperror"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSandboxToolLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建腾讯云自定义沙箱工具
func NewCreateSandboxToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSandboxToolLogic {
	return &CreateSandboxToolLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 向腾讯云发送请求
func (l *CreateSandboxToolLogic) CreateSandboxTool(req *types.CreateSandboxToolRequest) (resp *types.CreateSandboxToolResponse, err error) {
	if req == nil {
		return rejectedResponse("INVALID_ARGUMENT", "request body is required", ""), nil
	}
	if strings.TrimSpace(req.CustomConfiguration.Image) == "" {
		return rejectedResponse("INVALID_ARGUMENT", "customConfiguration.image is required", ""), nil
	}
	if l.svcCtx.ImageCommandResolver == nil {
		return nil, apperror.New(500, "IMAGE_COMMAND_RESOLVER_UNAVAILABLE", "镜像启动命令解析器未初始化", nil)
	}

	defaultCommand := config.DefaultSandboxCommand()
	if len(defaultCommand) == 0 {
		defaultCommand, err = l.svcCtx.ImageCommandResolver.ResolveCommand(l.ctx, req.CustomConfiguration.Image)
		if err != nil {
			l.Errorf("resolve default image command failed")
			return rejectedResponse("IMAGE_COMMAND_RESOLVE_FAILED", "获取镜像默认启动命令失败", ""), nil
		}
	}

	effectiveRequest := *req
	effectiveRequest.CustomConfiguration = req.CustomConfiguration
	if len(effectiveRequest.CustomConfiguration.Command) == 0 {
		effectiveRequest.CustomConfiguration.Command = defaultCommand
	}
	request, err := buildCreateSandboxToolRequest(&effectiveRequest)
	if err != nil {
		return rejectedResponse("INVALID_ARGUMENT", err.Error(), ""), nil
	}
	if l.svcCtx.SandboxToolClient == nil {
		return nil, apperror.New(500, "AGS_CLIENT_UNAVAILABLE", "腾讯云 AGS 客户端未初始化", nil)
	}

	createCtx, cancel := withOptionalTimeout(l.ctx, l.svcCtx.Config.TencentCloud.RequestTimeout)
	response, err := l.svcCtx.SandboxToolClient.CreateSandboxToolWithContext(createCtx, request)
	cancel()
	if err != nil {
		return createErrorResponse(err), nil
	}
	if response == nil || response.Response == nil || response.Response.ToolId == nil || strings.TrimSpace(*response.Response.ToolId) == "" {
		return rejectedResponse("INVALID_CLOUD_RESPONSE", "腾讯云 CreateSandboxTool 返回了无效响应", ""), nil
	}

	toolID := strings.TrimSpace(*response.Response.ToolId)
	requestID := ""
	if response.Response.RequestId != nil {
		requestID = *response.Response.RequestId
	}
	result := &types.CreateSandboxToolResponse{
		Accepted:  true,
		Status:    "CREATING",
		ToolId:    toolID,
		RequestId: requestID,
	}
	return result, nil
}

func createErrorResponse(err error) *types.CreateSandboxToolResponse {
	var sdkErr *tcerr.TencentCloudSDKError
	if errors.As(err, &sdkErr) {
		return rejectedResponse(sdkErr.GetCode(), sdkErr.GetMessage(), sdkErr.GetRequestId())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return rejectedResponse("TENCENTCLOUD_TIMEOUT", "调用腾讯云 CreateSandboxTool 超时；如已设置 clientToken，可使用同一 token 安全重试", "")
	}
	if errors.Is(err, context.Canceled) {
		return rejectedResponse("REQUEST_CANCELED", "创建请求已取消", "")
	}
	return rejectedResponse("CREATE_SANDBOX_TOOL_ERROR", err.Error(), "")
}

func rejectedResponse(code, message, requestID string) *types.CreateSandboxToolResponse {
	return &types.CreateSandboxToolResponse{
		Accepted:     false,
		Status:       "REJECTED",
		ErrorCode:    code,
		ErrorMessage: message,
		RequestId:    requestID,
	}
}

func withOptionalTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

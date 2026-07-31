// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package cubeSandboxImage

import (
	"context"
	"errors"
	"net/http"
	"strings"

	tcags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/apperror"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSandboxToolStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询腾讯云自定义沙箱工具状态
func NewGetSandboxToolStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSandboxToolStatusLogic {
	return &GetSandboxToolStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSandboxToolStatusLogic) GetSandboxToolStatus(req *types.GetSandboxToolStatusRequest) (resp *types.GetSandboxToolStatusResponse, err error) {
	if req == nil || strings.TrimSpace(req.ToolId) == "" {
		return nil, apperror.New(http.StatusBadRequest, "INVALID_ARGUMENT", "toolId is required", nil)
	}
	if l.svcCtx.SandboxToolClient == nil {
		return nil, apperror.New(http.StatusInternalServerError, "AGS_CLIENT_UNAVAILABLE", "腾讯云 AGS 客户端未初始化", nil)
	}

	toolID := strings.TrimSpace(req.ToolId)
	request := tcags.NewDescribeSandboxToolListRequest()
	request.ToolIds = []*string{stringPtr(toolID)}

	queryCtx, cancel := withOptionalTimeout(l.ctx, l.svcCtx.Config.TencentCloud.RequestTimeout)
	defer cancel()
	response, err := l.svcCtx.SandboxToolClient.DescribeSandboxToolListWithContext(queryCtx, request)
	if err != nil {
		l.Errorf("describe sandbox tool failed: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, apperror.New(http.StatusGatewayTimeout, "TENCENTCLOUD_TIMEOUT", "查询腾讯云沙箱工具状态超时", err)
		}
		return nil, apperror.New(http.StatusBadGateway, "STATUS_QUERY_FAILED", "查询腾讯云沙箱工具状态失败", err)
	}
	if response == nil || response.Response == nil {
		return nil, apperror.New(http.StatusBadGateway, "INVALID_CLOUD_RESPONSE", "腾讯云 DescribeSandboxToolList 返回了无效响应", nil)
	}

	tool := findSandboxTool(response.Response.SandboxToolSet, toolID)
	if tool == nil {
		return nil, apperror.New(http.StatusNotFound, "TOOL_NOT_FOUND", "未找到指定的沙箱工具", nil)
	}
	if tool.Status == nil || strings.TrimSpace(*tool.Status) == "" {
		return nil, apperror.New(http.StatusBadGateway, "INVALID_CLOUD_RESPONSE", "腾讯云返回的沙箱工具状态为空", nil)
	}

	result := &types.GetSandboxToolStatusResponse{
		ToolId: toolID,
		Status: strings.ToUpper(strings.TrimSpace(*tool.Status)),
	}
	if tool.StatusReason != nil {
		result.StatusReason = strings.TrimSpace(*tool.StatusReason)
	}
	if response.Response.RequestId != nil {
		result.RequestId = strings.TrimSpace(*response.Response.RequestId)
	}
	return result, nil
}

func findSandboxTool(tools []*tcags.SandboxTool, toolID string) *tcags.SandboxTool {
	for _, tool := range tools {
		if tool == nil || tool.ToolId == nil {
			continue
		}
		if strings.TrimSpace(*tool.ToolId) == toolID {
			return tool
		}
	}
	return nil
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package cubeSandboxImage

import (
	"net/http"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/apperror"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/logic/cubeSandboxImage"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 创建腾讯云自定义沙箱工具
func CreateSandboxToolHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateSandboxToolRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cubeSandboxImage.NewCreateSandboxToolLogic(r.Context(), svcCtx)
		resp, err := l.CreateSandboxTool(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err, apperror.Write)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

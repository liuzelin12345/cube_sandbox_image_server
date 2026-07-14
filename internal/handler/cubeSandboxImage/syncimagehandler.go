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

// 将镜像从源镜像仓库同步到目标镜像仓库
func SyncImageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SyncImageRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cubeSandboxImage.NewSyncImageLogic(r.Context(), svcCtx)
		resp, err := l.SyncImage(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err, apperror.Write)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

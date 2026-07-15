// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"net/http"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/logic/health"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Kubernetes Readiness 探针
func ReadinessHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := health.NewReadinessLogic(r.Context(), svcCtx)
		err := l.Readiness()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}

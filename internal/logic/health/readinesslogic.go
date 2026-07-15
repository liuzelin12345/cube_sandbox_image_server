// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"context"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReadinessLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Kubernetes Readiness 探针
func NewReadinessLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadinessLogic {
	return &ReadinessLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadinessLogic) Readiness() error {
	return nil
}

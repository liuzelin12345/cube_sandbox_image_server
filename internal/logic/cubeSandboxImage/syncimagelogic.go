// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package cubeSandboxImage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/apperror"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/client/imagesync"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SyncImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 将镜像从源镜像仓库同步到目标镜像仓库
func NewSyncImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncImageLogic {
	return &SyncImageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SyncImageLogic) SyncImage(req *types.SyncImageRequest) (resp *types.SyncImageResponse, err error) {
	request, err := buildSyncImageRequest(req)
	if err != nil {
		return nil, apperror.New(http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), err)
	}
	if !isAllowedImageName(request.Image, l.svcCtx.Config.ImageSync.AllowedImageNames) {
		return nil, apperror.New(http.StatusForbidden, "IMAGE_NOT_ALLOWED", "镜像名不在同步白名单中", nil)
	}
	if l.svcCtx.ImageSyncClient == nil {
		return nil, apperror.New(http.StatusInternalServerError, "IMAGE_SYNC_CLIENT_UNAVAILABLE", "镜像同步客户端未初始化", nil)
	}

	requestCtx, cancel := withOptionalTimeout(l.ctx, l.svcCtx.Config.ImageSync.RequestTimeout)
	defer cancel()
	result, err := l.svcCtx.ImageSyncClient.Sync(requestCtx, request)
	if err != nil {
		l.Errorf("image sync request failed: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, apperror.New(http.StatusGatewayTimeout, "IMAGE_SYNC_TIMEOUT", "镜像同步服务请求超时", err)
		}
		return nil, apperror.New(http.StatusBadGateway, "IMAGE_SYNC_FAILED", "镜像同步服务请求失败", err)
	}
	if result == nil {
		return nil, apperror.New(http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "镜像同步服务返回了无效响应", nil)
	}

	return &types.SyncImageResponse{
		Code: result.Code,
		Data: types.SyncImageData{
			Image:   result.Data.Image,
			NewSync: result.Data.NewSync,
			Time:    result.Data.Time,
		},
		Status: result.Status,
	}, nil
}

func isAllowedImageName(imageName string, allowedImageNames []string) bool {
	for _, allowedImageName := range allowedImageNames {
		if imageName == allowedImageName {
			return true
		}
	}
	return false
}

func buildSyncImageRequest(input *types.SyncImageRequest) (imagesync.Request, error) {
	if input == nil {
		return imagesync.Request{}, fmt.Errorf("request parameters are required")
	}

	fields := []struct {
		name   string
		value  *string
		maxLen int
	}{
		{name: "srcHub", value: &input.SrcHub, maxLen: 512},
		{name: "dstHub", value: &input.DstHub, maxLen: 512},
		{name: "group", value: &input.Group, maxLen: 256},
		{name: "image", value: &input.Image, maxLen: 1024},
		{name: "tag", value: &input.Tag, maxLen: 256},
	}
	for _, field := range fields {
		*field.value = strings.TrimSpace(*field.value)
		if *field.value == "" {
			return imagesync.Request{}, fmt.Errorf("%s is required", field.name)
		}
		if len(*field.value) > field.maxLen {
			return imagesync.Request{}, fmt.Errorf("%s cannot exceed %d bytes", field.name, field.maxLen)
		}
	}

	return imagesync.Request{
		SrcHub: input.SrcHub,
		DstHub: input.DstHub,
		Group:  input.Group,
		Image:  input.Image,
		Tag:    input.Tag,
	}, nil
}

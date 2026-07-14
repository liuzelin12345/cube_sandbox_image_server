package cubeSandboxImage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/apperror"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/client/imagesync"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/config"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/types"
)

type imageSyncClientMock struct {
	sync func(context.Context, imagesync.Request) (*imagesync.Response, error)
}

func (m *imageSyncClientMock) Sync(ctx context.Context, req imagesync.Request) (*imagesync.Response, error) {
	return m.sync(ctx, req)
}

func TestSyncImageMapsUpstreamResponse(t *testing.T) {
	client := &imageSyncClientMock{sync: func(_ context.Context, req imagesync.Request) (*imagesync.Response, error) {
		if req.SrcHub != "hub-dev.example.com:9544" || req.Image != "cube-sandbox/sandbox-code" {
			t.Fatalf("unexpected request: %#v", req)
		}
		return &imagesync.Response{
			Code:   0,
			Data:   imagesync.Data{Image: "target.example.com/ths/cube-sandbox/sandbox-code:latest", NewSync: false, Time: 1},
			Status: "success",
		}, nil
	}}
	logic := newSyncLogic(client, time.Second)

	response, err := logic.SyncImage(validSyncRequest())
	if err != nil {
		t.Fatalf("SyncImage() error = %v", err)
	}
	if response.Code != 0 || response.Status != "success" || response.Data.Time != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestSyncImageValidatesRequiredFields(t *testing.T) {
	logic := newSyncLogic(&imageSyncClientMock{sync: func(context.Context, imagesync.Request) (*imagesync.Response, error) {
		t.Fatal("Sync should not be called")
		return nil, nil
	}}, time.Second)
	request := validSyncRequest()
	request.Tag = "  "

	_, err := logic.SyncImage(request)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.StatusCode != 400 || appErr.Code != "INVALID_ARGUMENT" {
		t.Fatalf("SyncImage() error = %#v", err)
	}
}

func TestSyncImageReturnsGatewayTimeout(t *testing.T) {
	client := &imageSyncClientMock{sync: func(ctx context.Context, _ imagesync.Request) (*imagesync.Response, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	logic := newSyncLogic(client, 5*time.Millisecond)

	_, err := logic.SyncImage(validSyncRequest())
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.StatusCode != 504 || appErr.Code != "IMAGE_SYNC_TIMEOUT" {
		t.Fatalf("SyncImage() error = %#v", err)
	}
}

func newSyncLogic(client svc.ImageSyncClient, timeout time.Duration) *SyncImageLogic {
	return NewSyncImageLogic(context.Background(), &svc.ServiceContext{
		Config:          config.Config{ImageSync: config.ImageSyncConfig{RequestTimeout: timeout}},
		ImageSyncClient: client,
	})
}

func validSyncRequest() *types.SyncImageRequest {
	return &types.SyncImageRequest{
		SrcHub: "hub-dev.example.com:9544",
		DstHub: "target.example.com",
		Group:  "ths",
		Image:  "cube-sandbox/sandbox-code",
		Tag:    "latest",
	}
}

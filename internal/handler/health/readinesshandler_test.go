package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/svc"
)

func TestReadinessHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readiness", nil)

	ReadinessHandler(&svc.ServiceContext{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("response body = %q, want empty body", recorder.Body.String())
	}
}

package imagesync

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientSyncEncodesQueryAndDecodesResponse(t *testing.T) {
	want := Request{
		SrcHub: "hub.example.com:9544",
		DstHub: "registry.example.com",
		Group:  "team & platform",
		Image:  "sandbox/code interpreter",
		Tag:    "release+1",
	}
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		query := r.URL.Query()
		if query.Get("srcHub") != want.SrcHub || query.Get("dstHub") != want.DstHub ||
			query.Get("group") != want.Group || query.Get("image") != want.Image || query.Get("tag") != want.Tag {
			t.Fatalf("unexpected query: %#v", query)
		}
		return testHTTPResponse(http.StatusOK, `{"code":0,"data":{"image":"registry.example.com/team/sandbox/code:release+1","newSync":false,"time":1},"status":"success"}`), nil
	})

	client := newTestClient(t, transport, 1, 0, 1<<20)
	response, err := client.Sync(context.Background(), want)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if response.Code != 0 || response.Status != "success" || response.Data.Time != 1 || response.Data.NewSync {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestClientSyncRetriesTransientStatus(t *testing.T) {
	var calls atomic.Int32
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		if calls.Add(1) < 3 {
			return testHTTPResponse(http.StatusServiceUnavailable, "temporary failure"), nil
		}
		return testHTTPResponse(http.StatusOK, `{"code":0,"data":{"image":"target/image:tag","newSync":true,"time":2},"status":"success"}`), nil
	})

	client := newTestClient(t, transport, 3, time.Millisecond, 1<<20)
	response, err := client.Sync(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
	if !response.Data.NewSync {
		t.Fatalf("expected newSync=true")
	}
}

func TestClientSyncDoesNotRetryClientError(t *testing.T) {
	var calls atomic.Int32
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return testHTTPResponse(http.StatusBadRequest, "bad request"), nil
	})

	client := newTestClient(t, transport, 3, 0, 1<<20)
	_, err := client.Sync(context.Background(), Request{})
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("Sync() error = %v, want HTTP 400", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestClientSyncRejectsOversizedAndInvalidResponses(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		bodyLimit int64
		wantError string
	}{
		{name: "oversized", body: strings.Repeat("x", 65), bodyLimit: 64, wantError: "exceeds 64 bytes"},
		{name: "invalid json", body: `{not-json}`, bodyLimit: 1 << 20, wantError: "decode image sync response"},
		{name: "multiple json values", body: `{}` + `{}`, bodyLimit: 1 << 20, wantError: "multiple JSON values"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
				return testHTTPResponse(http.StatusOK, tt.body), nil
			})
			client := newTestClient(t, transport, 1, 0, tt.bodyLimit)
			_, err := client.Sync(context.Background(), Request{})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Sync() error = %v, want substring %q", err, tt.wantError)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newTestClient(t *testing.T, transport http.RoundTripper, maxAttempts int, retryInterval time.Duration, bodyLimit int64) *Client {
	t.Helper()
	client, err := NewClient(
		"http://image-sync.internal/sync/image",
		&http.Client{Transport: transport, Timeout: time.Second},
		bodyLimit,
		maxAttempts,
		retryInterval,
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

package registrycommand

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const (
	testRegistry   = "registry.test"
	testRepository = "team/sandbox"
	testUsername   = "registry-user"
	testPassword   = "registry-password"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type registryFixture struct {
	manifest       []byte
	manifestDigest v1.Hash
	config         []byte
	configDigest   v1.Hash
}

func TestResolveCommandPrefersEntrypointAndUsesBasicAuth(t *testing.T) {
	fixture := newRegistryFixture(t, []string{"/entrypoint", "--serve"}, []string{"fallback"})
	transport := fixture.basicTransport(t, "latest")
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	command, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest")
	if err != nil {
		t.Fatalf("ResolveCommand() error = %v", err)
	}
	if want := []string{"/entrypoint", "--serve"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("ResolveCommand() = %#v, want %#v", command, want)
	}
}

func TestResolveCommandFallsBackToCmdAndSupportsDigest(t *testing.T) {
	fixture := newRegistryFixture(t, nil, []string{"/bin/sh", "-c", "run"})
	reference := fmt.Sprintf("%s/%s@%s", testRegistry, testRepository, fixture.manifestDigest)
	client := newTestClient(t, []string{testRegistry}, time.Second, fixture.basicTransport(t, fixture.manifestDigest.String()))

	command, err := client.ResolveCommand(context.Background(), reference)
	if err != nil {
		t.Fatalf("ResolveCommand() error = %v", err)
	}
	if want := []string{"/bin/sh", "-c", "run"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("ResolveCommand() = %#v, want %#v", command, want)
	}
}

func TestResolveCommandSupportsBearerAuth(t *testing.T) {
	fixture := newRegistryFixture(t, []string{"/bearer-entrypoint"}, nil)
	transport := fixture.bearerTransport(t)
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	command, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest")
	if err != nil {
		t.Fatalf("ResolveCommand() error = %v", err)
	}
	if want := []string{"/bearer-entrypoint"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("ResolveCommand() = %#v, want %#v", command, want)
	}
}

func TestResolveCommandFollowsCrossHostHTTPSBlobRedirectWithoutCredentials(t *testing.T) {
	fixture := newRegistryFixture(t, []string{"/redirected-entrypoint"}, nil)
	redirectCalls := 0
	transport := fixture.redirectingBearerTransport(
		"https://object-storage.test/config?temporary-signature=redacted",
		&redirectCalls,
	)
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	command, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest")
	if err != nil {
		t.Fatalf("ResolveCommand() error = %v", err)
	}
	if want := []string{"/redirected-entrypoint"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("ResolveCommand() = %#v, want %#v", command, want)
	}
	if redirectCalls != 1 {
		t.Fatalf("object storage calls = %d, want 1", redirectCalls)
	}
}

func TestResolveCommandRejectsCrossHostHTTPBlobRedirect(t *testing.T) {
	fixture := newRegistryFixture(t, []string{"/entrypoint"}, nil)
	redirectCalls := 0
	transport := fixture.redirectingBearerTransport(
		"http://object-storage.test/config?temporary-signature=redacted",
		&redirectCalls,
	)
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	if _, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest"); err == nil {
		t.Fatal("ResolveCommand() error = nil")
	}
	if redirectCalls != 0 {
		t.Fatalf("insecure object storage calls = %d, want 0", redirectCalls)
	}
}

func TestResolveCommandSelectsLinuxAMD64FromIndex(t *testing.T) {
	amd64 := newRegistryFixture(t, []string{"/amd64-entrypoint"}, nil)
	arm64 := newRegistryFixture(t, []string{"/arm64-entrypoint"}, nil)
	indexBody := mustJSON(t, v1.IndexManifest{
		SchemaVersion: 2,
		MediaType:     types.OCIImageIndex,
		Manifests: []v1.Descriptor{
			{
				MediaType: types.OCIManifestSchema1,
				Size:      int64(len(arm64.manifest)),
				Digest:    arm64.manifestDigest,
				Platform:  &v1.Platform{OS: "linux", Architecture: "arm64"},
			},
			{
				MediaType: types.OCIManifestSchema1,
				Size:      int64(len(amd64.manifest)),
				Digest:    amd64.manifestDigest,
				Platform:  &v1.Platform{OS: "linux", Architecture: "amd64"},
			},
		},
	})
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/v2/":
			return testResponse(request, http.StatusOK, "", nil), nil
		case "/v2/" + testRepository + "/manifests/latest":
			return testResponse(request, http.StatusOK, string(types.OCIImageIndex), indexBody), nil
		case "/v2/" + testRepository + "/manifests/" + amd64.manifestDigest.String():
			return testResponse(request, http.StatusOK, string(types.OCIManifestSchema1), amd64.manifest), nil
		case "/v2/" + testRepository + "/blobs/" + amd64.configDigest.String():
			return testResponse(request, http.StatusOK, string(types.OCIConfigJSON), amd64.config), nil
		default:
			return testResponse(request, http.StatusNotFound, "application/json", nil), nil
		}
	})
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	command, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest")
	if err != nil {
		t.Fatalf("ResolveCommand() error = %v", err)
	}
	if want := []string{"/amd64-entrypoint"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("ResolveCommand() = %#v, want %#v", command, want)
	}
}

func TestResolveCommandRejectsIndexWithoutLinuxAMD64(t *testing.T) {
	arm64 := newRegistryFixture(t, []string{"/arm64-entrypoint"}, nil)
	indexBody := mustJSON(t, v1.IndexManifest{
		SchemaVersion: 2,
		MediaType:     types.OCIImageIndex,
		Manifests: []v1.Descriptor{{
			MediaType: types.OCIManifestSchema1,
			Size:      int64(len(arm64.manifest)),
			Digest:    arm64.manifestDigest,
			Platform:  &v1.Platform{OS: "linux", Architecture: "arm64"},
		}},
	})
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/v2/" {
			return testResponse(request, http.StatusOK, "", nil), nil
		}
		return testResponse(request, http.StatusOK, string(types.OCIImageIndex), indexBody), nil
	})
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	if _, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest"); err == nil {
		t.Fatal("ResolveCommand() error = nil")
	}
}

func TestResolveCommandRejectsMissingCommand(t *testing.T) {
	fixture := newRegistryFixture(t, nil, nil)
	client := newTestClient(t, []string{testRegistry}, time.Second, fixture.basicTransport(t, "latest"))

	if _, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest"); err == nil {
		t.Fatal("ResolveCommand() error = nil")
	}
}

func TestResolveCommandRejectsInvalidConfigJSON(t *testing.T) {
	config := []byte("not-json")
	configDigest, configSize, err := v1.SHA256(bytes.NewReader(config))
	if err != nil {
		t.Fatalf("hash config: %v", err)
	}
	manifest := mustJSON(t, v1.Manifest{
		SchemaVersion: 2,
		MediaType:     types.OCIManifestSchema1,
		Config: v1.Descriptor{
			MediaType: types.OCIConfigJSON,
			Size:      configSize,
			Digest:    configDigest,
		},
		Layers: []v1.Descriptor{},
	})
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/v2/":
			return testResponse(request, http.StatusOK, "", nil), nil
		case "/v2/" + testRepository + "/manifests/latest":
			return testResponse(request, http.StatusOK, string(types.OCIManifestSchema1), manifest), nil
		case "/v2/" + testRepository + "/blobs/" + configDigest.String():
			return testResponse(request, http.StatusOK, string(types.OCIConfigJSON), config), nil
		default:
			return testResponse(request, http.StatusNotFound, "application/json", nil), nil
		}
	})
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	if _, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest"); err == nil {
		t.Fatal("ResolveCommand() error = nil")
	}
}

func TestResolveCommandRejectsRegistryOutsideAllowlistBeforeNetwork(t *testing.T) {
	called := false
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("unexpected network call")
	})
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	_, err := client.ResolveCommand(context.Background(), "not-allowed.test/team/sandbox:latest")
	if !errors.Is(err, ErrRegistryNotAllowed) {
		t.Fatalf("ResolveCommand() error = %v, want ErrRegistryNotAllowed", err)
	}
	if called {
		t.Fatal("transport was called for a non-allowlisted registry")
	}
}

func TestResolveCommandNeverUsesInsecureRegistryTransport(t *testing.T) {
	var schemes []string
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		schemes = append(schemes, request.URL.Scheme)
		return nil, errors.New("unexpected network call")
	})
	client := newTestClient(t, []string{"localhost"}, time.Second, transport)

	if _, err := client.ResolveCommand(context.Background(), "localhost/team/sandbox:latest"); err == nil {
		t.Fatal("ResolveCommand() error = nil")
	}
	for _, scheme := range schemes {
		if scheme != "https" {
			t.Fatalf("transport used insecure scheme %q", scheme)
		}
	}
}

func TestResolveCommandDoesNotSendCredentialsToUnlistedBearerRealm(t *testing.T) {
	var hosts []string
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		hosts = append(hosts, request.URL.Host)
		if request.URL.Path == "/v2/" {
			header := http.Header{"Www-Authenticate": []string{`Bearer realm="https://auth.not-allowed.test/token",service="registry.test"`}}
			return testResponseWithHeader(request, http.StatusUnauthorized, header, nil), nil
		}
		return testResponse(request, http.StatusInternalServerError, "application/json", nil), nil
	})
	client := newTestClient(t, []string{testRegistry}, time.Second, transport)

	_, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest")
	if err == nil {
		t.Fatal("ResolveCommand() error = nil")
	}
	if len(hosts) != 1 || hosts[0] != testRegistry {
		t.Fatalf("transport hosts = %#v, credentials may have left the allowlist", hosts)
	}
}

func TestResolveCommandHandlesRegistryFailuresWithoutLeakingCredentials(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		transport http.RoundTripper
	}{
		{
			name:    "unauthorized",
			timeout: time.Second,
			transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				return testResponse(request, http.StatusUnauthorized, "application/json", []byte(`{"errors":[{"code":"UNAUTHORIZED"}]}`)), nil
			}),
		},
		{
			name:    "not found",
			timeout: time.Second,
			transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path == "/v2/" {
					return testResponse(request, http.StatusOK, "", nil), nil
				}
				return testResponse(request, http.StatusNotFound, "application/json", []byte(`{"errors":[{"code":"MANIFEST_UNKNOWN"}]}`)), nil
			}),
		},
		{
			name:    "forbidden",
			timeout: time.Second,
			transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path == "/v2/" {
					return testResponse(request, http.StatusOK, "", nil), nil
				}
				return testResponse(request, http.StatusForbidden, "application/json", []byte(`{"errors":[{"code":"DENIED"}]}`)), nil
			}),
		},
		{
			name:    "server error",
			timeout: 20 * time.Millisecond,
			transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path == "/v2/" {
					return testResponse(request, http.StatusOK, "", nil), nil
				}
				return testResponse(request, http.StatusInternalServerError, "application/json", nil), nil
			}),
		},
		{
			name:    "timeout",
			timeout: 10 * time.Millisecond,
			transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				<-request.Context().Done()
				return nil, request.Context().Err()
			}),
		},
		{
			name:    "invalid manifest",
			timeout: time.Second,
			transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path == "/v2/" {
					return testResponse(request, http.StatusOK, "", nil), nil
				}
				return testResponse(request, http.StatusOK, string(types.OCIManifestSchema1), []byte("not-json")), nil
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(t, []string{testRegistry}, test.timeout, test.transport)
			_, err := client.ResolveCommand(context.Background(), testRegistry+"/"+testRepository+":latest")
			if err == nil {
				t.Fatal("ResolveCommand() error = nil")
			}
			if strings.Contains(err.Error(), testUsername) || strings.Contains(err.Error(), testPassword) {
				t.Fatalf("ResolveCommand() leaked credentials: %v", err)
			}
		})
	}
}

func TestNewClientValidatesConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		hosts    []string
		username string
		password string
		timeout  time.Duration
	}{
		{name: "missing username", hosts: []string{testRegistry}, password: testPassword, timeout: time.Second},
		{name: "missing password", hosts: []string{testRegistry}, username: testUsername, timeout: time.Second},
		{name: "missing hosts", username: testUsername, password: testPassword, timeout: time.Second},
		{name: "invalid host", hosts: []string{"https://registry.test"}, username: testUsername, password: testPassword, timeout: time.Second},
		{name: "invalid timeout", hosts: []string{testRegistry}, username: testUsername, password: testPassword},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewClient(test.hosts, test.username, test.password, test.timeout, http.DefaultTransport); err == nil {
				t.Fatal("NewClient() error = nil")
			}
		})
	}
}

func newTestClient(t *testing.T, hosts []string, timeout time.Duration, transport http.RoundTripper) *Client {
	t.Helper()
	client, err := NewClient(hosts, testUsername, testPassword, timeout, transport)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func newRegistryFixture(t *testing.T, entrypoint, command []string) registryFixture {
	t.Helper()
	config := mustJSON(t, v1.ConfigFile{
		Architecture: "amd64",
		OS:           "linux",
		RootFS:       v1.RootFS{Type: "layers", DiffIDs: []v1.Hash{}},
		Config:       v1.Config{Entrypoint: entrypoint, Cmd: command},
	})
	configDigest, configSize, err := v1.SHA256(bytes.NewReader(config))
	if err != nil {
		t.Fatalf("hash config: %v", err)
	}
	manifest := mustJSON(t, v1.Manifest{
		SchemaVersion: 2,
		MediaType:     types.OCIManifestSchema1,
		Config: v1.Descriptor{
			MediaType: types.OCIConfigJSON,
			Size:      configSize,
			Digest:    configDigest,
		},
		Layers: []v1.Descriptor{},
	})
	manifestDigest, _, err := v1.SHA256(bytes.NewReader(manifest))
	if err != nil {
		t.Fatalf("hash manifest: %v", err)
	}
	return registryFixture{
		manifest:       manifest,
		manifestDigest: manifestDigest,
		config:         config,
		configDigest:   configDigest,
	}
}

func (f registryFixture) basicTransport(t *testing.T, manifestIdentifier string) http.RoundTripper {
	t.Helper()
	return roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/v2/" {
			header := http.Header{"Www-Authenticate": []string{`Basic realm="registry.test"`}}
			return testResponseWithHeader(request, http.StatusUnauthorized, header, nil), nil
		}
		username, password, ok := request.BasicAuth()
		if !ok || username != testUsername || password != testPassword {
			return nil, fmt.Errorf("missing registry basic authentication")
		}
		switch request.URL.Path {
		case "/v2/" + testRepository + "/manifests/" + manifestIdentifier:
			return testResponse(request, http.StatusOK, string(types.OCIManifestSchema1), f.manifest), nil
		case "/v2/" + testRepository + "/blobs/" + f.configDigest.String():
			return testResponse(request, http.StatusOK, string(types.OCIConfigJSON), f.config), nil
		default:
			return testResponse(request, http.StatusNotFound, "application/json", nil), nil
		}
	})
}

func (f registryFixture) bearerTransport(t *testing.T) http.RoundTripper {
	t.Helper()
	return roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/v2/":
			header := http.Header{"Www-Authenticate": []string{`Bearer realm="https://registry.test/token",service="registry.test"`}}
			return testResponseWithHeader(request, http.StatusUnauthorized, header, nil), nil
		case "/token":
			username, password, ok := request.BasicAuth()
			if !ok || username != testUsername || password != testPassword {
				return nil, fmt.Errorf("missing token endpoint basic authentication")
			}
			return testResponse(request, http.StatusOK, "application/json", []byte(`{"token":"bearer-token"}`)), nil
		case "/v2/" + testRepository + "/manifests/latest":
			if request.Header.Get("Authorization") != "Bearer bearer-token" {
				return nil, fmt.Errorf("missing bearer authentication")
			}
			return testResponse(request, http.StatusOK, string(types.OCIManifestSchema1), f.manifest), nil
		case "/v2/" + testRepository + "/blobs/" + f.configDigest.String():
			if request.Header.Get("Authorization") != "Bearer bearer-token" {
				return nil, fmt.Errorf("missing bearer authentication")
			}
			return testResponse(request, http.StatusOK, string(types.OCIConfigJSON), f.config), nil
		default:
			return testResponse(request, http.StatusNotFound, "application/json", nil), nil
		}
	})
}

func (f registryFixture) redirectingBearerTransport(location string, redirectCalls *int) http.RoundTripper {
	return roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != testRegistry {
			(*redirectCalls)++
			if request.Header.Get("Authorization") != "" {
				return nil, fmt.Errorf("registry authorization leaked to redirected host")
			}
			return testResponse(request, http.StatusOK, string(types.OCIConfigJSON), f.config), nil
		}

		switch request.URL.Path {
		case "/v2/":
			header := http.Header{"Www-Authenticate": []string{`Bearer realm="https://registry.test/token",service="registry.test"`}}
			return testResponseWithHeader(request, http.StatusUnauthorized, header, nil), nil
		case "/token":
			username, password, ok := request.BasicAuth()
			if !ok || username != testUsername || password != testPassword {
				return nil, fmt.Errorf("missing token endpoint basic authentication")
			}
			return testResponse(request, http.StatusOK, "application/json", []byte(`{"token":"bearer-token"}`)), nil
		case "/v2/" + testRepository + "/manifests/latest":
			if request.Header.Get("Authorization") != "Bearer bearer-token" {
				return nil, fmt.Errorf("missing bearer authentication")
			}
			return testResponse(request, http.StatusOK, string(types.OCIManifestSchema1), f.manifest), nil
		case "/v2/" + testRepository + "/blobs/" + f.configDigest.String():
			if request.Header.Get("Authorization") != "Bearer bearer-token" {
				return nil, fmt.Errorf("missing bearer authentication")
			}
			header := http.Header{"Location": []string{location}}
			return testResponseWithHeader(request, http.StatusTemporaryRedirect, header, nil), nil
		default:
			return testResponse(request, http.StatusNotFound, "application/json", nil), nil
		}
	})
}

func testResponse(request *http.Request, status int, contentType string, body []byte) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return testResponseWithHeader(request, status, header, body)
}

func testResponseWithHeader(request *http.Request, status int, header http.Header, body []byte) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       request,
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

package registrycommand

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var ErrRegistryNotAllowed = errors.New("image registry is not allowed")

type Client struct {
	allowedHosts   map[string]struct{}
	auth           authn.Authenticator
	requestTimeout time.Duration
	transport      http.RoundTripper
}

func NewClient(
	allowedHosts []string,
	username string,
	password string,
	requestTimeout time.Duration,
	transport http.RoundTripper,
) (*Client, error) {
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("registry username is required")
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("registry password is required")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("registry request timeout must be greater than zero")
	}
	if transport == nil {
		transport = http.DefaultTransport
	}

	hosts := make(map[string]struct{}, len(allowedHosts))
	for index, host := range allowedHosts {
		normalized := normalizeHost(host)
		if normalized == "" || strings.Contains(normalized, "://") || strings.ContainsAny(normalized, "/?#") {
			return nil, fmt.Errorf("registry allowedHosts[%d] is invalid", index)
		}
		hosts[normalized] = struct{}{}
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("at least one registry allowed host is required")
	}
	transport = &registryTransport{
		next:         transport,
		allowedHosts: hosts,
	}

	return &Client{
		allowedHosts: hosts,
		auth: &authn.Basic{
			Username: username,
			Password: password,
		},
		requestTimeout: requestTimeout,
		transport:      transport,
	}, nil
}

func (c *Client) ResolveCommand(ctx context.Context, imageReference string) ([]string, error) {
	reference, err := name.ParseReference(strings.TrimSpace(imageReference), name.StrictValidation)
	if err != nil {
		return nil, fmt.Errorf("parse image reference: %w", err)
	}
	if reference.Context().Registry.Scheme() != "https" {
		return nil, fmt.Errorf("image registry must use https")
	}

	host := normalizeHost(reference.Context().RegistryStr())
	if _, ok := c.allowedHosts[host]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrRegistryNotAllowed, host)
	}

	requestCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	image, err := remote.Image(
		reference,
		remote.WithAuth(c.auth),
		remote.WithContext(requestCtx),
		remote.WithPlatform(v1.Platform{OS: "linux", Architecture: "amd64"}),
		remote.WithTransport(c.transport),
	)
	if err != nil {
		return nil, fmt.Errorf("pull image metadata from registry %s failed", host)
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("read image config from registry %s failed", host)
	}
	if command := validCommand(configFile.Config.Entrypoint); len(command) > 0 {
		return command, nil
	}
	if command := validCommand(configFile.Config.Cmd); len(command) > 0 {
		return command, nil
	}

	return nil, fmt.Errorf("image config contains neither a valid Entrypoint nor Cmd")
}

func validCommand(command []string) []string {
	if len(command) == 0 {
		return nil
	}
	result := make([]string, len(command))
	for index, value := range command {
		if strings.TrimSpace(value) == "" {
			return nil
		}
		result[index] = value
	}
	return result
}

func normalizeHost(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
}

type registryTransport struct {
	next         http.RoundTripper
	allowedHosts map[string]struct{}
}

func (t *registryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, fmt.Errorf("registry request URL is invalid")
	}
	if !strings.EqualFold(request.URL.Scheme, "https") {
		return nil, fmt.Errorf("registry request must use https")
	}
	host := normalizeHost(request.URL.Host)
	if _, ok := t.allowedHosts[host]; ok {
		return t.next.RoundTrip(request)
	}

	// Registry implementations may redirect manifest/config blob downloads to
	// HTTPS object storage. go-containerregistry strips registry authorization
	// on cross-host redirects; enforce that boundary again before allowing the
	// unsigned read request to leave the registry allowlist.
	if request.URL.User != nil ||
		request.Header.Get("Authorization") != "" ||
		request.Header.Get("Proxy-Authorization") != "" {
		return nil, fmt.Errorf("cross-host registry request cannot include credentials")
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return nil, fmt.Errorf("cross-host registry request must be read-only")
	}
	return t.next.RoundTrip(request)
}

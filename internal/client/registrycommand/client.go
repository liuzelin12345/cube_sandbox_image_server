package registrycommand

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Client struct {
	auth           authn.Authenticator
	requestTimeout time.Duration
	transport      http.RoundTripper
}

func NewClient(
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
	transport = &httpsTransport{next: transport}

	return &Client{
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

type httpsTransport struct {
	next http.RoundTripper
}

func (t *httpsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, fmt.Errorf("registry request URL is invalid")
	}
	if !strings.EqualFold(request.URL.Scheme, "https") {
		return nil, fmt.Errorf("registry request must use https")
	}
	return t.next.RoundTrip(request)
}

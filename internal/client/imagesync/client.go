package imagesync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var errResponseTooLarge = errors.New("image sync response is too large")

type Request struct {
	SrcHub string
	DstHub string
	Group  string
	Image  string
	Tag    string
}

type Data struct {
	Image   string `json:"image"`
	NewSync bool   `json:"newSync"`
	Time    int64  `json:"time"`
}

type Response struct {
	Code   int64  `json:"code"`
	Data   Data   `json:"data"`
	Status string `json:"status"`
}

type Client struct {
	endpoint         *url.URL
	httpClient       *http.Client
	maxResponseBytes int64
	maxAttempts      int
	retryInterval    time.Duration
}

func NewClient(
	endpoint string,
	httpClient *http.Client,
	maxResponseBytes int64,
	maxAttempts int,
	retryInterval time.Duration,
) (*Client, error) {
	parsedEndpoint, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return nil, fmt.Errorf("parse image sync endpoint: %w", err)
	}
	if parsedEndpoint.Scheme != "http" && parsedEndpoint.Scheme != "https" {
		return nil, fmt.Errorf("image sync endpoint must use http or https")
	}
	if parsedEndpoint.Host == "" {
		return nil, fmt.Errorf("image sync endpoint host is required")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("image sync http client is required")
	}
	if maxResponseBytes <= 0 {
		return nil, fmt.Errorf("max response bytes must be greater than zero")
	}
	if maxAttempts <= 0 {
		return nil, fmt.Errorf("max attempts must be greater than zero")
	}
	if retryInterval < 0 {
		return nil, fmt.Errorf("retry interval cannot be negative")
	}

	return &Client{
		endpoint:         parsedEndpoint,
		httpClient:       httpClient,
		maxResponseBytes: maxResponseBytes,
		maxAttempts:      maxAttempts,
		retryInterval:    retryInterval,
	}, nil
}

func (c *Client) Sync(ctx context.Context, req Request) (*Response, error) {
	targetURL := c.buildURL(req)
	var lastErr error
	attempts := 0

	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		attempts = attempt
		response, retry, err := c.doRequest(ctx, targetURL)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if !retry || attempt == c.maxAttempts {
			break
		}
		if err := waitForRetry(ctx, c.retryDelay(attempt)); err != nil {
			return nil, fmt.Errorf("wait to retry image sync: %w", err)
		}
	}

	return nil, fmt.Errorf("image sync failed after %d attempt(s): %w", attempts, lastErr)
}

func (c *Client) buildURL(req Request) string {
	target := *c.endpoint
	query := target.Query()
	query.Set("srcHub", req.SrcHub)
	query.Set("dstHub", req.DstHub)
	query.Set("group", req.Group)
	query.Set("image", req.Image)
	query.Set("tag", req.Tag)
	target.RawQuery = query.Encode()
	return target.String()
}

func (c *Client) doRequest(ctx context.Context, targetURL string) (*Response, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("build image sync request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, fmt.Errorf("send image sync request: %w", err)
		}
		return nil, true, fmt.Errorf("send image sync request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readLimited(resp.Body, c.maxResponseBytes)
	if err != nil {
		retry := ctx.Err() == nil && !errors.Is(err, errResponseTooLarge)
		return nil, retry, fmt.Errorf("read image sync response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
		return nil, retry, fmt.Errorf("image sync upstream returned HTTP %d: %s", resp.StatusCode, compactBody(body))
	}

	var result Response
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&result); err != nil {
		return nil, false, fmt.Errorf("decode image sync response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, false, err
	}

	return &result, false, nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%w: response exceeds %d bytes", errResponseTooLarge, limit)
	}
	return body, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode image sync response: multiple JSON values")
		}
		return fmt.Errorf("decode image sync response: %w", err)
	}
	return nil
}

func compactBody(body []byte) string {
	const maxLength = 256
	message := strings.Join(strings.Fields(string(body)), " ")
	if len(message) > maxLength {
		return message[:maxLength] + "..."
	}
	return message
}

func (c *Client) retryDelay(attempt int) time.Duration {
	delay := c.retryInterval
	for i := 1; i < attempt && delay < 5*time.Second; i++ {
		delay *= 2
	}
	if delay > 5*time.Second {
		return 5 * time.Second
	}
	return delay
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

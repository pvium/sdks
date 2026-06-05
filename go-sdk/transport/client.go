package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
)

type HTTPClient struct {
	config     config.Config
	httpClient *http.Client
}

func NewHTTPClient(cfg config.Config) *HTTPClient {
	cfg = cfg.WithDefaults()
	return &HTTPClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

type Request struct {
	Method  string
	Path    string
	Body    any
	Query   map[string]string
	Options *models.RequestOptions
}

func (c *HTTPClient) Config() config.Config {
	return c.config
}

func (c *HTTPClient) setHeaders(req *http.Request, opts *models.RequestOptions, hasBody bool) {
	for k, v := range c.config.DefaultHeader {
		req.Header.Set(k, v)
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if opts != nil {
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}
		if opts.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+opts.AccessToken)
			return
		}
	}

	if opts != nil {
		if opts.SkipAPIKey {
			return
		}
		if opts.APIKey != "" {
			req.Header.Set("x-api-key", opts.APIKey)
			return
		}
	}
	if c.config.APIKey != "" {
		req.Header.Set("x-api-key", c.config.APIKey)
	}
}

func (c *HTTPClient) BuildURL(path string, query map[string]string) (string, error) {
	base, err := url.Parse(strings.TrimRight(c.config.BaseURL, "/"))
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(path, "/")
	q := base.Query()
	for k, v := range query {
		if v != "" {
			q.Set(k, v)
		}
	}
	base.RawQuery = q.Encode()
	return base.String(), nil
}

func (c *HTTPClient) Do(ctx context.Context, cfg Request) ([]byte, int, error) {
	var body io.Reader
	if cfg.Body != nil {
		buf, err := json.Marshal(cfg.Body)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(buf)
	}

	fullURL, err := c.BuildURL(cfg.Path, cfg.Query)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, cfg.Method, fullURL, body)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(req, cfg.Options, cfg.Body != nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return payload, resp.StatusCode, fmt.Errorf("pvium request failed: %s", resp.Status)
	}
	return payload, resp.StatusCode, nil
}

func Decode[T any](raw []byte) (T, error) {
	var out T
	if len(raw) == 0 {
		return out, nil
	}
	err := json.Unmarshal(raw, &out)
	return out, err
}

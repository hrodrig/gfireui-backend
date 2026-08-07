package gfire

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client is a minimal HTTP client for the GFire JSON API.
type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

// NewClient builds a GFire client from container-friendly config values.
func NewClient(baseURL, token string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	token = strings.TrimSpace(token)

	if baseURL == "" {
		return nil, errors.New("gfire base URL is required")
	}
	if token == "" {
		return nil, errors.New("gfire token is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse gfire base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("gfire base URL must include scheme and host")
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    parsed,
		token:      token,
		httpClient: httpClient,
	}, nil
}

// Do sends a request to GFire using the configured service bearer token.
func (c *Client) Do(ctx context.Context, method, requestPath string, body io.Reader) (*http.Response, error) {
	if c == nil {
		return nil, errors.New("gfire client is nil")
	}

	method = strings.TrimSpace(method)
	if method == "" {
		return nil, errors.New("http method is required")
	}

	requestURL, err := c.resolveURL(requestPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build gfire request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do gfire request: %w", err)
	}
	return resp, nil
}

func (c *Client) resolveURL(requestPath string) (*url.URL, error) {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		requestPath = "/"
	}

	relativeURL, err := url.Parse(requestPath)
	if err != nil {
		return nil, fmt.Errorf("parse gfire path: %w", err)
	}
	if relativeURL.IsAbs() {
		return nil, errors.New("gfire path must be relative to base URL")
	}

	resolved := *c.baseURL
	resolved.Path = joinURLPath(c.baseURL.Path, relativeURL.Path)
	resolved.RawPath = ""
	resolved.RawQuery = relativeURL.RawQuery
	resolved.Fragment = ""
	return &resolved, nil
}

func joinURLPath(basePath, relativePath string) string {
	switch {
	case basePath == "" && relativePath == "":
		return "/"
	case basePath == "":
		return ensureLeadingSlash(relativePath)
	case relativePath == "":
		return basePath
	default:
		return strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(relativePath, "/")
	}
}

func ensureLeadingSlash(value string) string {
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

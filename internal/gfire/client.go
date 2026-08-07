package gfire

import (
	"bytes"
	"context"
	"encoding/json"
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

// QueueSummary captures the minimal queue shape needed by the ops dashboard.
type QueueSummary struct {
	Name  string `json:"name"`
	Depth int    `json:"depth"`
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

// ListQueues returns the queue names and depths reported by GFire.
func (c *Client) ListQueues(ctx context.Context) ([]QueueSummary, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/v1/queues", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("gfire list queues failed: status %d", resp.StatusCode)
	}

	return decodeQueueSummaries(resp.Body)
}

// CountJobsByState returns the number of jobs GFire reports for a state filter.
func (c *Client) CountJobsByState(ctx context.Context, state string) (int, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return 0, errors.New("job state is required")
	}

	resp, err := c.Do(ctx, http.MethodGet, "/v1/jobs?state="+url.QueryEscape(state), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("gfire count jobs failed: status %d", resp.StatusCode)
	}

	return decodeJobCount(resp.Body)
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

type queuePayload struct {
	Name      string `json:"name"`
	Depth     *int   `json:"depth"`
	JobsCount *int   `json:"jobs_count"`
	Count     *int   `json:"count"`
}

type queueListPayload struct {
	Queues []queuePayload `json:"queues"`
	Data   []queuePayload `json:"data"`
	Items  []queuePayload `json:"items"`
	Result []queuePayload `json:"result"`
}

type jobsListPayload struct {
	Jobs      []json.RawMessage `json:"jobs"`
	Data      []json.RawMessage `json:"data"`
	Items     []json.RawMessage `json:"items"`
	Result    []json.RawMessage `json:"result"`
	Total     *int              `json:"total"`
	Count     *int              `json:"count"`
	JobsCount *int              `json:"jobs_count"`
}

func decodeQueueSummaries(body io.Reader) ([]QueueSummary, error) {
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read gfire queues response: %w", err)
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, errors.New("gfire queues response is empty")
	}

	summaries := make([]QueueSummary, 0)
	switch payload[0] {
	case '[':
		var items []queuePayload
		if err := json.Unmarshal(payload, &items); err != nil {
			return nil, fmt.Errorf("decode gfire queues response: %w", err)
		}
		summaries = append(summaries, mapQueueSummaries(items)...)
	default:
		var wrapper queueListPayload
		if err := json.Unmarshal(payload, &wrapper); err != nil {
			return nil, fmt.Errorf("decode gfire queues response: %w", err)
		}
		switch {
		case len(wrapper.Queues) > 0:
			summaries = append(summaries, mapQueueSummaries(wrapper.Queues)...)
		case len(wrapper.Data) > 0:
			summaries = append(summaries, mapQueueSummaries(wrapper.Data)...)
		case len(wrapper.Items) > 0:
			summaries = append(summaries, mapQueueSummaries(wrapper.Items)...)
		case len(wrapper.Result) > 0:
			summaries = append(summaries, mapQueueSummaries(wrapper.Result)...)
		default:
			var single queuePayload
			if err := json.Unmarshal(payload, &single); err == nil && single.Name != "" {
				summaries = append(summaries, mapQueueSummaries([]queuePayload{single})...)
			}
		}
	}

	if len(summaries) == 0 {
		return nil, errors.New("gfire queues response did not contain any queues")
	}

	return summaries, nil
}

func mapQueueSummaries(items []queuePayload) []QueueSummary {
	summaries := make([]QueueSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, QueueSummary{
			Name:  item.Name,
			Depth: firstInt(item.Depth, item.JobsCount, item.Count),
		})
	}
	return summaries
}

func decodeJobCount(body io.Reader) (int, error) {
	payload, err := io.ReadAll(body)
	if err != nil {
		return 0, fmt.Errorf("read gfire jobs response: %w", err)
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return 0, errors.New("gfire jobs response is empty")
	}

	switch payload[0] {
	case '[':
		var jobs []json.RawMessage
		if err := json.Unmarshal(payload, &jobs); err != nil {
			return 0, fmt.Errorf("decode gfire jobs response: %w", err)
		}
		return len(jobs), nil
	default:
		var wrapper jobsListPayload
		if err := json.Unmarshal(payload, &wrapper); err != nil {
			return 0, fmt.Errorf("decode gfire jobs response: %w", err)
		}
		switch {
		case len(wrapper.Jobs) > 0:
			return len(wrapper.Jobs), nil
		case len(wrapper.Data) > 0:
			return len(wrapper.Data), nil
		case len(wrapper.Items) > 0:
			return len(wrapper.Items), nil
		case len(wrapper.Result) > 0:
			return len(wrapper.Result), nil
		default:
			return firstInt(wrapper.Total, wrapper.Count, wrapper.JobsCount), nil
		}
	}
}

func firstInt(values ...*int) int {
	for _, value := range values {
		if value != nil {
			return *value
		}
	}
	return 0
}

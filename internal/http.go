package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// HTTPClient 封装 HTTP 请求
type HTTPClient struct {
	BaseURL    string
	APIKey     string
	Doer       func(*http.Request) (*http.Response, error)
	MaxRetries int
}

func NewHTTPClient(baseURL, apiKey string, doer func(*http.Request) (*http.Response, error), maxRetries int) *HTTPClient {
	if doer == nil {
		doer = http.DefaultClient.Do
	}
	return &HTTPClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Doer:       doer,
		MaxRetries: maxRetries,
	}
}

func (c *HTTPClient) POST(ctx context.Context, path string, body any, result any) error {
	req, err := c.buildRequest(ctx, http.MethodPost, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *HTTPClient) POSTStream(ctx context.Context, path string, body any) (io.ReadCloser, error) {
	req, err := c.buildRequest(ctx, http.MethodPost, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.Doer(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: respBody}
	}
	return resp.Body, nil
}

func (c *HTTPClient) buildRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *HTTPClient) do(req *http.Request, result any) error {
	resp, err := c.Doer(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: respBody}
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// HTTPError 内部 HTTP 错误
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPError) Error() string {
	return "http error: " + e.Status
}

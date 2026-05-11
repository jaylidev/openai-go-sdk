package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// HTTPClient 封装 HTTP 请求
type HTTPClient struct {
	BaseURL    string
	APIKey     string
	Doer       func(*http.Request) (*http.Response, error)
	MaxRetries int
	logReq     func(method, fullURL string, body []byte)
	logResp    func(method, fullURL string, statusCode int, body []byte, dur time.Duration, err error)
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

// SetLogHooks 注入日志回调钩子
func (c *HTTPClient) SetLogHooks(
	logReq func(method, fullURL string, body []byte),
	logResp func(method, fullURL string, statusCode int, body []byte, dur time.Duration, err error),
) {
	c.logReq = logReq
	c.logResp = logResp
}

func (c *HTTPClient) POST(ctx context.Context, path string, body any, result any) error {
	req, err := c.buildRequest(ctx, http.MethodPost, c.BaseURL+path, body)
	if err != nil {
		return err
	}

	bodyBytes, _ := json.Marshal(body)
	if c.logReq != nil {
		c.logReq(http.MethodPost, req.URL.String(), bodyBytes)
	}

	start := time.Now()
	err = c.do(req, result)
	dur := time.Since(start)

	if c.logResp != nil {
		statusCode := 200
		if err != nil {
			statusCode = 0
			if httpErr, ok := err.(*HTTPError); ok {
				statusCode = httpErr.StatusCode
			}
		}
		var respBytes []byte
		if err == nil && result != nil {
			respBytes, _ = json.Marshal(result)
		}
		if err != nil {
			if httpErr, ok := err.(*HTTPError); ok {
				respBytes = httpErr.Body
			}
		}
		c.logResp(http.MethodPost, req.URL.String(), statusCode, respBytes, dur, err)
	}

	return err
}

func (c *HTTPClient) POSTStream(ctx context.Context, path string, body any) (io.ReadCloser, error) {
	req, err := c.buildRequest(ctx, http.MethodPost, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	bodyBytes, _ := json.Marshal(body)
	if c.logReq != nil {
		c.logReq(http.MethodPost, req.URL.String(), bodyBytes)
	}

	start := time.Now()
	resp, err := c.Doer(req)
	dur := time.Since(start)

	if err != nil {
		if c.logResp != nil {
			c.logResp(http.MethodPost, req.URL.String(), 0, nil, dur, err)
		}
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if c.logResp != nil {
			c.logResp(http.MethodPost, req.URL.String(), resp.StatusCode, respBody, dur, nil)
		}
		return nil, &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: respBody}
	}

	if c.logResp != nil {
		c.logResp(http.MethodPost, req.URL.String(), 200, []byte("(stream)"), dur, nil)
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

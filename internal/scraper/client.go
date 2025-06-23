// internal/scraper/client.go

package scraper

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

// HTTPClient defines the interface for making HTTP requests.
// This abstraction allows for easy testing and swapping of HTTP client implementations.
type HTTPClient interface {
	// Get performs an HTTP GET request to the specified URL.
	// The context can be used to cancel the request.
	Get(ctx context.Context, url string) (*http.Response, error)
	
	// Post performs an HTTP POST request to the specified URL with the given body.
	Post(ctx context.Context, url string, contentType string, body []byte) (*http.Response, error)
	
	// Do performs an HTTP request with full control over the request.
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient provides a standard implementation of HTTPClient.
type DefaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a new HTTP client with default settings.
func NewDefaultHTTPClient() *DefaultHTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get implements HTTPClient.Get
func (c *DefaultHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

// Post implements HTTPClient.Post
func (c *DefaultHTTPClient) Post(ctx context.Context, url string, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.client.Do(req)
}

// Do implements HTTPClient.Do
func (c *DefaultHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.client.Do(req.WithContext(ctx))
}

// internal/scraper/client_test.go
package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	config := &HTTPClientConfig{
		Timeout:       10 * time.Second,
		RetryAttempts: 3,
		UserAgents:    []string{"TestAgent/1.0"},
	}

	client := NewHTTPClient(config)
	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.config != config {
		t.Error("Client should reference the provided config")
	}

	if client.stats == nil {
		t.Error("Stats should be initialized")
	}
}

func TestNewHTTPClient_DefaultConfig(t *testing.T) {
	client := NewHTTPClient(nil)
	if client == nil {
		t.Fatal("Expected client to be created with default config")
	}

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", client.config.Timeout)
	}

	if client.config.RetryAttempts != 3 {
		t.Errorf("Expected default retry attempts 3, got %d", client.config.RetryAttempts)
	}
}

func TestHTTPClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Test Content</body></html>"))
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:       10 * time.Second,
		RetryAttempts: 3,
		UserAgents:    []string{"TestAgent/1.0"},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Expected successful request, got error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", resp.Attempts)
	}

	if len(resp.BodyBytes) == 0 {
		t.Error("Expected response body to be read")
	}

	bodyStr := string(resp.BodyBytes)
	if !strings.Contains(bodyStr, "Test Content") {
		t.Errorf("Expected body to contain 'Test Content', got: %s", bodyStr)
	}
}

func TestHTTPClient_Get_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success after retries"))
		}
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:          10 * time.Second,
		RetryAttempts:    3,
		RetryBackoffBase: 10 * time.Millisecond,
		UserAgents:       []string{"TestAgent/1.0"},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Expected successful request after retries, got error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", resp.Attempts)
	}

	if attempts != 3 {
		t.Errorf("Expected server to receive 3 requests, got %d", attempts)
	}
}

func TestHTTPClient_Get_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // Always return 500 to trigger retries
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:          10 * time.Second,
		RetryAttempts:    2,
		RetryBackoffBase: 10 * time.Millisecond,
		UserAgents:       []string{"TestAgent/1.0"},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	resp, _ := client.Get(ctx, server.URL)

	// Should get final response even after retries fail
	if resp == nil {
		t.Fatal("Expected response even after retries failed")
	}

	if resp.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	if resp.Attempts != 3 { // 1 initial + 2 retries
		t.Errorf("Expected 3 attempts, got %d", resp.Attempts)
	}

	// Since 500 errors return response but no Go error, check status
	if resp.StatusCode < 400 {
		t.Error("Expected error status code after max retries exceeded")
	}
}

func TestHTTPClient_UserAgentRotation(t *testing.T) {
	var receivedUserAgents []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgents = append(receivedUserAgents, r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:       10 * time.Second,
		RetryAttempts: 0,
		UserAgents:    []string{"Agent1/1.0", "Agent2/1.0", "Agent3/1.0"},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	// Make multiple requests
	for i := 0; i < 4; i++ {
		_, err := client.Get(ctx, server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	if len(receivedUserAgents) != 4 {
		t.Fatalf("Expected 4 user agents, got %d", len(receivedUserAgents))
	}

	// Check rotation pattern
	expected := []string{"Agent1/1.0", "Agent2/1.0", "Agent3/1.0", "Agent1/1.0"}
	for i, expected := range expected {
		if receivedUserAgents[i] != expected {
			t.Errorf("Request %d: expected user agent %s, got %s", i, expected, receivedUserAgents[i])
		}
	}
}

func TestHTTPClient_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:       10 * time.Second,
		RetryAttempts: 0,
		UserAgents:    []string{"TestAgent/1.0"},
		Headers: map[string]string{
			"X-Custom-Header": "CustomValue",
			"Authorization":   "Bearer token123",
		},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	_, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if receivedHeaders.Get("X-Custom-Header") != "CustomValue" {
		t.Errorf("Expected custom header 'CustomValue', got '%s'", receivedHeaders.Get("X-Custom-Header"))
	}

	if receivedHeaders.Get("Authorization") != "Bearer token123" {
		t.Errorf("Expected authorization header 'Bearer token123', got '%s'", receivedHeaders.Get("Authorization"))
	}

	if receivedHeaders.Get("User-Agent") != "TestAgent/1.0" {
		t.Errorf("Expected user agent 'TestAgent/1.0', got '%s'", receivedHeaders.Get("User-Agent"))
	}
}

func TestHTTPClient_Cookies(t *testing.T) {
	var receivedCookies []*http.Cookie
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCookies = r.Cookies()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:       10 * time.Second,
		RetryAttempts: 0,
		UserAgents:    []string{"TestAgent/1.0"},
		Cookies: map[string]string{
			"session_id": "abc123",
			"user_pref":  "dark_mode",
		},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	_, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if len(receivedCookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(receivedCookies))
	}

	cookieMap := make(map[string]string)
	for _, cookie := range receivedCookies {
		cookieMap[cookie.Name] = cookie.Value
	}

	if cookieMap["session_id"] != "abc123" {
		t.Errorf("Expected session_id 'abc123', got '%s'", cookieMap["session_id"])
	}

	if cookieMap["user_pref"] != "dark_mode" {
		t.Errorf("Expected user_pref 'dark_mode', got '%s'", cookieMap["user_pref"])
	}
}

func TestHTTPClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:       500 * time.Millisecond,
		RetryAttempts: 0,
		UserAgents:    []string{"TestAgent/1.0"},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	start := time.Now()
	_, err := client.Get(ctx, server.URL)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if duration > 1*time.Second {
		t.Errorf("Request took too long: %v", duration)
	}
}

func TestHTTPClient_RateLimit(t *testing.T) {
	requestTimes := make([]time.Time, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:       10 * time.Second,
		RetryAttempts: 0,
		UserAgents:    []string{"TestAgent/1.0"},
		RateLimit:     500 * time.Millisecond,
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	// Make 3 requests
	for i := 0; i < 3; i++ {
		_, err := client.Get(ctx, server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	if len(requestTimes) != 3 {
		t.Fatalf("Expected 3 requests, got %d", len(requestTimes))
	}

	// Check that requests are spaced by at least the rate limit
	for i := 1; i < len(requestTimes); i++ {
		gap := requestTimes[i].Sub(requestTimes[i-1])
		if gap < 400*time.Millisecond { // Allow some tolerance
			t.Errorf("Request %d came too soon after previous: %v", i, gap)
		}
	}
}

func TestHTTPClient_Stats(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError) // 500 - triggers retry
		} else if attempts == 2 {
			w.WriteHeader(http.StatusBadGateway) // 502 - triggers retry
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:          10 * time.Second,
		RetryAttempts:    3,
		RetryBackoffBase: 10 * time.Millisecond,
		UserAgents:       []string{"TestAgent/1.0"},
	}

	client := NewHTTPClient(config)
	ctx := context.Background()

	_, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	stats := client.GetStats()
	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats.TotalRequests)
	}

	if stats.SuccessfulReqs != 1 {
		t.Errorf("Expected 1 successful request, got %d", stats.SuccessfulReqs)
	}

	if stats.FailedRequests != 2 {
		t.Errorf("Expected 2 failed requests, got %d", stats.FailedRequests)
	}

	if stats.RetryCount != 2 {
		t.Errorf("Expected 2 retries, got %d", stats.RetryCount)
	}

	if stats.ErrorsByCode[500] != 1 {
		t.Errorf("Expected 1 error with code 500, got %d", stats.ErrorsByCode[500])
	}

	if stats.ErrorsByCode[502] != 1 {
		t.Errorf("Expected 1 error with code 502, got %d", stats.ErrorsByCode[502])
	}
}

func TestHTTPClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPClientConfig{
		Timeout:       10 * time.Second,
		RetryAttempts: 0,
		UserAgents:    []string{"TestAgent/1.0"},
	}

	client := NewHTTPClient(config)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.Get(ctx, server.URL)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if duration > 1*time.Second {
		t.Errorf("Request took too long after context cancellation: %v", duration)
	}
}

func TestHTTPClient_SetUserAgent(t *testing.T) {
	client := NewHTTPClient(&HTTPClientConfig{
		UserAgents: []string{"Original/1.0"},
	})

	client.SetUserAgent("NewAgent/2.0")
	userAgent := client.GetCurrentUserAgent()

	if userAgent != "NewAgent/2.0" {
		t.Errorf("Expected 'NewAgent/2.0', got '%s'", userAgent)
	}
}

func TestHTTPClient_AddUserAgent(t *testing.T) {
	client := NewHTTPClient(&HTTPClientConfig{
		UserAgents: []string{"Agent1/1.0"},
	})

	client.AddUserAgent("Agent2/1.0")

	if len(client.config.UserAgents) != 2 {
		t.Errorf("Expected 2 user agents, got %d", len(client.config.UserAgents))
	}
}

func TestHTTPClient_SetHeaderAndCookie(t *testing.T) {
	client := NewHTTPClient(&HTTPClientConfig{})

	client.SetHeader("X-Test", "TestValue")
	client.SetCookie("test_cookie", "cookie_value")

	if client.config.Headers["X-Test"] != "TestValue" {
		t.Errorf("Expected header not set correctly")
	}

	if client.config.Cookies["test_cookie"] != "cookie_value" {
		t.Errorf("Expected cookie not set correctly")
	}
}

func TestHTTPClient_Close(t *testing.T) {
	client := NewHTTPClient(&HTTPClientConfig{})

	err := client.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}

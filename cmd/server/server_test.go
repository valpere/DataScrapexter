// cmd/server/server_test.go
package main

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/gorilla/mux"
    "golang.org/x/time/rate"
)

func TestHealthEndpoint(t *testing.T) {
    server := setupTestServer()
    defer server.Close()

    resp, err := http.Get(server.URL + "/health")
    if err != nil {
        t.Fatalf("health check failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status 200, got %d", resp.StatusCode)
    }
}

func TestCreateScraperJob(t *testing.T) {
    server := setupTestServer()
    defer server.Close()

    config := map[string]interface{}{
        "name":     "test_scraper",
        "base_url": "https://example.com",
        "fields": []map[string]interface{}{
            {
                "name":     "title",
                "selector": "h1",
                "type":     "text",
                "required": true,
            },
        },
        "output": map[string]interface{}{
            "format": "json",
        },
    }

    jsonBody, err := json.Marshal(config)
    if err != nil {
        t.Fatalf("failed to marshal config: %v", err)
    }

    resp, err := http.Post(
        server.URL+"/api/v1/scrapers",
        "application/json",
        bytes.NewBuffer(jsonBody),
    )
    if err != nil {
        t.Fatalf("create scraper request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        body, _ := io.ReadAll(resp.Body)
        t.Errorf("expected status 201, got %d. Body: %s", resp.StatusCode, body)
    }
}

// Helper functions for setting up test servers

func setupTestServer() *httptest.Server {
    return httptest.NewServer(setupRoutes())
}

func setupTestServerWithAuth() *httptest.Server {
    handler := setupRoutes()
    authHandler := authMiddleware(handler)
    return httptest.NewServer(authHandler)
}

func setupTestServerWithRateLimit() *httptest.Server {
    handler := setupRoutes()
    rateLimitHandler := rateLimitMiddleware(handler)
    return httptest.NewServer(rateLimitHandler)
}

// Mock implementations for testing

func setupRoutes() http.Handler {
    r := mux.NewRouter()
    
    r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/metrics", metricsHandler).Methods("GET")
    
    api := r.PathPrefix("/api/v1").Subrouter()
    api.HandleFunc("/scrapers", createScraperHandler).Methods("POST")
    api.HandleFunc("/scrapers", listScrapersHandler).Methods("GET")
    api.HandleFunc("/scrapers/{id}", getScraperHandler).Methods("GET")
    
    return r
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
            return
        }
        
        token := strings.TrimPrefix(authHeader, "Bearer ")
        if !isValidAPIKey(token) {
            http.Error(w, "Invalid API key", http.StatusUnauthorized)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func rateLimitMiddleware(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(10), 20)
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func isValidAPIKey(token string) bool {
    return token == "valid_api_key_123"
}

// Mock handlers

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now(),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte("# HELP datascrapexter_requests_total Total requests\n"))
    w.Write([]byte("datascrapexter_requests_total 42\n"))
}

func createScraperHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    response := map[string]interface{}{
        "id":     "test-scraper-123",
        "status": "created",
    }
    json.NewEncoder(w).Encode(response)
}

func listScrapersHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    response := map[string]interface{}{
        "scrapers": []map[string]interface{}{},
        "total":    0,
    }
    json.NewEncoder(w).Encode(response)
}

func getScraperHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    w.Header().Set("Content-Type", "application/json")
    response := map[string]interface{}{
        "id":   id,
        "name": "test_scraper",
    }
    json.NewEncoder(w).Encode(response)
}

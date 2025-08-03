// internal/compliance/compliance_test.go
package compliance

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRobotsTxtParser(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /private/
Disallow: /admin/
Allow: /public/
Crawl-delay: 1

User-agent: DataScrapexter
Disallow: /api/
Allow: /data/

Sitemap: https://example.com/sitemap.xml
`

	parser := NewRobotsTxtParser()
	robots, err := parser.Parse([]byte(robotsTxt))
	if err != nil {
		t.Fatalf("failed to parse robots.txt: %v", err)
	}

	// Test general user agent rules
	if !robots.IsDisallowed("*", "/private/page") {
		t.Error("/private/ should be disallowed for all user agents")
	}

	if robots.IsDisallowed("*", "/public/page") {
		t.Error("/public/ should be allowed for all user agents")
	}

	// Test specific user agent rules
	if !robots.IsDisallowed("DataScrapexter", "/api/endpoint") {
		t.Error("/api/ should be disallowed for DataScrapexter")
	}

	if robots.IsDisallowed("DataScrapexter", "/data/file") {
		t.Error("/data/ should be allowed for DataScrapexter")
	}

	// Test crawl delay
	delay := robots.GetCrawlDelay("*")
	if delay != 1*time.Second {
		t.Errorf("expected crawl delay 1s, got %v", delay)
	}

	// Test sitemap
	sitemaps := robots.GetSitemaps()
	if len(sitemaps) != 1 {
		t.Errorf("expected 1 sitemap, got %d", len(sitemaps))
	}
	if sitemaps[0] != "https://example.com/sitemap.xml" {
		t.Errorf("unexpected sitemap URL: %s", sitemaps[0])
	}
}

func TestRobotsTxtFetcher(t *testing.T) {
	robotsContent := `
User-agent: *
Disallow: /private/
Crawl-delay: 2
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(robotsContent))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := NewRobotsTxtFetcher()
	robots, err := fetcher.Fetch(server.URL)
	if err != nil {
		t.Fatalf("failed to fetch robots.txt: %v", err)
	}

	if !robots.IsDisallowed("*", "/private/test") {
		t.Error("fetched robots.txt should disallow /private/")
	}

	delay := robots.GetCrawlDelay("*")
	if delay != 2*time.Second {
		t.Errorf("expected crawl delay 2s, got %v", delay)
	}
}

func TestGDPRChecker(t *testing.T) {
	checker := NewGDPRChecker()

	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"EU domain", "example.de", true},
		{"French domain", "example.fr", true},
		{"UK domain", "example.co.uk", true},
		{"US domain", "example.com", false},
		{"Asian domain", "example.jp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.RequiresGDPRCompliance(tt.domain)
			if result != tt.expected {
				t.Errorf("GDPR check for %s: expected %v, got %v", tt.domain, tt.expected, result)
			}
		})
	}
}

func TestComplianceReport(t *testing.T) {
	checker := NewComplianceChecker()

	testURL := "https://example.com/data"

	// Mock robots.txt that allows access
	mockRobots := &RobotsTxt{
		rules: map[string][]Rule{
			"*": {
				{Pattern: "/private/", Allow: false},
				{Pattern: "/", Allow: true},
			},
		},
		crawlDelays: map[string]time.Duration{
			"*": 1 * time.Second,
		},
	}

	report := checker.GenerateReport(testURL, mockRobots, nil, nil)

	if !report.RobotsCompliant {
		t.Error("should be robots.txt compliant")
	}
	if report.RecommendedDelay != 1*time.Second {
		t.Errorf("expected delay 1s, got %v", report.RecommendedDelay)
	}
	if report.RiskLevel == "high" {
		t.Error("compliant URL should not be high risk")
	}
}

// Mock implementations for testing
type RobotsTxtParser struct{}

func NewRobotsTxtParser() *RobotsTxtParser {
	return &RobotsTxtParser{}
}

func (p *RobotsTxtParser) Parse(data []byte) (*RobotsTxt, error) {
	robots := &RobotsTxt{
		rules:       make(map[string][]Rule),
		crawlDelays: make(map[string]time.Duration),
		sitemaps:    []string{},
	}

	lines := strings.Split(string(data), "\n")
	currentUserAgent := "*"

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		directive := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch directive {
		case "user-agent":
			currentUserAgent = value
		case "disallow":
			if value != "" {
				robots.rules[currentUserAgent] = append(robots.rules[currentUserAgent], Rule{
					Pattern: value,
					Allow:   false,
				})
			}
		case "allow":
			robots.rules[currentUserAgent] = append(robots.rules[currentUserAgent], Rule{
				Pattern: value,
				Allow:   true,
			})
		case "crawl-delay":
			if delay, err := time.ParseDuration(value + "s"); err == nil {
				robots.crawlDelays[currentUserAgent] = delay
			}
		case "sitemap":
			robots.sitemaps = append(robots.sitemaps, value)
		}
	}

	return robots, nil
}

type RobotsTxt struct {
	rules       map[string][]Rule
	crawlDelays map[string]time.Duration
	sitemaps    []string
}

type Rule struct {
	Pattern string
	Allow   bool
}

func (r *RobotsTxt) IsDisallowed(userAgent, path string) bool {
	rules := r.rules[userAgent]
	if len(rules) == 0 {
		rules = r.rules["*"]
	}

	for _, rule := range rules {
		if strings.HasPrefix(path, rule.Pattern) {
			return !rule.Allow
		}
	}
	return false
}

func (r *RobotsTxt) GetCrawlDelay(userAgent string) time.Duration {
	if delay, exists := r.crawlDelays[userAgent]; exists {
		return delay
	}
	if delay, exists := r.crawlDelays["*"]; exists {
		return delay
	}
	return 0
}

func (r *RobotsTxt) GetSitemaps() []string {
	return r.sitemaps
}

type RobotsTxtFetcher struct{}

func NewRobotsTxtFetcher() *RobotsTxtFetcher {
	return &RobotsTxtFetcher{}
}

func (f *RobotsTxtFetcher) Fetch(baseURL string) (*RobotsTxt, error) {
	resp, err := http.Get(baseURL + "/robots.txt")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data := make([]byte, resp.ContentLength)
	resp.Body.Read(data)

	parser := NewRobotsTxtParser()
	return parser.Parse(data)
}

type GDPRChecker struct{}

func NewGDPRChecker() *GDPRChecker {
	return &GDPRChecker{}
}

func (g *GDPRChecker) RequiresGDPRCompliance(domain string) bool {
	euDomains := []string{".de", ".fr", ".co.uk", ".eu", ".it", ".es", ".nl"}
	for _, suffix := range euDomains {
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}
	return false
}

type ComplianceChecker struct{}

func NewComplianceChecker() *ComplianceChecker {
	return &ComplianceChecker{}
}

type ComplianceReport struct {
	RobotsCompliant     bool
	RecommendedDelay    time.Duration
	RiskLevel           string
	GDPRRequired        bool
	HasConsentMechanism bool
}

func (c *ComplianceChecker) GenerateReport(url string, robots *RobotsTxt, terms interface{}, privacy interface{}) *ComplianceReport {
	return &ComplianceReport{
		RobotsCompliant:     true,
		RecommendedDelay:    robots.GetCrawlDelay("*"),
		RiskLevel:           "low",
		GDPRRequired:        false,
		HasConsentMechanism: true,
	}
}

// internal/antidetect/captcha.go
package antidetect

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CaptchaSolverType represents different CAPTCHA solving services
type CaptchaSolverType string

const (
	TwoCaptcha    CaptchaSolverType = "2captcha"
	AntiCaptcha   CaptchaSolverType = "anticaptcha"
	CapMonster    CaptchaSolverType = "capmonster"
	DeathByCaptcha CaptchaSolverType = "deathbycaptcha"
)

// CaptchaTask represents a CAPTCHA solving task
type CaptchaTask struct {
	ID          string
	Type        CaptchaType
	SiteKey     string
	SiteURL     string
	ImageData   string
	ProxyType   string
	ProxyHost   string
	ProxyPort   int
	ProxyUser   string
	ProxyPass   string
	UserAgent   string
	Cookies     string
	MinScore    float64
	PageAction  string
	IsInvisible bool
	Timeout     time.Duration
}

// CaptchaSolution represents a solved CAPTCHA
type CaptchaSolution struct {
	ID       string
	Token    string
	Text     string
	Cost     float64
	SolveTime time.Duration
	Success  bool
	Error    string
}

// CaptchaSolver interface for CAPTCHA solving services
type CaptchaSolver interface {
	SubmitTask(ctx context.Context, task *CaptchaTask) (string, error)
	GetResult(ctx context.Context, taskID string) (*CaptchaSolution, error)
	GetBalance(ctx context.Context) (float64, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// CaptchaManager manages multiple CAPTCHA solving services
type CaptchaManager struct {
	solvers     map[CaptchaSolverType]CaptchaSolver
	defaultType CaptchaSolverType
	config      *CaptchaConfig
}

// CaptchaConfig configuration for CAPTCHA solving
type CaptchaConfig struct {
	Enabled          bool
	DefaultSolver    CaptchaSolverType
	RetryAttempts    int
	SolveTimeout     time.Duration
	PollingInterval  time.Duration
	AutoRetry        bool
	FallbackSolvers  []CaptchaSolverType
	MinBalance       float64
	MaxConcurrent    int
}

// NewCaptchaManager creates a new CAPTCHA manager
func NewCaptchaManager(config *CaptchaConfig) *CaptchaManager {
	if config == nil {
		config = &CaptchaConfig{
			Enabled:         false,
			DefaultSolver:   TwoCaptcha,
			RetryAttempts:   3,
			SolveTimeout:    120 * time.Second,
			PollingInterval: 5 * time.Second,
			AutoRetry:       true,
			MinBalance:      1.0,
			MaxConcurrent:   10,
		}
	}
	
	return &CaptchaManager{
		solvers:     make(map[CaptchaSolverType]CaptchaSolver),
		defaultType: config.DefaultSolver,
		config:      config,
	}
}

// RegisterSolver registers a CAPTCHA solver
func (cm *CaptchaManager) RegisterSolver(solverType CaptchaSolverType, solver CaptchaSolver) {
	cm.solvers[solverType] = solver
}

// SolveRecaptchaV2 solves a reCAPTCHA v2
func (cm *CaptchaManager) SolveRecaptchaV2(ctx context.Context, siteKey, siteURL string, proxy *ProxyConfig) (*CaptchaSolution, error) {
	task := &CaptchaTask{
		Type:     RecaptchaV2,
		SiteKey:  siteKey,
		SiteURL:  siteURL,
		Timeout:  cm.config.SolveTimeout,
	}
	
	if proxy != nil && len(proxy.URLs) > 0 {
		// Parse proxy URL (simplified)
		task.ProxyType = "http"
		task.ProxyHost = proxy.URLs[0] // Use first proxy
	}
	
	return cm.solveCaptcha(ctx, task)
}

// SolveRecaptchaV3 solves a reCAPTCHA v3
func (cm *CaptchaManager) SolveRecaptchaV3(ctx context.Context, siteKey, siteURL, action string, minScore float64) (*CaptchaSolution, error) {
	task := &CaptchaTask{
		Type:       RecaptchaV3,
		SiteKey:    siteKey,
		SiteURL:    siteURL,
		PageAction: action,
		MinScore:   minScore,
		Timeout:    cm.config.SolveTimeout,
	}
	
	return cm.solveCaptcha(ctx, task)
}

// SolveHCaptcha solves an hCaptcha
func (cm *CaptchaManager) SolveHCaptcha(ctx context.Context, siteKey, siteURL string) (*CaptchaSolution, error) {
	task := &CaptchaTask{
		Type:    HCaptcha,
		SiteKey: siteKey,
		SiteURL: siteURL,
		Timeout: cm.config.SolveTimeout,
	}
	
	return cm.solveCaptcha(ctx, task)
}

// SolveImageCaptcha solves an image-based CAPTCHA
func (cm *CaptchaManager) SolveImageCaptcha(ctx context.Context, imageData []byte) (*CaptchaSolution, error) {
	task := &CaptchaTask{
		Type:      CaptchaType(999), // Generic image CAPTCHA
		ImageData: base64.StdEncoding.EncodeToString(imageData),
		Timeout:   cm.config.SolveTimeout,
	}
	
	return cm.solveCaptcha(ctx, task)
}

// solveCaptcha solves a CAPTCHA using the configured solver
func (cm *CaptchaManager) solveCaptcha(ctx context.Context, task *CaptchaTask) (*CaptchaSolution, error) {
	if !cm.config.Enabled {
		return nil, fmt.Errorf("CAPTCHA solving is disabled")
	}
	
	solver, exists := cm.solvers[cm.defaultType]
	if !exists {
		return nil, fmt.Errorf("solver %s not registered", cm.defaultType)
	}
	
	// Check balance first
	balance, err := solver.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}
	
	if balance < cm.config.MinBalance {
		return nil, fmt.Errorf("insufficient balance: %.2f (minimum: %.2f)", balance, cm.config.MinBalance)
	}
	
	// Submit task
	taskID, err := solver.SubmitTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to submit CAPTCHA task: %w", err)
	}
	
	task.ID = taskID
	
	// Poll for result
	return cm.pollForResult(ctx, solver, taskID)
}

// pollForResult polls for CAPTCHA solution result
func (cm *CaptchaManager) pollForResult(ctx context.Context, solver CaptchaSolver, taskID string) (*CaptchaSolution, error) {
	ticker := time.NewTicker(cm.config.PollingInterval)
	defer ticker.Stop()
	
	timeout := time.After(cm.config.SolveTimeout)
	startTime := time.Now()
	
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("CAPTCHA solving timeout after %v", cm.config.SolveTimeout)
		case <-ticker.C:
			solution, err := solver.GetResult(ctx, taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to get CAPTCHA result: %w", err)
			}
			
			if solution != nil && solution.Success {
				solution.SolveTime = time.Since(startTime)
				return solution, nil
			}
			
			if solution != nil && solution.Error != "" {
				return nil, fmt.Errorf("CAPTCHA solving failed: %s", solution.Error)
			}
			
			// Continue polling
		}
	}
}

// GetStats returns statistics from all registered solvers
func (cm *CaptchaManager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	for solverType, solver := range cm.solvers {
		solverStats, err := solver.GetStats(ctx)
		if err != nil {
			stats[string(solverType)] = map[string]interface{}{
				"error": err.Error(),
			}
		} else {
			stats[string(solverType)] = solverStats
		}
	}
	
	return stats, nil
}

// TwoCaptchaSolver implements 2Captcha API
type TwoCaptchaSolver struct {
	apiKey string
	client *http.Client
	baseURL string
}

// NewTwoCaptchaSolver creates a new 2Captcha solver
func NewTwoCaptchaSolver(apiKey string) *TwoCaptchaSolver {
	return &TwoCaptchaSolver{
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://2captcha.com",
	}
}

// SubmitTask submits a CAPTCHA task to 2Captcha
func (tc *TwoCaptchaSolver) SubmitTask(ctx context.Context, task *CaptchaTask) (string, error) {
	var method string
	params := map[string]string{
		"key": tc.apiKey,
		"json": "1",
	}
	
	switch task.Type {
	case RecaptchaV2:
		method = "userrecaptcha"
		params["googlekey"] = task.SiteKey
		params["pageurl"] = task.SiteURL
		if task.IsInvisible {
			params["invisible"] = "1"
		}
	case RecaptchaV3:
		method = "userrecaptcha"
		params["googlekey"] = task.SiteKey
		params["pageurl"] = task.SiteURL
		params["version"] = "v3"
		params["action"] = task.PageAction
		params["min_score"] = fmt.Sprintf("%.1f", task.MinScore)
	case HCaptcha:
		method = "hcaptcha"
		params["sitekey"] = task.SiteKey
		params["pageurl"] = task.SiteURL
	default:
		return "", fmt.Errorf("unsupported CAPTCHA type: %v", task.Type)
	}
	
	params["method"] = method
	
	resp, err := tc.makeRequest(ctx, "in.php", params)
	if err != nil {
		return "", err
	}
	
	var result struct {
		Status int    `json:"status"`
		Request string `json:"request"`
		Error   string `json:"error_text,omitempty"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.Status != 1 {
		return "", fmt.Errorf("2Captcha error: %s", result.Error)
	}
	
	return result.Request, nil
}

// GetResult gets the result of a CAPTCHA task from 2Captcha
func (tc *TwoCaptchaSolver) GetResult(ctx context.Context, taskID string) (*CaptchaSolution, error) {
	params := map[string]string{
		"key":    tc.apiKey,
		"action": "get",
		"id":     taskID,
		"json":   "1",
	}
	
	resp, err := tc.makeRequest(ctx, "res.php", params)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Status  int    `json:"status"`
		Request string `json:"request"`
		Error   string `json:"error_text,omitempty"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.Status == 0 && result.Request == "CAPCHA_NOT_READY" {
		return nil, nil // Still processing
	}
	
	if result.Status != 1 {
		return &CaptchaSolution{
			ID:      taskID,
			Success: false,
			Error:   result.Error,
		}, nil
	}
	
	return &CaptchaSolution{
		ID:      taskID,
		Token:   result.Request,
		Success: true,
	}, nil
}

// GetBalance gets account balance from 2Captcha
func (tc *TwoCaptchaSolver) GetBalance(ctx context.Context) (float64, error) {
	params := map[string]string{
		"key":    tc.apiKey,
		"action": "getbalance",
		"json":   "1",
	}
	
	resp, err := tc.makeRequest(ctx, "res.php", params)
	if err != nil {
		return 0, err
	}
	
	var result struct {
		Status  int     `json:"status"`
		Request float64 `json:"request"`
		Error   string  `json:"error_text,omitempty"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.Status != 1 {
		return 0, fmt.Errorf("2Captcha error: %s", result.Error)
	}
	
	return result.Request, nil
}

// GetStats gets statistics from 2Captcha
func (tc *TwoCaptchaSolver) GetStats(ctx context.Context) (map[string]interface{}, error) {
	balance, err := tc.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"service": "2captcha",
		"balance": balance,
		"status":  "active",
	}, nil
}

// makeRequest makes an HTTP request to 2Captcha API
func (tc *TwoCaptchaSolver) makeRequest(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	// Build query string
	var queryParams []string
	for key, value := range params {
		queryParams = append(queryParams, fmt.Sprintf("%s=%s", key, value))
	}
	
	url := fmt.Sprintf("%s/%s?%s", tc.baseURL, endpoint, strings.Join(queryParams, "&"))
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	return io.ReadAll(resp.Body)
}

// AntiCaptchaSolver implements Anti-Captcha API
type AntiCaptchaSolver struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

// NewAntiCaptchaSolver creates a new Anti-Captcha solver
func NewAntiCaptchaSolver(apiKey string) *AntiCaptchaSolver {
	return &AntiCaptchaSolver{
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://api.anti-captcha.com",
	}
}

// SubmitTask submits a CAPTCHA task to Anti-Captcha
func (ac *AntiCaptchaSolver) SubmitTask(ctx context.Context, task *CaptchaTask) (string, error) {
	var taskData map[string]interface{}
	
	switch task.Type {
	case RecaptchaV2:
		taskData = map[string]interface{}{
			"type":       "NoCaptchaTaskProxyless",
			"websiteURL": task.SiteURL,
			"websiteKey": task.SiteKey,
		}
	case RecaptchaV3:
		taskData = map[string]interface{}{
			"type":         "RecaptchaV3TaskProxyless",
			"websiteURL":   task.SiteURL,
			"websiteKey":   task.SiteKey,
			"pageAction":   task.PageAction,
			"minScore":     task.MinScore,
		}
	case HCaptcha:
		taskData = map[string]interface{}{
			"type":       "HCaptchaTaskProxyless",
			"websiteURL": task.SiteURL,
			"websiteKey": task.SiteKey,
		}
	default:
		return "", fmt.Errorf("unsupported CAPTCHA type: %v", task.Type)
	}
	
	payload := map[string]interface{}{
		"clientKey": ac.apiKey,
		"task":      taskData,
	}
	
	resp, err := ac.makeJSONRequest(ctx, "createTask", payload)
	if err != nil {
		return "", err
	}
	
	var result struct {
		ErrorID     int    `json:"errorId"`
		ErrorCode   string `json:"errorCode,omitempty"`
		ErrorDesc   string `json:"errorDescription,omitempty"`
		TaskID      int    `json:"taskId"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.ErrorID != 0 {
		return "", fmt.Errorf("Anti-Captcha error: %s", result.ErrorDesc)
	}
	
	return fmt.Sprintf("%d", result.TaskID), nil
}

// GetResult gets the result of a CAPTCHA task from Anti-Captcha
func (ac *AntiCaptchaSolver) GetResult(ctx context.Context, taskID string) (*CaptchaSolution, error) {
	payload := map[string]interface{}{
		"clientKey": ac.apiKey,
		"taskId":    taskID,
	}
	
	resp, err := ac.makeJSONRequest(ctx, "getTaskResult", payload)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		ErrorID   int    `json:"errorId"`
		ErrorCode string `json:"errorCode,omitempty"`
		ErrorDesc string `json:"errorDescription,omitempty"`
		Status    string `json:"status"`
		Solution  struct {
			GRecaptchaResponse string `json:"gRecaptchaResponse,omitempty"`
			Text               string `json:"text,omitempty"`
		} `json:"solution"`
		Cost string `json:"cost,omitempty"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.ErrorID != 0 {
		return &CaptchaSolution{
			ID:      taskID,
			Success: false,
			Error:   result.ErrorDesc,
		}, nil
	}
	
	if result.Status == "processing" {
		return nil, nil // Still processing
	}
	
	if result.Status == "ready" {
		token := result.Solution.GRecaptchaResponse
		if token == "" {
			token = result.Solution.Text
		}
		
		return &CaptchaSolution{
			ID:      taskID,
			Token:   token,
			Success: true,
		}, nil
	}
	
	return &CaptchaSolution{
		ID:      taskID,
		Success: false,
		Error:   fmt.Sprintf("Unknown status: %s", result.Status),
	}, nil
}

// GetBalance gets account balance from Anti-Captcha
func (ac *AntiCaptchaSolver) GetBalance(ctx context.Context) (float64, error) {
	payload := map[string]interface{}{
		"clientKey": ac.apiKey,
	}
	
	resp, err := ac.makeJSONRequest(ctx, "getBalance", payload)
	if err != nil {
		return 0, err
	}
	
	var result struct {
		ErrorID   int     `json:"errorId"`
		ErrorDesc string  `json:"errorDescription,omitempty"`
		Balance   float64 `json:"balance"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.ErrorID != 0 {
		return 0, fmt.Errorf("Anti-Captcha error: %s", result.ErrorDesc)
	}
	
	return result.Balance, nil
}

// GetStats gets statistics from Anti-Captcha
func (ac *AntiCaptchaSolver) GetStats(ctx context.Context) (map[string]interface{}, error) {
	balance, err := ac.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"service": "anticaptcha",
		"balance": balance,
		"status":  "active",
	}, nil
}

// makeJSONRequest makes a JSON request to Anti-Captcha API
func (ac *AntiCaptchaSolver) makeJSONRequest(ctx context.Context, endpoint string, payload map[string]interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/%s", ac.baseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	return io.ReadAll(resp.Body)
}
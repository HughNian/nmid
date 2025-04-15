package dashboard

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type HealthCheckResult struct {
	URL             string        `json:"url"`
	StatusCode      int           `json:"status_code"`
	ResponseTime    time.Duration `json:"response_time"`
	IsUp            bool          `json:"is_up"`
	LastCheckedAt   time.Time     `json:"last_checked_at"`
	ErrorMessage    string        `json:"error_message,omitempty"`
	CheckIntervalMs int           `json:"check_interval_ms"`
}

type HealthChecker struct {
	targets map[string]*HealthCheckResult
	mu      sync.RWMutex
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		targets: make(map[string]*HealthCheckResult),
	}
}

// 添加检测目标
func (hc *HealthChecker) AddTarget(url string, intervalMs int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// 如果目标已存在，更新时间间隔
	if result, exists := hc.targets[url]; exists {
		result.CheckIntervalMs = intervalMs
		return
	}

	// 创建新的检测结果
	hc.targets[url] = &HealthCheckResult{
		URL:             url,
		LastCheckedAt:   time.Time{},
		CheckIntervalMs: intervalMs,
	}

	// 启动检测协程
	go hc.startChecking(url)
}

// 移除检测目标
func (hc *HealthChecker) RemoveTarget(url string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.targets, url)
}

// 获取单个目标的检测结果
func (hc *HealthChecker) GetResult(url string) *HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if result, exists := hc.targets[url]; exists {
		return result
	}
	return nil
}

// 获取所有检测结果
func (hc *HealthChecker) GetAllResults() []*HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make([]*HealthCheckResult, 0, len(hc.targets))
	for _, result := range hc.targets {
		results = append(results, result)
	}
	return results
}

// 启动检测协程
func (hc *HealthChecker) startChecking(url string) {
	for {
		// 确认目标仍然存在
		hc.mu.RLock()
		result, exists := hc.targets[url]
		interval := 0
		if exists {
			interval = result.CheckIntervalMs
		}
		hc.mu.RUnlock()

		if !exists {
			return
		}

		// 执行检测
		hc.checkHealth(url)

		// 等待指定时间间隔
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

// 执行健康检测
func (hc *HealthChecker) checkHealth(url string) {
	hc.mu.RLock()
	result, exists := hc.targets[url]
	hc.mu.RUnlock()

	if !exists {
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Get(url)
	responseTime := time.Since(startTime)

	// 更新检测结果
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// 再次检查目标是否存在（可能在HTTP请求期间被移除）
	if _, exists := hc.targets[url]; !exists {
		return
	}

	result.ResponseTime = responseTime
	result.LastCheckedAt = time.Now()

	if err != nil {
		result.IsUp = false
		result.ErrorMessage = err.Error()
		result.StatusCode = 0
	} else {
		defer resp.Body.Close()
		result.StatusCode = resp.StatusCode
		result.IsUp = resp.StatusCode >= 200 && resp.StatusCode < 400
		result.ErrorMessage = ""
	}
}

func (hc *HealthChecker) requestTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// 获取所有目标
		results := hc.GetAllResults()
		json.NewEncoder(w).Encode(results)

	case "POST":
		// 添加新目标
		var req struct {
			URL        string `json:"url"`
			IntervalMs int    `json:"interval_ms"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.IntervalMs < 100 {
			req.IntervalMs = 5000 // 默认5秒
		}

		hc.AddTarget(req.URL, req.IntervalMs)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})

	case "DELETE":
		// 删除目标
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "Missing URL parameter", http.StatusBadRequest)
			return
		}

		hc.RemoveTarget(url)
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (hc *HealthChecker) getTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	url := r.URL.Path[len("/api/targets/"):]

	result := hc.GetResult(url)
	if result == nil {
		http.Error(w, "Target not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(result)
}

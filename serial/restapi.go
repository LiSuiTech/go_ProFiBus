package serial

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// RestAPI RESTful API数据源实现
type RestAPI struct {
	config     *RestAPIConfig
	client     *http.Client
	monitoring bool
	dataChan   chan *interfaces.DataUpdate
	stopChan   chan struct{}
	mu         sync.RWMutex
	stats      *interfaces.ProtocolStatus
}

// RestAPIConfig REST API配置
type RestAPIConfig struct {
	// 基础配置
	BaseURL string // 基础URL (e.g., "https://api.example.com")
	APIKey  string // API密钥
	APISecret string // API密钥

	// 认证方式
	AuthType string // none/basic/bearer/apikey/oauth2
	Username string // Basic认证用户名
	Password string // Basic认证密码
	Token    string // Bearer Token

	// 请求配置
	DefaultEndpoint  string            // 默认端点
	DefaultMethod    string            // 默认HTTP方法
	DefaultHeaders   map[string]string // 默认请求头
	DefaultParams    map[string]string // 默认查询参数

	// 轮询配置
	PollingEndpoint string        // 轮询端点
	PollingInterval time.Duration // 轮询间隔
	PollingMethod   string        // 轮询方法

	// 超时配置
	Timeout         time.Duration // 请求超时
	DialTimeout     time.Duration // 连接超时
	KeepAlive       time.Duration // 保持连接时间
	IdleConnTimeout time.Duration // 空闲连接超时

	// TLS配置
	TLSSkipVerify bool   // 跳过TLS验证
	TLSCertFile   string // TLS证书文件
	TLSKeyFile    string // TLS密钥文件

	// 重试配置
	MaxRetries    int           // 最大重试次数
	RetryInterval time.Duration // 重试间隔

	// 响应处理
	ResponseFormat string // json/xml/plain
	DataPath       string // JSON数据路径 (e.g., "data.items")
}

// Validate 验证配置
func (c *RestAPIConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	if c.AuthType == "" {
		c.AuthType = "none"
	}

	if c.DefaultMethod == "" {
		c.DefaultMethod = "GET"
	}

	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	if c.DialTimeout == 0 {
		c.DialTimeout = 10 * time.Second
	}

	if c.KeepAlive == 0 {
		c.KeepAlive = 30 * time.Second
	}

	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = 90 * time.Second
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.RetryInterval == 0 {
		c.RetryInterval = 1 * time.Second
	}

	if c.PollingInterval == 0 {
		c.PollingInterval = 5 * time.Second
	}

	if c.ResponseFormat == "" {
		c.ResponseFormat = "json"
	}

	return nil
}

// GetType 获取配置类型
func (c *RestAPIConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolHTTP
}

// NewRestAPI 创建REST API数据源
func NewRestAPI(config *RestAPIConfig) (*RestAPI, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建HTTP客户端
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: config.KeepAlive,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		IdleConnTimeout:       config.IdleConnTimeout,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// TLS配置
	if config.TLSSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &RestAPI{
		config:   config,
		client:   client,
		dataChan: make(chan *interfaces.DataUpdate, 1000),
		stopChan: make(chan struct{}),
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}, nil
}

// Open 初始化REST API连接
func (r *RestAPI) Open(devicePath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
	defer cancel()

	endpoint := r.config.DefaultEndpoint
	if endpoint == "" {
		endpoint = "/"
	}

	_, err := r.request(ctx, "GET", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	r.stats.Connected = true
	return nil
}

// Close 关闭REST API连接
func (r *RestAPI) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.monitoring {
		close(r.stopChan)
		r.monitoring = false
	}

	r.client.CloseIdleConnections()
	r.stats.Connected = false
	r.stats.Running = false

	return nil
}

// Get 发送GET请求 - 实现数据读取功能
func (r *RestAPI) Get(ctx context.Context, endpoint string, params map[string]string) (interface{}, error) {
	return r.request(ctx, "GET", endpoint, params, nil)
}

// Post 发送POST请求 - 实现数据写入功能
func (r *RestAPI) Post(ctx context.Context, endpoint string, data interface{}) (interface{}, error) {
	return r.request(ctx, "POST", endpoint, nil, data)
}

// Put 发送PUT请求
func (r *RestAPI) Put(ctx context.Context, endpoint string, data interface{}) (interface{}, error) {
	return r.request(ctx, "PUT", endpoint, nil, data)
}

// Delete 发送DELETE请求
func (r *RestAPI) Delete(ctx context.Context, endpoint string) (interface{}, error) {
	return r.request(ctx, "DELETE", endpoint, nil, nil)
}

// Patch 发送PATCH请求
func (r *RestAPI) Patch(ctx context.Context, endpoint string, data interface{}) (interface{}, error) {
	return r.request(ctx, "PATCH", endpoint, nil, data)
}

// request 发送HTTP请求
func (r *RestAPI) request(ctx context.Context, method string, endpoint string, params map[string]string, body interface{}) (interface{}, error) {
	startTime := time.Now()

	// 构建完整URL
	url := r.config.BaseURL + endpoint

	// 添加查询参数
	if len(params) > 0 || len(r.config.DefaultParams) > 0 {
		query := url + "?"
		for k, v := range r.config.DefaultParams {
			query += fmt.Sprintf("%s=%s&", k, v)
		}
		for k, v := range params {
			query += fmt.Sprintf("%s=%s&", k, v)
		}
		url = query[:len(query)-1] // 移除最后的&
	}

	// 构建请求体
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加默认请求头
	for k, v := range r.config.DefaultHeaders {
		req.Header.Set(k, v)
	}

	// 添加Content-Type
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 添加认证
	r.addAuthentication(req)

	// 执行请求（带重试）
	var resp *http.Response
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.config.RetryInterval)
		}

		resp, err = r.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break // 成功或客户端错误（不重试）
		}
		if err != nil && attempt < r.config.MaxRetries {
			continue
		}
	}

	if err != nil {
		r.stats.ErrorCount++
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	r.stats.RequestCount++
	r.stats.CurrentLatency = time.Since(startTime)

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result interface{}
	switch r.config.ResponseFormat {
	case "json":
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	case "plain":
		result = string(respBody)
	default:
		result = respBody
	}

	return result, nil
}

// addAuthentication 添加认证信息
func (r *RestAPI) addAuthentication(req *http.Request) {
	switch r.config.AuthType {
	case "basic":
		if r.config.Username != "" {
			req.SetBasicAuth(r.config.Username, r.config.Password)
		}

	case "bearer":
		if r.config.Token != "" {
			req.Header.Set("Authorization", "Bearer "+r.config.Token)
		}

	case "apikey":
		if r.config.APIKey != "" {
			req.Header.Set("X-API-Key", r.config.APIKey)
		}
		if r.config.APISecret != "" {
			req.Header.Set("X-API-Secret", r.config.APISecret)
		}
	}
}

// StartPolling 启动数据轮询 - 实现数据监听功能
func (r *RestAPI) StartPolling(ctx context.Context) error {
	r.mu.Lock()
	if r.monitoring {
		r.mu.Unlock()
		return fmt.Errorf("polling already started")
	}

	if r.config.PollingEndpoint == "" {
		r.mu.Unlock()
		return fmt.Errorf("polling endpoint not configured")
	}

	r.monitoring = true
	r.stats.Running = true
	r.mu.Unlock()

	go r.pollingLoop(ctx)

	return nil
}

// StopPolling 停止数据轮询
func (r *RestAPI) StopPolling() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.monitoring {
		return nil
	}

	r.stopChan <- struct{}{}
	r.monitoring = false
	r.stats.Running = false

	return nil
}

// pollingLoop 轮询循环
func (r *RestAPI) pollingLoop(ctx context.Context) {
	ticker := time.NewTicker(r.config.PollingInterval)
	defer ticker.Stop()

	var lastResult interface{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			method := r.config.PollingMethod
			if method == "" {
				method = "GET"
			}

			result, err := r.request(ctx, method, r.config.PollingEndpoint, nil, nil)
			if err != nil {
				continue
			}

			// 检测变化
			if r.isDataChanged(lastResult, result) {
				update := &interfaces.DataUpdate{
					TargetID:   r.config.PollingEndpoint,
					Value:      result,
					OldValue:   lastResult,
					Quality:    1.0,
					Timestamp:  time.Now(),
					ChangeType: "value",
					Metadata: map[string]interface{}{
						"protocol": "rest_api",
						"endpoint": r.config.PollingEndpoint,
						"method":   method,
					},
				}

				select {
				case r.dataChan <- update:
				default:
					// 缓冲区满，丢弃
				}
			}

			lastResult = result
		}
	}
}

// isDataChanged 检测数据是否变化
func (r *RestAPI) isDataChanged(old, new interface{}) bool {
	if old == nil {
		return true
	}

	// 简化比较：序列化后比较字符串
	oldJSON, _ := json.Marshal(old)
	newJSON, _ := json.Marshal(new)

	return string(oldJSON) != string(newJSON)
}

// GetDataChannel 获取数据通道
func (r *RestAPI) GetDataChannel() <-chan *interfaces.DataUpdate {
	return r.dataChan
}

// GetStatus 获取协议状态
func (r *RestAPI) GetStatus() *interfaces.ProtocolStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := *r.stats
	return &status
}

// Read 实现io.Reader接口
func (r *RestAPI) Read(buffer []byte) (int, error) {
	return 0, fmt.Errorf("Read not supported for REST API")
}

// Write 实现io.Writer接口
func (r *RestAPI) Write(data []byte) (int, error) {
	return 0, fmt.Errorf("Write not supported for REST API")
}

// DownloadFile 下载文件
func (r *RestAPI) DownloadFile(ctx context.Context, endpoint string, filepath string) error {
	resp, err := r.client.Get(r.config.BaseURL + endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// UploadFile 上传文件
func (r *RestAPI) UploadFile(ctx context.Context, endpoint string, filepath string, fieldName string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filepath)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", r.config.BaseURL+endpoint, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.addAuthentication(req)

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload failed: HTTP %d", resp.StatusCode)
	}

	return nil
}

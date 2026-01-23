package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_ProFiBus/serial"
)

// 本示例展示：RESTful API数据采集功能
//
// 功能：
// 1. 数据读取 - 通过GET请求获取数据
// 2. 数据监听 - 定时轮询API端点
// 3. 数据写入 - 通过POST/PUT/PATCH请求发送数据
// 4. 多种认证 - Basic/Bearer/API Key

func main() {
	fmt.Println("=== RESTful API数据采集示例 ===\n")
	fmt.Println("通过HTTP/HTTPS协议采集云平台、Web服务的数据\n")

	// RESTful API配置示例
	config := &serial.RestAPIConfig{
		BaseURL: "https://api.example.com", // API基础URL

		// 认证配置（根据实际API选择）
		AuthType: "apikey",              // none/basic/bearer/apikey
		APIKey:   "your-api-key-here",   // API密钥
		APISecret: "your-api-secret-here", // API密钥（如果需要）

		// Basic认证示例:
		// AuthType: "basic",
		// Username: "user",
		// Password: "password",

		// Bearer Token示例:
		// AuthType: "bearer",
		// Token:    "your-bearer-token",

		// 默认请求配置
		DefaultEndpoint: "/api/v1/status",
		DefaultMethod:   "GET",
		DefaultHeaders: map[string]string{
			"User-Agent":   "go_ProFiBus/1.0",
			"Content-Type": "application/json",
		},

		// 轮询配置
		PollingEndpoint: "/api/v1/sensors/data",
		PollingInterval: 10 * time.Second,
		PollingMethod:   "GET",

		// 超时配置
		Timeout:     30 * time.Second,
		DialTimeout: 10 * time.Second,

		// TLS配置
		TLSSkipVerify: false, // 生产环境应设为false

		// 重试配置
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,

		// 响应格式
		ResponseFormat: "json",
	}

	fmt.Println("📋 REST API配置:")
	fmt.Printf("  基础URL: %s\n", config.BaseURL)
	fmt.Printf("  认证方式: %s\n", config.AuthType)
	fmt.Printf("  轮询端点: %s\n", config.PollingEndpoint)
	fmt.Printf("  轮询间隔: %v\n", config.PollingInterval)
	fmt.Printf("  请求超时: %v\n", config.Timeout)
	fmt.Println()

	// 创建REST API数据源
	api, err := serial.NewRestAPI(config)
	if err != nil {
		log.Fatalf("创建REST API失败: %v", err)
	}

	// 打开连接（测试连通性）
	err = api.Open("")
	if err != nil {
		log.Printf("连接REST API失败: %v\n", err)
		log.Println("\n提示: 请确保:")
		log.Println("  1. API服务可访问")
		log.Println("  2. 认证信息正确")
		log.Println("  3. 网络连接正常")
		log.Println("  4. API端点路径正确")
		log.Println("\n将使用模拟模式演示...")
		useSimulationMode()
		return
	}

	defer api.Close()

	fmt.Println("✓ REST API连接成功")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. GET请求演示
	fmt.Println("=== 功能1: GET请求（数据读取）===")
	demonstrateGetRequest(ctx, api)

	// 2. POST请求演示
	fmt.Println("\n=== 功能2: POST请求（数据写入）===")
	demonstratePostRequest(ctx, api)

	// 3. PUT/PATCH请求演示
	fmt.Println("\n=== 功能3: PUT/PATCH请求（数据更新）===")
	demonstratePutPatchRequest(ctx, api)

	// 4. 带参数的请求
	fmt.Println("\n=== 功能4: 参数化请求 ===")
	demonstrateParameterizedRequest(ctx, api)

	// 5. 数据轮询演示
	fmt.Println("\n=== 功能5: 数据轮询（监听）===")
	err = api.StartPolling(ctx)
	if err != nil {
		log.Printf("启动轮询失败: %v", err)
	} else {
		fmt.Println("API轮询已启动")
		go monitorDataUpdates(api)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n--- REST API采集运行中 ---")
	fmt.Println("轮询周期: 10秒")
	fmt.Println("按 Ctrl+C 退出...\n")

	<-sigChan

	fmt.Println("\n\n正在关闭...")
	cancel()
	time.Sleep(500 * time.Millisecond)

	// 显示最终统计
	status := api.GetStatus()
	fmt.Println("\n=== 运行统计 ===")
	fmt.Printf("请求次数: %d\n", status.RequestCount)
	fmt.Printf("错误次数: %d\n", status.ErrorCount)
	fmt.Printf("平均延迟: %v\n", status.AverageLatency)

	fmt.Println("\n✓ 示例完成!")
}

// demonstrateGetRequest 演示GET请求
func demonstrateGetRequest(ctx context.Context, api *serial.RestAPI) {
	endpoints := []string{
		"/api/v1/devices",
		"/api/v1/sensors/temperature",
		"/api/v1/sensors/pressure",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("  GET %s ... ", endpoint)

		result, err := api.Get(ctx, endpoint, nil)
		if err != nil {
			fmt.Printf("失败: %v\n", err)
			continue
		}

		fmt.Printf("成功\n")
		if data, ok := result.(map[string]interface{}); ok {
			fmt.Printf("      响应: %v\n", formatResponse(data))
		}
	}
}

// demonstratePostRequest 演示POST请求
func demonstratePostRequest(ctx context.Context, api *serial.RestAPI) {
	// 发送传感器数据
	data := map[string]interface{}{
		"device_id":   "PLC-001",
		"sensor_type": "temperature",
		"value":       25.5,
		"unit":        "°C",
		"timestamp":   time.Now().Unix(),
	}

	fmt.Printf("  POST /api/v1/sensors/data\n")
	fmt.Printf("  数据: %v\n", data)

	result, err := api.Post(ctx, "/api/v1/sensors/data", data)
	if err != nil {
		fmt.Printf("  失败: %v\n", err)
		return
	}

	fmt.Printf("  ✓ 成功\n")
	if resp, ok := result.(map[string]interface{}); ok {
		fmt.Printf("  响应: %v\n", formatResponse(resp))
	}
}

// demonstratePutPatchRequest 演示PUT/PATCH请求
func demonstratePutPatchRequest(ctx context.Context, api *serial.RestAPI) {
	// PUT请求 - 完整更新
	updateData := map[string]interface{}{
		"device_id": "PLC-001",
		"status":    "active",
		"config": map[string]interface{}{
			"sample_rate": 1000,
			"enabled":     true,
		},
	}

	fmt.Printf("  PUT /api/v1/devices/PLC-001\n")
	result, err := api.Put(ctx, "/api/v1/devices/PLC-001", updateData)
	if err != nil {
		fmt.Printf("  失败: %v\n", err)
	} else {
		fmt.Printf("  ✓ 成功\n")
	}

	// PATCH请求 - 部分更新
	patchData := map[string]interface{}{
		"status": "maintenance",
	}

	fmt.Printf("\n  PATCH /api/v1/devices/PLC-001\n")
	result, err = api.Patch(ctx, "/api/v1/devices/PLC-001", patchData)
	if err != nil {
		fmt.Printf("  失败: %v\n", err)
	} else {
		fmt.Printf("  ✓ 成功\n")
		if resp, ok := result.(map[string]interface{}); ok {
			fmt.Printf("  响应: %v\n", formatResponse(resp))
		}
	}
}

// demonstrateParameterizedRequest 演示参数化请求
func demonstrateParameterizedRequest(ctx context.Context, api *serial.RestAPI) {
	// 带查询参数的GET请求
	params := map[string]string{
		"device_id": "PLC-001",
		"start":     "2024-01-01",
		"end":       "2024-01-31",
		"limit":     "100",
	}

	fmt.Printf("  GET /api/v1/history?")
	for k, v := range params {
		fmt.Printf("%s=%s&", k, v)
	}
	fmt.Println()

	result, err := api.Get(ctx, "/api/v1/history", params)
	if err != nil {
		fmt.Printf("  失败: %v\n", err)
		return
	}

	fmt.Printf("  ✓ 成功\n")
	if data, ok := result.(map[string]interface{}); ok {
		if records, ok := data["records"].([]interface{}); ok {
			fmt.Printf("  返回记录数: %d\n", len(records))
		}
	}
}

// monitorDataUpdates 监听数据更新
func monitorDataUpdates(api *serial.RestAPI) {
	dataChan := api.GetDataChannel()

	updateCount := 0

	for update := range dataChan {
		updateCount++

		fmt.Printf("[轮询] API数据更新 #%d\n", updateCount)
		fmt.Printf("       端点: %s\n", update.TargetID)
		fmt.Printf("       时间: %s\n", update.Timestamp.Format("15:04:05"))

		if data, ok := update.Value.(map[string]interface{}); ok {
			fmt.Printf("       数据: %v\n", formatResponse(data))
		}

		if update.Metadata != nil {
			if method, ok := update.Metadata["method"].(string); ok {
				fmt.Printf("       方法: %s\n", method)
			}
		}

		fmt.Println()
	}
}

// formatResponse 格式化响应数据
func formatResponse(data map[string]interface{}) string {
	if len(data) > 3 {
		// 只显示前3个字段
		count := 0
		result := "{ "
		for k, v := range data {
			if count >= 3 {
				result += "... "
				break
			}
			result += fmt.Sprintf("%s: %v, ", k, v)
			count++
		}
		result += "}"
		return result
	}

	return fmt.Sprintf("%v", data)
}

// useSimulationMode 使用模拟模式演示
func useSimulationMode() {
	fmt.Println("\n=== 模拟模式演示 ===\n")

	fmt.Println("1. GET请求 - 获取设备列表:")
	fmt.Println("   GET https://api.example.com/api/v1/devices")
	fmt.Println("   响应: {\"devices\": [{\"id\": \"PLC-001\", \"status\": \"active\"}, ...]}")

	fmt.Println("\n2. POST请求 - 提交传感器数据:")
	fmt.Println("   POST https://api.example.com/api/v1/sensors/data")
	fmt.Println("   Body: {\"device_id\": \"PLC-001\", \"value\": 25.5, \"unit\": \"°C\"}")
	fmt.Println("   响应: {\"success\": true, \"id\": \"12345\"}")

	fmt.Println("\n3. PUT请求 - 更新设备配置:")
	fmt.Println("   PUT https://api.example.com/api/v1/devices/PLC-001")
	fmt.Println("   Body: {\"status\": \"active\", \"config\": {...}}")
	fmt.Println("   响应: {\"success\": true}")

	fmt.Println("\n4. 带参数的GET请求:")
	fmt.Println("   GET https://api.example.com/api/v1/history?device_id=PLC-001&limit=100")
	fmt.Println("   响应: {\"records\": [...], \"total\": 1250}")

	fmt.Println("\n5. 数据轮询:")
	fmt.Println("   每10秒请求: GET /api/v1/sensors/data")
	fmt.Println("   检测到数据变化 → 发送更新通知")
	fmt.Println("   [轮询] 端点: /api/v1/sensors/data")
	fmt.Println("   [轮询] 数据: {temperature: 26.5, pressure: 1.2}")

	fmt.Println("\nREST API认证方式:")
	fmt.Println("  • None: 无认证")
	fmt.Println("  • Basic: HTTP Basic认证 (用户名+密码)")
	fmt.Println("  • Bearer: Bearer Token (OAuth 2.0)")
	fmt.Println("  • API Key: 自定义API密钥 (X-API-Key header)")

	fmt.Println("\nREST API特性:")
	fmt.Println("  • 支持所有HTTP方法: GET, POST, PUT, PATCH, DELETE")
	fmt.Println("  • 多种认证方式")
	fmt.Println("  • 自动重试机制")
	fmt.Println("  • TLS/SSL支持")
	fmt.Println("  • 请求超时控制")
	fmt.Println("  • 连接池管理")
	fmt.Println("  • 数据轮询监听")

	fmt.Println("\n应用场景:")
	fmt.Println("  • 云平台数据采集 (AWS IoT, Azure IoT)")
	fmt.Println("  • MES/ERP系统集成")
	fmt.Println("  • 第三方服务调用")
	fmt.Println("  • Webhook数据接收")
	fmt.Println("  • 微服务数据交换")
	fmt.Println("  • IoT平台对接")
}

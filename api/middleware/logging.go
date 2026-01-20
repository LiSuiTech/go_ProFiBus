package middleware

import (
	"go_ProFiBus/logger"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	log := logger.GetLogger()

	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 结束时间
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// 根据状态码选择日志级别
		switch {
		case statusCode >= 500:
			log.Error("[API] %3d | %13v | %15s | %-7s %s | %s",
				statusCode, latency, clientIP, method, path, c.Errors.String())
		case statusCode >= 400:
			log.Warn("[API] %3d | %13v | %15s | %-7s %s",
				statusCode, latency, clientIP, method, path)
		default:
			log.Info("[API] %3d | %13v | %15s | %-7s %s",
				statusCode, latency, clientIP, method, path)
		}
	}
}

package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 跨域资源共享中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		// 设置CORS响应头
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Cache-Control, X-Requested-With")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400") // 24小时
		}

		// 处理OPTIONS预检请求
		if method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// CORSWithConfig 带配置的CORS中间件
func CORSWithConfig(allowOrigins []string, allowMethods []string, allowHeaders []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		// 检查origin是否在允许列表中
		allowed := false
		if len(allowOrigins) == 0 || (len(allowOrigins) == 1 && allowOrigins[0] == "*") {
			allowed = true
		} else {
			for _, allowOrigin := range allowOrigins {
				if origin == allowOrigin {
					allowed = true
					break
				}
			}
		}

		if allowed && origin != "" {
			// 使用请求的origin或第一个允许的origin
			if len(allowOrigins) > 0 && allowOrigins[0] == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
			}

			// 设置允许的方法
			if len(allowMethods) > 0 {
				methods := ""
				for i, m := range allowMethods {
					if i > 0 {
						methods += ", "
					}
					methods += m
				}
				c.Header("Access-Control-Allow-Methods", methods)
			}

			// 设置允许的请求头
			if len(allowHeaders) > 0 {
				headers := ""
				for i, h := range allowHeaders {
					if i > 0 {
						headers += ", "
					}
					headers += h
				}
				c.Header("Access-Control-Allow-Headers", headers)
			}

			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// 处理OPTIONS预检请求
		if method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

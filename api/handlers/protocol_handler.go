package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/serial"

	"github.com/gin-gonic/gin"
)

// ProtocolHandler 协议管理处理器
type ProtocolHandler struct {
	protocols map[string]ProtocolInstance // 协议实例管理
}

// ProtocolInstance 协议实例
type ProtocolInstance struct {
	ID          string                    // 实例ID
	Type        interfaces.ProtocolType   // 协议类型
	Config      interface{}               // 配置
	Instance    interface{}               // 协议实例
	Status      *interfaces.ProtocolStatus // 状态
	CreatedAt   time.Time                 // 创建时间
	UpdatedAt   time.Time                 // 更新时间
	Connected   bool                      // 是否已连接
	Running     bool                      // 是否正在运行
}

// NewProtocolHandler 创建协议管理处理器
func NewProtocolHandler() *ProtocolHandler {
	return &ProtocolHandler{
		protocols: make(map[string]ProtocolInstance),
	}
}

// ListProtocols 列出所有支持的协议类型
// @Summary 列出所有支持的协议类型
// @Tags Protocol
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/types [get]
func (h *ProtocolHandler) ListProtocols(c *gin.Context) {
	protocols := []map[string]interface{}{
		{
			"type":        "profibus",
			"name":        "PROFIBUS DP/PA",
			"category":    "fieldbus",
			"description": "Process Field Bus - 过程现场总线",
			"features":    []string{"Master/Slave", "Cyclic", "Diagnostics"},
		},
		{
			"type":        "modbus",
			"name":        "Modbus RTU/TCP/ASCII",
			"category":    "fieldbus",
			"description": "Modbus协议 - 支持RTU/TCP/ASCII模式",
			"features":    []string{"Master/Slave", "Read/Write", "All function codes"},
		},
		{
			"type":        "hart",
			"name":        "HART Protocol",
			"category":    "fieldbus",
			"description": "Highway Addressable Remote Transducer",
			"features":    []string{"4-20mA+Digital", "Point-to-Point", "Multidrop"},
		},
		{
			"type":        "devicenet",
			"name":        "DeviceNet",
			"category":    "fieldbus",
			"description": "CAN-based Fieldbus",
			"features":    []string{"CAN 2.0A", "I/O Polling", "Explicit Messaging"},
		},
		{
			"type":        "profinet",
			"name":        "PROFINET IO",
			"category":    "industrial_ethernet",
			"description": "Process Field Network - 工业以太网",
			"features":    []string{"Real-time", "IRT/RTC", "DCP"},
		},
		{
			"type":        "ethercat",
			"name":        "EtherCAT",
			"category":    "industrial_ethernet",
			"description": "Ethernet for Control Automation Technology",
			"features":    []string{"Ultra fast", "Distributed Clock", "Hot Connect"},
		},
		{
			"type":        "ethernetip",
			"name":        "EtherNet/IP",
			"category":    "industrial_ethernet",
			"description": "Rockwell Automation Industrial Protocol",
			"features":    []string{"CIP", "Implicit/Explicit", "Tag-based"},
		},
		{
			"type":        "opcua",
			"name":        "OPC UA",
			"category":    "it_layer",
			"description": "OPC Unified Architecture",
			"features":    []string{"Client/Server", "Pub/Sub", "Security"},
		},
		{
			"type":        "mqtt",
			"name":        "MQTT",
			"category":    "it_layer",
			"description": "Message Queuing Telemetry Transport",
			"features":    []string{"Pub/Sub", "QoS", "Lightweight"},
		},
		{
			"type":        "restapi",
			"name":        "RESTful API",
			"category":    "it_layer",
			"description": "HTTP/HTTPS REST API",
			"features":    []string{"GET/POST/PUT/DELETE", "Auth", "Polling"},
		},
		{
			"type":        "database",
			"name":        "Database (ODBC)",
			"category":    "it_layer",
			"description": "SQL Database Connection",
			"features":    []string{"MySQL", "PostgreSQL", "SQL Server", "SQLite"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"protocols": protocols,
		"count":     len(protocols),
	})
}

// ListInstances 列出所有协议实例
// @Summary 列出所有协议实例
// @Tags Protocol
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances [get]
func (h *ProtocolHandler) ListInstances(c *gin.Context) {
	instances := make([]map[string]interface{}, 0, len(h.protocols))

	for id, instance := range h.protocols {
		instances = append(instances, map[string]interface{}{
			"id":         id,
			"type":       instance.Type,
			"connected":  instance.Connected,
			"running":    instance.Running,
			"created_at": instance.CreatedAt,
			"updated_at": instance.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"instances": instances,
		"count":     len(instances),
	})
}

// GetInstance 获取协议实例详情
// @Summary 获取协议实例详情
// @Tags Protocol
// @Param id path string true "实例ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances/{id} [get]
func (h *ProtocolHandler) GetInstance(c *gin.Context) {
	id := c.Param("id")

	instance, exists := h.protocols[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Protocol instance not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"instance": instance,
	})
}

// CreateInstance 创建协议实例
// @Summary 创建协议实例
// @Tags Protocol
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "协议配置"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances [post]
func (h *ProtocolHandler) CreateInstance(c *gin.Context) {
	var request struct {
		ID     string                  `json:"id" binding:"required"`
		Type   interfaces.ProtocolType `json:"type" binding:"required"`
		Config json.RawMessage         `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 检查ID是否已存在
	if _, exists := h.protocols[request.ID]; exists {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "Protocol instance already exists",
		})
		return
	}

	// 根据类型创建协议实例
	instance, config, err := h.createProtocolByType(request.Type, request.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to create protocol: %v", err),
		})
		return
	}

	// 保存实例
	h.protocols[request.ID] = ProtocolInstance{
		ID:        request.ID,
		Type:      request.Type,
		Config:    config,
		Instance:  instance,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Connected: false,
		Running:   false,
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Protocol instance created",
		"id":      request.ID,
	})
}

// createProtocolByType 根据类型创建协议实例
func (h *ProtocolHandler) createProtocolByType(protocolType interfaces.ProtocolType, configData json.RawMessage) (interface{}, interface{}, error) {
	switch protocolType {
	case interfaces.ProtocolModbus:
		var config serial.ModbusConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewModbus(&config)
		return instance, &config, err

	case interfaces.ProtocolPROFIBUS:
		var config serial.PROFIBUSConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewPROFIBUS(&config)
		return instance, &config, err

	case interfaces.ProtocolHART:
		var config serial.HARTConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewHART(&config)
		return instance, &config, err

	case interfaces.ProtocolDeviceNet:
		var config serial.DeviceNetConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewDeviceNet(&config)
		return instance, &config, err

	case interfaces.ProtocolPROFINET:
		var config serial.PROFINETConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewPROFINET(&config)
		return instance, &config, err

	case interfaces.ProtocolEtherCAT:
		var config serial.EtherCATConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewEtherCAT(&config)
		return instance, &config, err

	case interfaces.ProtocolEtherNetIP:
		var config serial.EtherNetIPConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewEtherNetIP(&config)
		return instance, &config, err

	case interfaces.ProtocolOPCUA:
		var config serial.OPCUAConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewOPCUA(&config)
		return instance, &config, err

	case interfaces.ProtocolMQTT:
		var config serial.MQTTConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewMQTT(&config)
		return instance, &config, err

	case interfaces.ProtocolHTTP:
		var config serial.RestAPIConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewRestAPI(&config)
		return instance, &config, err

	case interfaces.ProtocolDatabase:
		var config serial.DatabaseConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, nil, err
		}
		instance, err := serial.NewDatabase(&config)
		return instance, &config, err

	default:
		return nil, nil, fmt.Errorf("unsupported protocol type: %s", protocolType)
	}
}

// ConnectInstance 连接协议实例
// @Summary 连接协议实例
// @Tags Protocol
// @Param id path string true "实例ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances/{id}/connect [post]
func (h *ProtocolHandler) ConnectInstance(c *gin.Context) {
	id := c.Param("id")

	instance, exists := h.protocols[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Protocol instance not found",
		})
		return
	}

	// 调用Open方法连接
	var err error
	switch proto := instance.Instance.(type) {
	case interface{ Open(string) error }:
		err = proto.Open("")
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Protocol does not support connection",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to connect: %v", err),
		})
		return
	}

	// 更新状态
	instance.Connected = true
	instance.UpdatedAt = time.Now()
	h.protocols[id] = instance

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Protocol connected successfully",
	})
}

// DisconnectInstance 断开协议实例
// @Summary 断开协议实例
// @Tags Protocol
// @Param id path string true "实例ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances/{id}/disconnect [post]
func (h *ProtocolHandler) DisconnectInstance(c *gin.Context) {
	id := c.Param("id")

	instance, exists := h.protocols[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Protocol instance not found",
		})
		return
	}

	// 调用Close方法断开
	switch proto := instance.Instance.(type) {
	case interface{ Close() error }:
		if err := proto.Close(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to disconnect: %v", err),
			})
			return
		}
	}

	// 更新状态
	instance.Connected = false
	instance.Running = false
	instance.UpdatedAt = time.Now()
	h.protocols[id] = instance

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Protocol disconnected successfully",
	})
}

// StartMonitoring 启动数据监听
// @Summary 启动数据监听
// @Tags Protocol
// @Param id path string true "实例ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances/{id}/monitor/start [post]
func (h *ProtocolHandler) StartMonitoring(c *gin.Context) {
	id := c.Param("id")

	instance, exists := h.protocols[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Protocol instance not found",
		})
		return
	}

	if !instance.Connected {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Protocol is not connected",
		})
		return
	}

	// 根据协议类型启动监听
	ctx := context.Background()
	var err error

	switch proto := instance.Instance.(type) {
	case interface{ StartCyclicExchange(context.Context) error }:
		err = proto.StartCyclicExchange(ctx)
	case interface{ StartPolling(context.Context) error }:
		err = proto.StartPolling(ctx)
	case interface{ StartCyclicIO(context.Context) error }:
		err = proto.StartCyclicIO(ctx)
	case interface{ StartIOPolling(context.Context) error }:
		err = proto.StartIOPolling(ctx)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Protocol does not support monitoring",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to start monitoring: %v", err),
		})
		return
	}

	// 更新状态
	instance.Running = true
	instance.UpdatedAt = time.Now()
	h.protocols[id] = instance

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Monitoring started successfully",
	})
}

// StopMonitoring 停止数据监听
// @Summary 停止数据监听
// @Tags Protocol
// @Param id path string true "实例ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances/{id}/monitor/stop [post]
func (h *ProtocolHandler) StopMonitoring(c *gin.Context) {
	id := c.Param("id")

	instance, exists := h.protocols[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Protocol instance not found",
		})
		return
	}

	// 根据协议类型停止监听
	switch proto := instance.Instance.(type) {
	case interface{ StopPolling() error }:
		proto.StopPolling()
	case interface{ StopCyclicIO() error }:
		proto.StopCyclicIO()
	}

	// 更新状态
	instance.Running = false
	instance.UpdatedAt = time.Now()
	h.protocols[id] = instance

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Monitoring stopped successfully",
	})
}

// GetStatus 获取协议状态
// @Summary 获取协议状态
// @Tags Protocol
// @Param id path string true "实例ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances/{id}/status [get]
func (h *ProtocolHandler) GetStatus(c *gin.Context) {
	id := c.Param("id")

	instance, exists := h.protocols[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Protocol instance not found",
		})
		return
	}

	// 获取协议状态
	var status *interfaces.ProtocolStatus
	switch proto := instance.Instance.(type) {
	case interface{ GetStatus() *interfaces.ProtocolStatus }:
		status = proto.GetStatus()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  status,
		"connected": instance.Connected,
		"running": instance.Running,
	})
}

// DeleteInstance 删除协议实例
// @Summary 删除协议实例
// @Tags Protocol
// @Param id path string true "实例ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols/instances/{id} [delete]
func (h *ProtocolHandler) DeleteInstance(c *gin.Context) {
	id := c.Param("id")

	instance, exists := h.protocols[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Protocol instance not found",
		})
		return
	}

	// 如果已连接，先断开
	if instance.Connected {
		switch proto := instance.Instance.(type) {
		case interface{ Close() error }:
			proto.Close()
		}
	}

	// 删除实例
	delete(h.protocols, id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Protocol instance deleted",
	})
}

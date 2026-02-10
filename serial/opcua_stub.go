//go:build !gopcua

// 未使用 -tags=gopcua 时的桩实现，避免 gopcua 库 API 与当前代码不兼容导致编译失败。
// 需要 OPC-UA 功能时请使用: go build -tags gopcua ./cmd/server 并适配 serial/opcua.go 中的 gopcua API。

package serial

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OPCUA OPC-UA协议实现（桩，未链接 gopcua 时使用）
type OPCUA struct {
	config    *OPCUAConfig
	dataChan  chan []byte
	stopChan  chan struct{}
	connected bool
	mu        sync.RWMutex
	closeOnce sync.Once
	ctx       context.Context
	cancel    context.CancelFunc
}

// OPCUAConfig OPC-UA配置
type OPCUAConfig struct {
	Endpoint       string
	SecurityPolicy string
	SecurityMode   string
	Username       string
	Password       string
	CertFile       string
	KeyFile        string
	NodeID         string
	SampleInterval time.Duration
}

// NewOPCUA 创建OPC-UA实例（桩）
func NewOPCUA(config *OPCUAConfig) *OPCUA {
	if config.SampleInterval == 0 {
		config.SampleInterval = 1 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &OPCUA{
		config:   config,
		dataChan: make(chan []byte, 1000),
		stopChan: make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func errNotCompiled() error { return fmt.Errorf("OPC-UA support not compiled in (build with -tags gopcua)") }

func (o *OPCUA) Open(devicePath string) error   { return errNotCompiled() }
func (o *OPCUA) Write(data []byte) (int, error) { return 0, errNotCompiled() }
func (o *OPCUA) Read(buffer []byte) (int, error) { return 0, errNotCompiled() }
func (o *OPCUA) Close() error                  { o.closeOnce.Do(func() { o.cancel(); close(o.stopChan) }); return nil }
func (o *OPCUA) ReadNode(nodeID string) (interface{}, error) { return nil, errNotCompiled() }
func (o *OPCUA) WriteNode(nodeID string, value interface{}) error { return errNotCompiled() }
func (o *OPCUA) CallMethod(objectID, methodID string, args ...interface{}) ([]interface{}, error) {
	return nil, errNotCompiled()
}
func (o *OPCUA) IsConnected() bool { return false }

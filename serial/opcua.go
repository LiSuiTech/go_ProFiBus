package serial

// NOTE: This file requires gopcua v0.3.x or compatible version
// Current compilation errors are due to API changes in different gopcua versions
// TODO: Update to use the correct gopcua API for your installed version

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// OPCUA OPC-UA协议实现
type OPCUA struct {
	client   *opcua.Client
	config   *OPCUAConfig
	dataChan chan []byte
	stopChan chan struct{}
	connected bool
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// OPCUAConfig OPC-UA配置
type OPCUAConfig struct {
	Endpoint       string        // OPC-UA服务器端点
	SecurityPolicy string        // 安全策略
	SecurityMode   string        // 安全模式
	Username       string        // 用户名
	Password       string        // 密码
	CertFile       string        // 证书文件
	KeyFile        string        // 密钥文件
	NodeID         string        // 要读写的节点ID
	SampleInterval time.Duration // 采样间隔
}

// NewOPCUA 创建OPC-UA实例
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

// Open 连接OPC-UA服务器
func (o *OPCUA) Open(devicePath string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 如果提供了devicePath，使用它作为endpoint
	endpoint := o.config.Endpoint
	if devicePath != "" {
		endpoint = devicePath
	}

	opts := []opcua.Option{
		opcua.SecurityPolicy(o.config.SecurityPolicy),
		opcua.SecurityModeString(o.config.SecurityMode),
	}

	if o.config.Username != "" {
		opts = append(opts, opcua.AuthUsername(o.config.Username, o.config.Password))
	}

	if o.config.CertFile != "" && o.config.KeyFile != "" {
		opts = append(opts, opcua.CertificateFile(o.config.CertFile), opcua.PrivateKeyFile(o.config.KeyFile))
	}

	client, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		return fmt.Errorf("创建OPC-UA客户端失败: %w", err)
	}

	if err := client.Connect(o.ctx); err != nil {
		return fmt.Errorf("连接OPC-UA服务器失败: %w", err)
	}

	o.client = client
	o.connected = true

	// 启动数据采集
	go o.collectLoop()

	return nil
}

// Write 写入数据到OPC-UA节点
func (o *OPCUA) Write(data []byte) (int, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.connected || o.client == nil {
		return 0, fmt.Errorf("OPC-UA未连接")
	}

	if o.config.NodeID == "" {
		return 0, fmt.Errorf("未配置节点ID")
	}

	nodeID, err := ua.ParseNodeID(o.config.NodeID)
	if err != nil {
		return 0, fmt.Errorf("无效的节点ID: %w", err)
	}

	// 将数据转换为OPC-UA值
	value := &ua.Variant{
		Type:  ua.TypeIDByteString,
		Value: data,
	}

	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					Value: value,
				},
			},
		},
	}

	resp, err := o.client.Write(o.ctx, req)
	if err != nil {
		return 0, fmt.Errorf("写入节点失败: %w", err)
	}

	if len(resp.Results) > 0 && !resp.Results[0].IsGood() {
		return 0, fmt.Errorf("写入状态: %s", resp.Results[0])
	}

	return len(data), nil
}

// Read 从OPC-UA节点读取数据
func (o *OPCUA) Read(buffer []byte) (int, error) {
	o.mu.RLock()
	connected := o.connected
	o.mu.RUnlock()

	if !connected {
		return 0, fmt.Errorf("OPC-UA未连接")
	}

	// 从通道读取数据
	select {
	case data := <-o.dataChan:
		n := copy(buffer, data)
		return n, nil
	case <-time.After(5 * time.Second):
		return 0, fmt.Errorf("读取超时")
	}
}

// Close 关闭OPC-UA连接
func (o *OPCUA) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	close(o.stopChan)
	o.cancel()

	if o.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		o.client.Close(ctx)
	}

	o.connected = false
	return nil
}

// ReadNode 读取指定节点
func (o *OPCUA) ReadNode(nodeID string) (interface{}, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.connected || o.client == nil {
		return nil, fmt.Errorf("OPC-UA未连接")
	}

	id, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("无效的节点ID: %w", err)
	}

	req := &ua.ReadRequest{
		MaxAge:             2000,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: id},
		},
	}

	resp, err := o.client.Read(o.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("读取节点失败: %w", err)
	}

	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("没有返回结果")
	}

	if !resp.Results[0].StatusCode().IsGood() {
		return nil, fmt.Errorf("读取状态: %s", resp.Results[0].StatusCode())
	}

	return resp.Results[0].Value(), nil
}

// WriteNode 写入指定节点
func (o *OPCUA) WriteNode(nodeID string, value interface{}) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.connected || o.client == nil {
		return fmt.Errorf("OPC-UA未连接")
	}

	id, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return fmt.Errorf("无效的节点ID: %w", err)
	}

	// 创建变量
	variant, err := ua.NewVariant(value)
	if err != nil {
		return fmt.Errorf("创建变量失败: %w", err)
	}

	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      id,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					Value: variant,
				},
			},
		},
	}

	resp, err := o.client.Write(o.ctx, req)
	if err != nil {
		return fmt.Errorf("写入节点失败: %w", err)
	}

	if len(resp.Results) > 0 && !resp.Results[0].IsGood() {
		return fmt.Errorf("写入状态: %s", resp.Results[0])
	}

	return nil
}

// CallMethod 调用OPC-UA方法
func (o *OPCUA) CallMethod(objectID, methodID string, args ...interface{}) ([]interface{}, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.connected || o.client == nil {
		return nil, fmt.Errorf("OPC-UA未连接")
	}

	objID, err := ua.ParseNodeID(objectID)
	if err != nil {
		return nil, fmt.Errorf("无效的对象ID: %w", err)
	}

	methID, err := ua.ParseNodeID(methodID)
	if err != nil {
		return nil, fmt.Errorf("无效的方法ID: %w", err)
	}

	// 转换参数
	variants := make([]*ua.Variant, len(args))
	for i, arg := range args {
		v, err := ua.NewVariant(arg)
		if err != nil {
			return nil, fmt.Errorf("参数%d转换失败: %w", i, err)
		}
		variants[i] = v
	}

	req := &ua.CallMethodRequest{
		ObjectID:       objID,
		MethodID:       methID,
		InputArguments: variants,
	}

	resp, err := o.client.Call(o.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("调用方法失败: %w", err)
	}

	if !resp.StatusCode.IsGood() {
		return nil, fmt.Errorf("调用状态: %s", resp.StatusCode)
	}

	// 转换返回值
	results := make([]interface{}, len(resp.OutputArguments))
	for i, v := range resp.OutputArguments {
		results[i] = v.Value()
	}

	return results, nil
}

// IsConnected 检查连接状态
func (o *OPCUA) IsConnected() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.connected
}

// collectLoop 数据采集循环
func (o *OPCUA) collectLoop() {
	if o.config.NodeID == "" {
		return
	}

	ticker := time.NewTicker(o.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopChan:
			return
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.collectData()
		}
	}
}

// collectData 采集数据
func (o *OPCUA) collectData() {
	value, err := o.ReadNode(o.config.NodeID)
	if err != nil {
		return
	}

	// 将值转换为字节
	data := fmt.Sprintf("%v", value)

	select {
	case o.dataChan <- []byte(data):
	default:
		// 缓冲区满，丢弃数据
	}
}

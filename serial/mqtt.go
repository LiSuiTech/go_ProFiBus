package serial

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTT MQTT协议实现
type MQTT struct {
	client    mqtt.Client
	config    *MQTTConfig
	dataChan  chan []byte
	connected bool
	mu        sync.RWMutex
}

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Broker       string   // MQTT代理地址 (e.g., "tcp://localhost:1883")
	ClientID     string   // 客户端ID
	Username     string   // 用户名
	Password     string   // 密码
	Topics       []string // 订阅主题列表
	QoS          byte     // QoS等级 (0, 1, 2)
	CleanSession bool     // 清除会话
	KeepAlive    int      // 保活时间（秒）
	UseTLS       bool     // 是否使用TLS
	TLSInsecure  bool     // 是否跳过TLS验证
}

// NewMQTT 创建MQTT实例
func NewMQTT(config *MQTTConfig) *MQTT {
	if config.KeepAlive == 0 {
		config.KeepAlive = 60
	}
	if config.ClientID == "" {
		config.ClientID = fmt.Sprintf("go_profibus_%d", time.Now().Unix())
	}

	return &MQTT{
		config:   config,
		dataChan: make(chan []byte, 1000),
	}
}

// Open 连接MQTT代理
func (m *MQTT) Open(devicePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果提供了devicePath，使用它作为broker地址
	broker := m.config.Broker
	if devicePath != "" {
		broker = devicePath
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(m.config.ClientID)
	opts.SetCleanSession(m.config.CleanSession)
	opts.SetKeepAlive(time.Duration(m.config.KeepAlive) * time.Second)
	opts.SetAutoReconnect(true)

	if m.config.Username != "" {
		opts.SetUsername(m.config.Username)
		opts.SetPassword(m.config.Password)
	}

	if m.config.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: m.config.TLSInsecure,
		}
		opts.SetTLSConfig(tlsConfig)
	}

	opts.SetOnConnectHandler(m.onConnect)
	opts.SetConnectionLostHandler(m.onConnectionLost)

	m.client = mqtt.NewClient(opts)
	token := m.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT连接超时")
	}
	if token.Error() != nil {
		return fmt.Errorf("MQTT连接失败: %w", token.Error())
	}

	m.connected = true
	return nil
}

// Write 发布MQTT消息
func (m *MQTT) Write(data []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected || m.client == nil {
		return 0, fmt.Errorf("MQTT未连接")
	}

	// 默认发布到第一个主题
	if len(m.config.Topics) == 0 {
		return 0, fmt.Errorf("没有配置发布主题")
	}

	topic := m.config.Topics[0]
	token := m.client.Publish(topic, m.config.QoS, false, data)
	if !token.WaitTimeout(5 * time.Second) {
		return 0, fmt.Errorf("发布消息超时")
	}
	if token.Error() != nil {
		return 0, token.Error()
	}

	return len(data), nil
}

// Read 读取MQTT消息
func (m *MQTT) Read(buffer []byte) (int, error) {
	m.mu.RLock()
	connected := m.connected
	m.mu.RUnlock()

	if !connected {
		return 0, fmt.Errorf("MQTT未连接")
	}

	// 从通道读取数据
	select {
	case data := <-m.dataChan:
		n := copy(buffer, data)
		return n, nil
	case <-time.After(5 * time.Second):
		return 0, fmt.Errorf("读取超时")
	}
}

// Close 关闭MQTT连接
func (m *MQTT) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil && m.client.IsConnected() {
		// 取消订阅
		for _, topic := range m.config.Topics {
			m.client.Unsubscribe(topic)
		}
		m.client.Disconnect(250)
	}

	m.connected = false
	return nil
}

// Publish 发布消息到指定主题
func (m *MQTT) Publish(topic string, payload interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected || m.client == nil {
		return fmt.Errorf("MQTT未连接")
	}

	var data []byte
	var err error

	// 根据payload类型处理
	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("序列化payload失败: %w", err)
		}
	}

	token := m.client.Publish(topic, m.config.QoS, false, data)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("发布消息超时")
	}

	return token.Error()
}

// Subscribe 订阅主题
func (m *MQTT) Subscribe(topic string) error {
	m.mu.Lock()
	m.config.Topics = append(m.config.Topics, topic)
	m.mu.Unlock()

	if m.client != nil && m.client.IsConnected() {
		token := m.client.Subscribe(topic, m.config.QoS, m.messageHandler)
		if !token.WaitTimeout(5 * time.Second) {
			return fmt.Errorf("订阅超时")
		}
		return token.Error()
	}

	return nil
}

// GetDataChannel 获取数据通道
func (m *MQTT) GetDataChannel() <-chan []byte {
	return m.dataChan
}

// IsConnected 检查连接状态
func (m *MQTT) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// onConnect 连接成功回调
func (m *MQTT) onConnect(client mqtt.Client) {
	m.mu.Lock()
	m.connected = true
	topics := m.config.Topics
	qos := m.config.QoS
	m.mu.Unlock()

	// 订阅所有主题
	for _, topic := range topics {
		client.Subscribe(topic, qos, m.messageHandler)
	}
}

// onConnectionLost 连接丢失回调
func (m *MQTT) onConnectionLost(client mqtt.Client, err error) {
	m.mu.Lock()
	m.connected = false
	m.mu.Unlock()
}

// messageHandler 消息处理器
func (m *MQTT) messageHandler(client mqtt.Client, msg mqtt.Message) {
	select {
	case m.dataChan <- msg.Payload():
	default:
		// 缓冲区满，丢弃消息
	}
}

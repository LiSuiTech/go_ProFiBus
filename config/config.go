package config

import (
	"fmt"
	"go_ProFiBus/logger"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

// Config 全局配置
type Config struct {
	System     SystemConfig     `yaml:"system"`
	Logging    LoggingConfig    `yaml:"logging"`
	Protocols  []ProtocolConfig `yaml:"protocols"`
	Collector  CollectorConfig  `yaml:"collector"`
	Fusion     FusionConfig     `yaml:"fusion"`
	Inference  InferenceConfig  `yaml:"inference"`
	Multimodal MultimodalConfig `yaml:"multimodal"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"` // dev, test, prod
	Debug       bool   `yaml:"debug"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `yaml:"level"`       // DEBUG, INFO, WARN, ERROR, FATAL
	EnableFile bool   `yaml:"enable_file"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size_mb"` // 日志文件最大大小(MB)
}

// ProtocolConfig 协议配置
type ProtocolConfig struct {
	ID          string            `yaml:"id"`
	Type        string            `yaml:"type"`        // UART, RS485, I2C, SPI, etc.
	Port        string            `yaml:"port"`        // 端口路径
	BaudRate    int               `yaml:"baud_rate"`
	DataBits    int               `yaml:"data_bits"`
	StopBits    int               `yaml:"stop_bits"`
	Parity      string            `yaml:"parity"`      // none, odd, even
	I2CAddress  int               `yaml:"i2c_address"`
	Enabled     bool              `yaml:"enabled"`
	Metadata    map[string]string `yaml:"metadata"`
}

// CollectorConfig 数据采集配置
type CollectorConfig struct {
	BufferSize       int    `yaml:"buffer_size"`
	DefaultSampleRate string `yaml:"default_sample_rate"` // 如 "100ms", "1s"
	EnableCache      bool   `yaml:"enable_cache"`
	CacheSize        int    `yaml:"cache_size"`
	RetryCount       int    `yaml:"retry_count"`
	RetryDelay       string `yaml:"retry_delay"`
}

// FusionConfig 数据融合配置
type FusionConfig struct {
	Strategy       string `yaml:"strategy"`        // average, weighted, kalman
	TimeWindow     string `yaml:"time_window"`     // 如 "1s", "100ms"
	DefaultWeights map[string]float64 `yaml:"default_weights"`
}

// InferenceConfig 推理配置
type InferenceConfig struct {
	Models []ModelConfig `yaml:"models"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Name       string   `yaml:"name"`
	Type       string   `yaml:"type"`        // linear_regression, neural_network, etc.
	ModelPath  string   `yaml:"model_path"`
	InputShape []int    `yaml:"input_shape"`
	Enabled    bool     `yaml:"enabled"`
}

// MultimodalConfig 多模态配置
type MultimodalConfig struct {
	Alignment      string `yaml:"alignment"`       // nearest_neighbor, linear_interp, dtw
	AnalyzeInterval string `yaml:"analyze_interval"` // 如 "1s"
	EnabledModalities []string `yaml:"enabled_modalities"`
}

// LoadConfig 从YAML文件加载配置
func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("无法解析配置文件: %v", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return &config, nil
}

// SaveConfig 保存配置到YAML文件
func SaveConfig(filepath string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("无法序列化配置: %v", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("无法写入配置文件: %v", err)
	}

	return nil
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证系统配置
	if config.System.Name == "" {
		return fmt.Errorf("系统名称不能为空")
	}

	// 验证日志级别
	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
		"FATAL": true,
	}
	if !validLevels[config.Logging.Level] {
		return fmt.Errorf("无效的日志级别: %s", config.Logging.Level)
	}

	// 验证协议配置
	for i, proto := range config.Protocols {
		if proto.Port == "" && proto.Enabled {
			return fmt.Errorf("协议 %d: 端口不能为空", i)
		}
	}

	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *Config {
	return &Config{
		System: SystemConfig{
			Name:        "go_ProFiBus",
			Version:     "1.0.0",
			Environment: "dev",
			Debug:       true,
		},
		Logging: LoggingConfig{
			Level:      "INFO",
			EnableFile: true,
			FilePath:   "logs/profibus.log",
			MaxSize:    100,
		},
		Protocols: []ProtocolConfig{
			{
				ID:       "rs485_1",
				Type:     "RS485",
				Port:     "/dev/ttyUSB0",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Enabled:  true,
			},
		},
		Collector: CollectorConfig{
			BufferSize:       1000,
			DefaultSampleRate: "100ms",
			EnableCache:      true,
			CacheSize:        100,
			RetryCount:       3,
			RetryDelay:       "100ms",
		},
		Fusion: FusionConfig{
			Strategy:   "weighted",
			TimeWindow: "1s",
			DefaultWeights: map[string]float64{
				"rs485_1": 1.0,
			},
		},
		Inference: InferenceConfig{
			Models: []ModelConfig{
				{
					Name:       "anomaly_detector",
					Type:       "neural_network",
					ModelPath:  "models/anomaly_detector.bin",
					InputShape: []int{-1, 10},
					Enabled:    false,
				},
			},
		},
		Multimodal: MultimodalConfig{
			Alignment:       "linear_interp",
			AnalyzeInterval: "1s",
			EnabledModalities: []string{
				"time_series",
				"sensor",
			},
		},
	}
}

// ParseDuration 解析时间字符串
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// GetLogLevel 获取日志级别
func GetLogLevel(levelStr string) logger.LogLevel {
	switch levelStr {
	case "DEBUG":
		return logger.DEBUG
	case "INFO":
		return logger.INFO
	case "WARN":
		return logger.WARN
	case "ERROR":
		return logger.ERROR
	case "FATAL":
		return logger.FATAL
	default:
		return logger.INFO
	}
}

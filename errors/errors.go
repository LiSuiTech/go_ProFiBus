package errors

import "fmt"

// ErrorCode 定义错误代码
type ErrorCode int

const (
	// 通用错误
	ErrUnknown ErrorCode = iota + 1000
	ErrInvalidParam
	ErrNotImplemented

	// 串口错误 (2000-2999)
	ErrPortNotOpen     ErrorCode = iota + 2000
	ErrPortOpenFailed
	ErrPortCloseFailed
	ErrPortReadFailed
	ErrPortWriteFailed
	ErrPortConfigFailed
	ErrInvalidBaudRate
	ErrInvalidDataBits
	ErrInvalidStopBits
	ErrInvalidParity

	// 协议错误 (3000-3999)
	ErrProtocolNotSupported ErrorCode = iota + 3000
	ErrProtocolInitFailed
	ErrInvalidFrame
	ErrCRCCheckFailed
	ErrFrameTooShort
	ErrFrameTooLong

	// 数据采集错误 (4000-4999)
	ErrCollectorNotRunning ErrorCode = iota + 4000
	ErrCollectorAlreadyRunning
	ErrCollectorStartFailed
	ErrCollectorStopFailed
	ErrInvalidSampleRate

	// 模型推理错误 (5000-5999)
	ErrModelNotLoaded ErrorCode = iota + 5000
	ErrModelLoadFailed
	ErrInferenceFailed
	ErrInvalidInputShape
	ErrInvalidOutputShape

	// 数据融合错误 (6000-6999)
	ErrFusionFailed ErrorCode = iota + 6000
	ErrInvalidDataSource
	ErrDataMismatch
	ErrSyncFailed
)

var errorMessages = map[ErrorCode]string{
	// 通用错误
	ErrUnknown:        "未知错误",
	ErrInvalidParam:   "无效参数",
	ErrNotImplemented: "功能未实现",

	// 串口错误
	ErrPortNotOpen:      "串口未打开",
	ErrPortOpenFailed:   "打开串口失败",
	ErrPortCloseFailed:  "关闭串口失败",
	ErrPortReadFailed:   "读取串口数据失败",
	ErrPortWriteFailed:  "写入串口数据失败",
	ErrPortConfigFailed: "配置串口失败",
	ErrInvalidBaudRate:  "无效的波特率",
	ErrInvalidDataBits:  "无效的数据位",
	ErrInvalidStopBits:  "无效的停止位",
	ErrInvalidParity:    "无效的校验位",

	// 协议错误
	ErrProtocolNotSupported: "不支持的协议",
	ErrProtocolInitFailed:   "协议初始化失败",
	ErrInvalidFrame:         "无效的数据帧",
	ErrCRCCheckFailed:       "CRC校验失败",
	ErrFrameTooShort:        "数据帧太短",
	ErrFrameTooLong:         "数据帧太长",

	// 数据采集错误
	ErrCollectorNotRunning:     "数据采集器未运行",
	ErrCollectorAlreadyRunning: "数据采集器已在运行",
	ErrCollectorStartFailed:    "启动数据采集器失败",
	ErrCollectorStopFailed:     "停止数据采集器失败",
	ErrInvalidSampleRate:       "无效的采样率",

	// 模型推理错误
	ErrModelNotLoaded:     "模型未加载",
	ErrModelLoadFailed:    "加载模型失败",
	ErrInferenceFailed:    "推理失败",
	ErrInvalidInputShape:  "无效的输入形状",
	ErrInvalidOutputShape: "无效的输出形状",

	// 数据融合错误
	ErrFusionFailed:      "数据融合失败",
	ErrInvalidDataSource: "无效的数据源",
	ErrDataMismatch:      "数据不匹配",
	ErrSyncFailed:        "数据同步失败",
}

// ProFiBusError 自定义错误类型
type ProFiBusError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error 实现 error 接口
func (e *ProFiBusError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现错误链
func (e *ProFiBusError) Unwrap() error {
	return e.Err
}

// New 创建新的错误
func New(code ErrorCode, err error) *ProFiBusError {
	message := errorMessages[code]
	if message == "" {
		message = errorMessages[ErrUnknown]
	}

	return &ProFiBusError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Newf 创建新的错误并格式化消息
func Newf(code ErrorCode, format string, args ...interface{}) *ProFiBusError {
	return &ProFiBusError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     nil,
	}
}

// Wrap 包装已有错误
func Wrap(code ErrorCode, err error) *ProFiBusError {
	return New(code, err)
}

// Is 检查错误代码
func Is(err error, code ErrorCode) bool {
	if e, ok := err.(*ProFiBusError); ok {
		return e.Code == code
	}
	return false
}

package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// Logger 日志记录器
type Logger struct {
	level      LogLevel
	logger     *log.Logger
	mu         sync.Mutex
	enableFile bool
	file       *os.File
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// GetLogger 获取默认日志记录器（单例）
func GetLogger() *Logger {
	once.Do(func() {
		defaultLogger = &Logger{
			level:  INFO,
			logger: log.New(os.Stdout, "", 0),
		}
	})
	return defaultLogger
}

// NewLogger 创建新的日志记录器
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// EnableFileLog 启用文件日志
func (l *Logger) EnableFileLog(filename string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %v", err)
	}

	l.file = file
	l.enableFile = true
	l.logger = log.New(file, "", 0)
	return nil
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// log 内部日志记录方法
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, v...)

	logMessage := fmt.Sprintf("[%s] [%s] %s", timestamp, levelName, message)

	if l.enableFile && l.file != nil {
		l.logger.Println(logMessage)
	} else {
		fmt.Println(logMessage)
	}

	// FATAL 级别直接退出
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 输出调试级别日志
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

// Info 输出信息级别日志
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

// Warn 输出警告级别日志
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, format, v...)
}

// Error 输出错误级别日志
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

// Fatal 输出致命错误级别日志并退出程序
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(FATAL, format, v...)
}

// 包级别的便捷函数
func Debug(format string, v ...interface{}) {
	GetLogger().Debug(format, v...)
}

func Info(format string, v ...interface{}) {
	GetLogger().Info(format, v...)
}

func Warn(format string, v ...interface{}) {
	GetLogger().Warn(format, v...)
}

func Error(format string, v ...interface{}) {
	GetLogger().Error(format, v...)
}

func Fatal(format string, v ...interface{}) {
	GetLogger().Fatal(format, v...)
}

func SetLevel(level LogLevel) {
	GetLogger().SetLevel(level)
}

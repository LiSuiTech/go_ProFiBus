package logger

import (
	"testing"
)

func TestLogger(t *testing.T) {
	log := GetLogger()

	t.Run("SetLevel", func(t *testing.T) {
		log.SetLevel(DEBUG)
		log.Debug("This is a debug message")
		log.Info("This is an info message")
		log.Warn("This is a warning message")
		log.Error("This is an error message")
	})

	t.Run("DifferentLevels", func(t *testing.T) {
		log.SetLevel(WARN)
		log.Debug("This debug should not appear")
		log.Info("This info should not appear")
		log.Warn("This warning should appear")
		log.Error("This error should appear")
	})
}

func TestNewLogger(t *testing.T) {
	log := NewLogger(INFO)

	log.Info("Testing new logger instance")
	log.Warn("Warning from new logger")
}

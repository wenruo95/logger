package logger

import (
	"testing"
)

func Test_logger(t *testing.T) {
	logger := NewLogger("/tmp/logs/test.log")
	for i := 0; i < 100; i++ {
		logger.Debug("[%d] test:debug", i)
		logger.Info("[%d] test:info", i)
		logger.Error("[%d] test:error", i)
		logger.Warning("[%d] test:warning", i)
	}
}

func Test_loggerArgs(t *testing.T) {
	logger := NewLoggerArgs("/tmp/logs/test.log", 0, nil)
	for i := 0; i < 100; i++ {
		logger.Debug("[%d] test:debug", i)
		logger.Info("[%d] test:info", i)
		logger.Error("[%d] test:error", i)
		logger.Warning("[%d] test:warning", i)
	}
}

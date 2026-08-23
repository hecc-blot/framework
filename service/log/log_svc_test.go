package service

import (
	"context"
	"testing"

	logConfig "github.com/hecc-blot/framework/config/log"
	"github.com/hecc-blot/framework/enum/trace"

	"github.com/stretchr/testify/assert"
)

var localConf = &logConfig.LocalConfig{
	RootDir:    "./runtime/logs",
	MaxSize:    1,
	MaxBackups: 3,
	MaxAge:     7,
	Compress:   false,
}

func TestLogSvc(t *testing.T) {
	logger, err := newLogSvc(localConf)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	ctx := context.Background()

	cases := []struct {
		level string
		text  string
	}{
		{
			level: "debug",
			text:  "test debug",
		},
		{
			level: "info",
			text:  "test info",
		},
		{
			level: "warn",
			text:  "test warn",
		},
		{
			level: "error",
			text:  "test error",
		},
	}
	t.Run("log", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.level, func(t *testing.T) {
				switch c.level {
				case "debug":
					logger.Debug(ctx, c.text)
				case "info":
					logger.Info(ctx, c.text)
				case "warn":
					logger.Warn(ctx, c.text)
				case "error":
					logger.Error(ctx, c.text)
				}
			})
		}
	})

}

func TestLogSvcWithTrace(t *testing.T) {
	logger, err := newLogSvc(localConf)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	ctxWithTraceId := context.WithValue(context.Background(), trace.TraceIdKey, "test-trace-id")

	cases := []struct {
		level string
		text  string
	}{
		{
			level: "debug",
			text:  "test debug",
		},
		{
			level: "info",
			text:  "test info",
		},
		{
			level: "warn",
			text:  "test warn",
		},
		{
			level: "error",
			text:  "test error",
		},
	}

	t.Run("log with trace", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.level, func(t *testing.T) {
				switch c.level {
				case "debug":
					logger.Debug(ctxWithTraceId, c.text)
				case "info":
					logger.Info(ctxWithTraceId, c.text)
				case "warn":
					logger.Warn(ctxWithTraceId, c.text)
				case "error":
					logger.Error(ctxWithTraceId, c.text)
				}
			})
		}
	})
}

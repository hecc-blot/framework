package log

import (
	"context"
)

type ILog interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
}

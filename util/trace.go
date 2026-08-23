package util

import (
	"context"

	"github.com/hecc-blot/framework/enum/trace"
)

// ExtractTraceId 从 context 提取链路追踪 ID（无则返回空串）。
func ExtractTraceId(ctx context.Context) string {
	ctx = ExtractContext(ctx)
	traceId := ctx.Value(trace.TraceIdKey)
	if traceId != nil {
		return traceId.(string)
	}
	return ""
}

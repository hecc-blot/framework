package util

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ExtractContext 从 *gin.Context 中提取 context.Context
func ExtractContext(ctx context.Context) context.Context {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return ginCtx.Request.Context()
	}
	return ctx
}

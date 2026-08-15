package sse

import "github.com/gin-gonic/gin"

type ISse interface {
	Serve(ctx *gin.Context) error
}

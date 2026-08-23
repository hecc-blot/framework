package api

import (
	coreError "github.com/hecc-blot/core/contract/error"

	"github.com/gin-gonic/gin"
)

type IApi interface {
	Call(ctx *gin.Context) (interface{}, coreError.IError)
}

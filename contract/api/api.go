package api

import (
	coreError "github.com/hecc-blot/framework/contract/error"
)

type IApi interface {
	Call(ctx IContext) (interface{}, coreError.IError)
}

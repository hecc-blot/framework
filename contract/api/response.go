package api

import (
	coreError "github.com/hecc-blot/framework/contract/error"
)

type IResponse interface {
	Regular(ctx IContext, data interface{}, err coreError.IError)
}

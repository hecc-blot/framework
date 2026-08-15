package api

import (
	"context"

	coreError "github.com/hecc-blot/hecc-blot-core/contract/error"
)

type IResponse interface {
	Regular(ctx context.Context, data interface{}, err coreError.IError)
}

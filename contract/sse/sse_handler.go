package sse

import iCoreApi "github.com/hecc-blot/hecc-blot-core/contract/api"

type ISseHandle interface {
	Get(apiPath string, sse ISse)
	Middleware(middlewares ...iCoreApi.IMiddleware) ISseHandle
}

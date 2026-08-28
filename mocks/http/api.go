package mocks

import (
	"github.com/hecc-blot/framework/contract/api"
	coreError "github.com/hecc-blot/framework/contract/error"
)

// MockResponse 是 IResponse 接口的 mock 实现，记录最近一次响应的 data 与 err。
type MockResponse struct {
	Data any
	Err  coreError.IError
}

func (m *MockResponse) Regular(ctx api.IContext, data any, err coreError.IError) {
	m.Data = data
	m.Err = err
}

// MockMiddleware 是 IMiddleware 接口的 mock 实现，可通过 Fn 定制中间件行为。
type MockMiddleware struct {
	Fn api.MiddlewareFunc
}

func (m *MockMiddleware) Middleware() api.MiddlewareFunc {
	if m.Fn != nil {
		return m.Fn
	}
	return func(ctx api.IContext) { ctx.Next() }
}

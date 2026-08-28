package api

// MiddlewareFunc 强类型中间件函数：接收框架自有 IContext，屏蔽具体 HTTP 内核。
type MiddlewareFunc func(ctx IContext)

// IMiddleware 中间件契约。
//
// 实现方通过 Middleware() 返回强类型函数，取代原先返回 interface{} 的弱类型设计
// （避免靠 reflect 断言兜底、类型安全弱的问题）。
type IMiddleware interface {
	Middleware() MiddlewareFunc
}

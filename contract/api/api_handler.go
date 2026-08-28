package api

import "net/http"

type IApiHandle interface {
	Get(apiPath string, api interface{})
	Post(apiPath string, api interface{})
	Middleware(middlewares ...IMiddleware) IApiHandle
	Group(relativePath string, middlewares ...IMiddleware) IApiHandle
	// Handle 注册原生 http.Handler（如 Prometheus 采集端点、SSE 流等需要直接读写连接的能力），
	// 不经过 IApi 的参数绑定与响应包装；中间件/分组语义同 Get/Post。
	Handle(method, path string, handler http.Handler)
	Listen(onShutdown ...func())
}

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

	// SetResponseSvc 注入 API 响应渲染实现（把 Call 的返回值包成 {code, message, data}）。
	//
	// 它只服务 Get/Post 注册的 API 路由；用 Handle 挂原生 handler 的内核
	// （纯 WS / SSE 服务、Prometheus 采集端口）不需要它，也就**不必构造**。
	// 未注入就注册 Get/Post 会在**注册期** panic，而不是等到第一个请求才 nil 崩。
	// 传 nil 忽略（与 OnShutdown(nil) 一致）。
	//
	// 值在注册路由时读取，因此需要在注册 Get/Post 之前调用。
	SetResponseSvc(response IResponse)

	// OnShutdown 注册关闭钩子：收到 SIGINT/SIGTERM 后、http.Server.Shutdown 之前执行。
	//
	// 这个位置是承重的，只有框架内部够得着：Shutdown 会先关 listener、再等活跃连接变 idle，
	// 而 SSE / WS 这类长连接永远不会自己变 idle —— 必须有钩子先发关闭帧并取消连接，
	// handler 才能返回、Shutdown 才能迅速收尾。挪到 Listen 返回之后没有等价替代：
	// 那时连接已随进程退出被带走，帧发出去没人收，客户端只能突然断连。
	//
	// 约定（框架无法代劳，只能靠实现方遵守）：
	//   - 钩子必须**幂等** —— Go 中函数值除 nil 外不可比较，框架无法去重；
	//   - 每个钩子独立 recover，一个钩子 panic 不影响后续钩子与 Shutdown；
	//   - 按注册顺序执行，但各钩子之间不应有依赖（顺序不是契约）。
	//
	// 长连接模块（sse / ws）应在**构造时**注册一次，而不是在每个路由注册点注册，
	// 否则多路由会注册多份。
	OnShutdown(fn func())

	// Listen 启动服务并阻塞，直到收到 SIGINT/SIGTERM 后优雅停机。
	Listen()
}

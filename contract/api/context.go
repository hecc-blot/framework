package api

import "context"

// IContext 框架自有的 HTTP 请求上下文，屏蔽具体 HTTP 内核（当前为 gin）。
//
// 契约层与业务层只依赖本接口，不 import 具体 HTTP 框架类型；
// 换 HTTP 内核时只需在实现层替换（framework/service/http 的 gin 实现），无需改动契约与业务代码。
// 只透出取参 / 响应 / 上下文三类框架能力，隐藏内核实现细节。
type IContext interface {
	context.Context

	// ---- 取参 ----
	Param(key string) string     // 路径参数，如 /account/:id 中的 id
	Query(key string) string     // URL 查询参数
	GetHeader(key string) string // 请求头
	ClientIP() string            // 客户端 IP
	Method() string              // 请求方法（GET/POST/...）
	Path() string                // 请求路径（不含 query）
	FullPath() string            // 路由模板（如 /account/:id），未匹配返回 ""

	// ---- 请求级键值（跨中间件/API 传递数据） ----
	Set(key string, value any)
	Get(key string) (value any, exists bool)

	// ---- 中间件链路控制 ----
	Next()  // 继续执行后续 handler
	Abort() // 终止后续 handler

	// ---- 上下文传播 ----
	// SetRequestContext 替换底层请求的 context，供链路追踪等中间件注入上下文。
	SetRequestContext(ctx context.Context)

	// ---- 响应 ----
	JSON(code int, obj any)                        // 写 JSON 响应
	String(code int, format string, values ...any) // 写字符串响应
	Header(key, value string)                      // 设置响应头
	Status() int                                   // 当前响应状态码
	AbortWithStatus(code int)                      // 终止并写状态码
	AbortWithStatusJSON(code int, obj any)         // 终止并写 JSON
}

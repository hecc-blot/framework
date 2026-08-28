package service

import (
	"context"

	iCoreApi "github.com/hecc-blot/framework/contract/api"

	"github.com/gin-gonic/gin"
)

// Context 是契约层 IContext 的 gin 实现，包装 *gin.Context 并隐藏其类型。
// 通过内嵌 *gin.Context 直接透出取参/响应/链路控制等方法，业务层仅能通过 IContext 访问，不可见 gin。
type Context struct {
	*gin.Context
}

// NewContext 将 *gin.Context 包装为契约层 IContext。
func NewContext(c *gin.Context) iCoreApi.IContext {
	return &Context{Context: c}
}

// AdaptMiddleware 将契约层强类型中间件适配为 gin.HandlerFunc。
// 供 http 与 sse 等内核把 IMiddleware.Middleware() 注册到 gin 路由时使用。
func AdaptMiddleware(mw iCoreApi.MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		mw(NewContext(c))
	}
}

// Method 返回请求方法。
func (c *Context) Method() string { return c.Request.Method }

// Path 返回请求路径（不含 query）。
func (c *Context) Path() string { return c.Request.URL.Path }

// Set 存入请求级键值。键收敛为 string，屏蔽 gin 的 any 键。
func (c *Context) Set(key string, value any) { c.Context.Set(key, value) }

// Get 读取请求级键值。键收敛为 string，屏蔽 gin 的 any 键。
func (c *Context) Get(key string) (value any, exists bool) { return c.Context.Get(key) }

// Status 返回当前响应状态码（读取语义，屏蔽 gin 的 Status(code) 写语义）。
func (c *Context) Status() int { return c.Writer.Status() }

// SetRequestContext 替换底层请求的 context，供链路追踪等中间件注入上下文。
func (c *Context) SetRequestContext(ctx context.Context) {
	c.Request = c.Request.WithContext(ctx)
}

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hecc-blot/core/contract/ioc"
	traceEnum "github.com/hecc-blot/core/enum/trace"
	iocSvc "github.com/hecc-blot/core/service/ioc"
	coreUtil "github.com/hecc-blot/core/util"
	serverConfig "github.com/hecc-blot/framework/config/http"
	iCoreApi "github.com/hecc-blot/framework/contract/api"
	coreError "github.com/hecc-blot/framework/contract/error"
	envEnum "github.com/hecc-blot/framework/enum/env"
	"github.com/hecc-blot/framework/enum/response"

	"github.com/stretchr/testify/assert"
)

// ============================ 测试用 API 与中间件 ============================

// greeter 用于验证依赖注入贯穿 API 注册到每请求实例
type greeter interface {
	Greet(name string) string
}

type echoGreeter struct{}

func (e *echoGreeter) Greet(name string) string { return "hello " + name }

// echoApi 无依赖注入，仅验证参数绑定 + 校验 + 响应包装
type echoApi struct {
	Name string `form:"name" binding:"required"`
}

func (a *echoApi) Call(ctx iCoreApi.IContext) (any, coreError.IError) {
	return a.Name, nil
}

// injectApi 注入接口依赖，验证 registerAPI 的 Inject 链路
type injectApi struct {
	Greeter greeter `inject:""`
	Name    string  `form:"name"`
}

func (a *injectApi) Call(ctx iCoreApi.IContext) (any, coreError.IError) {
	return a.Greeter.Greet(a.Name), nil
}

type headerMiddleware struct{}

func (m *headerMiddleware) Middleware() iCoreApi.MiddlewareFunc {
	return func(ctx iCoreApi.IContext) {
		ctx.Header("X-Test-Middleware", "hit")
		ctx.Next()
	}
}

// ============================ 工具函数 ============================

func newTestHandle(t *testing.T, container ioc.IContainer) *ApiHandle {
	t.Helper()
	handle := NewApiSvc(
		&serverConfig.Config{Env: envEnum.TestMode},
		container,
	).(*ApiHandle)
	handle.SetResponseSvc(NewResponseSvc())
	return handle
}

func doRequest(t *testing.T, h *ApiHandle, method, path, body string) (*httptest.ResponseRecorder, *ResponseSvc) {
	t.Helper()
	w := httptest.NewRecorder()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	h.Engine().ServeHTTP(w, req)

	var resp ResponseSvc
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	return w, &resp
}

// ============================ 测试 ============================

func TestNewApiSvc(t *testing.T) {
	t.Run("nil 容器 panic", func(t *testing.T) {
		assert.Panics(t, func() {
			NewApiSvc(&serverConfig.Config{Env: envEnum.TestMode}, nil)
		})
	})

	t.Run("非法环境 panic", func(t *testing.T) {
		assert.Panics(t, func() {
			NewApiSvc(&serverConfig.Config{Env: envEnum.Value("invalid")}, iocSvc.New())
		})
	})
}

// TestSetResponseSvc：响应渲染实现只服务 Get/Post，因此改为按需注入 ——
// 纯原生 handler 的内核（WS / SSE / Prometheus 采集端点）不必构造一个用不上的东西。
func TestSetResponseSvc(t *testing.T) {
	newHandle := func() *ApiHandle {
		return NewApiSvc(&serverConfig.Config{Env: envEnum.TestMode}, iocSvc.New()).(*ApiHandle)
	}

	t.Run("未注入就注册 API 路由，注册期 panic", func(t *testing.T) {
		assert.Panics(t, func() { newHandle().Get("/echo", &echoApi{}) })
	})

	t.Run("注入后可注册并正常渲染", func(t *testing.T) {
		handle := newHandle()
		handle.SetResponseSvc(NewResponseSvc())
		handle.Get("/echo", &echoApi{})

		w, resp := doRequest(t, handle, http.MethodGet, "/echo?name=hecc", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, response.Success, resp.Code)
		assert.Equal(t, "hecc", resp.Data)
	})

	t.Run("先建分组、后在根上注入也生效", func(t *testing.T) {
		// response 在分组间用指针共享，否则这个顺序会让分组永远拿不到实现
		handle := newHandle()
		group := handle.Group("admin")
		handle.SetResponseSvc(NewResponseSvc())

		group.Get("/echo", &echoApi{})

		w, resp := doRequest(t, handle, http.MethodGet, "/admin/echo?name=hecc", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, response.Success, resp.Code)
	})

	t.Run("只挂原生 handler 的内核不需要响应实现", func(t *testing.T) {
		// 纯 WS / SSE 服务的形态：全程不 SetResponseSvc
		handle := newHandle()
		handle.Handle(http.MethodGet, "/raw", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}))

		w := httptest.NewRecorder()
		handle.Engine().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/raw", nil))
		assert.Equal(t, http.StatusTeapot, w.Code)
	})

	t.Run("传 nil 忽略，内核仍视为未注入", func(t *testing.T) {
		handle := newHandle()
		handle.SetResponseSvc(nil)
		assert.Panics(t, func() { handle.Get("/echo", &echoApi{}) })
	})
}

const testTraceId = "4bf92f3577b34da6a3ce929d0e0e4736"

// traceIdApi 返回从 IContext 取到的 traceId，用于验证类型化 key 的透传。
type traceIdApi struct{}

func (a *traceIdApi) Call(ctx iCoreApi.IContext) (any, coreError.IError) {
	return coreUtil.ExtractTraceId(ctx), nil
}

// traceInjectMiddleware 模拟 trace.NewHttpMiddleware：把带 traceId 的 ctx 写回 request。
type traceInjectMiddleware struct{}

func (m *traceInjectMiddleware) Middleware() iCoreApi.MiddlewareFunc {
	return func(ctx iCoreApi.IContext) {
		ctx.SetRequestContext(context.WithValue(context.Background(), traceEnum.TraceIdKey, testTraceId))
		ctx.Next()
	}
}

// TestContextValueFallback IContext 必须能读到写进 Request.Context() 的类型化 key。
//
// 回归防护：gin 的 ContextWithFallback 默认为 false，此时 *gin.Context 的
// Value/Deadline/Done/Err 都不委托给 Request.Context()；而 core/enum/trace 的
// TraceIdKey 是**定义类型**（`type Value string`，不是 string 别名），也走不到 gin
// 里 `key.(string)` 那条分支 —— 两者叠加会让 traceId 在业务日志与 otel span 里静默丢失。
// 若 NewApiSvc 漏开该开关，本用例即失败。
func TestContextValueFallback(t *testing.T) {
	handle := newTestHandle(t, iocSvc.New())
	handle.Middleware(&traceInjectMiddleware{}) // 必须在注册路由之前
	handle.Get("/trace", &traceIdApi{})

	_, resp := doRequest(t, handle, http.MethodGet, "/trace", "")
	assert.Equal(t, response.Success, resp.Code)
	assert.Equal(t, testTraceId, resp.Data)
}

func TestGetRoute(t *testing.T) {
	handle := newTestHandle(t, iocSvc.New())
	handle.Get("/echo", &echoApi{})

	t.Run("成功绑定并响应", func(t *testing.T) {
		_, resp := doRequest(t, handle, http.MethodGet, "/echo?name=world", "")
		assert.Equal(t, response.Success, resp.Code)
		assert.Equal(t, response.CodeMap[response.Success], resp.Message)
		assert.Equal(t, "world", resp.Data)
	})

	t.Run("缺失必填参数校验失败", func(t *testing.T) {
		_, resp := doRequest(t, handle, http.MethodGet, "/echo", "")
		assert.Equal(t, response.ValidateError, resp.Code)
		assert.Equal(t, response.CodeMap[response.ValidateError], resp.Message)
	})
}

func TestPostRoute(t *testing.T) {
	handle := newTestHandle(t, iocSvc.New())
	handle.Post("/echo", &echoApi{})

	_, resp := doRequest(t, handle, http.MethodPost, "/echo", "name=posted")
	assert.Equal(t, response.Success, resp.Code)
	assert.Equal(t, "posted", resp.Data)
}

func TestMiddleware(t *testing.T) {
	handle := newTestHandle(t, iocSvc.New())
	handle.Middleware(&headerMiddleware{})
	handle.Get("/echo", &echoApi{})

	w, _ := doRequest(t, handle, http.MethodGet, "/echo?name=x", "")
	assert.Equal(t, "hit", w.Header().Get("X-Test-Middleware"))
}

func TestGroup(t *testing.T) {
	handle := newTestHandle(t, iocSvc.New())
	group := handle.Group("/v1", &headerMiddleware{})
	group.Get("/echo", &echoApi{})

	w, resp := doRequest(t, handle, http.MethodGet, "/v1/echo?name=group", "")
	assert.Equal(t, "hit", w.Header().Get("X-Test-Middleware"))
	assert.Equal(t, "group", resp.Data)
}

func TestDependencyInjection(t *testing.T) {
	container := iocSvc.New()
	container.Set(new(greeter), &echoGreeter{})

	handle := newTestHandle(t, container)
	handle.Get("/hello", &injectApi{})

	_, resp := doRequest(t, handle, http.MethodGet, "/hello?name=blot", "")
	assert.Equal(t, response.Success, resp.Code)
	assert.Equal(t, "hello blot", resp.Data)
}

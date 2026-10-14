# hecc-blot-framework

框架内核：HTTP 内核（路由 + 中间件 + 参数校验 + 统一响应）、统一错误、分页工具。它是所有业务方与其它模块的公共依赖层。

- [安装](#安装)
- [快速开始](#快速开始)
- [接口定义](#接口定义)
- [使用](#使用)
- [配置](#配置)
- [语义与约定](#语义与约定)
- [相关模块](#相关模块)

## 安装

```bash
go get github.com/hecc-blot/framework
```

## 快速开始

```go
import (
	iCoreApi "github.com/hecc-blot/framework/contract/api"
	httpSvc "github.com/hecc-blot/framework/service/http"
)

func main() {
	// ... 前面的组装（日志 / trace / db / cache / container.Set）见下面「前置条件」

	apiHandle := httpSvc.NewApiSvc(&config.Server, container)

	// 响应渲染只服务 Get/Post 注册的 API 路由；纯 WS / SSE 服务没有这行也能跑
	apiHandle.SetResponseSvc(httpSvc.NewResponseSvc())


	// trace 中间件决定 traceId 能否进入日志与下游调用；不注册则日志里没有 traceId
	apiHandle.Middleware(trace.NewHttpMiddleware(traceSvc))

	apiHandle.Post("account/add", &AddAccountApi{})
	apiHandle.Get("account/list", &ListAccountApi{})

	apiHandle.Listen() // 阻塞，收到 SIGINT/SIGTERM 后优雅停机
}
```

**前置条件**：

- `container` 必须由 `ioc.New()`（`github.com/hecc-blot/core/service/ioc`）创建，且**依赖已全部 `Set` 完毕**——注册路由时会对 API 结构体 `Inject`，依赖未注册直接 panic。
- `NewApiSvc` 在 `container` 为 nil 或 `config.Env` 非法时 panic。
- `SetResponseSvc` **可以完全不调用** —— 那是纯原生 handler 内核（只跑 WS / SSE 服务、Prometheus 采集端点的进程）的形态。但注册 `Get`/`Post` 之前必须先注入，否则**注册期** panic（它只管 API 路由的响应包装，与 `Handle` 无关）。
- 完整的应用组装顺序（日志 → trace → db/cache → 容器 → 路由 → Listen）见[组织主页](https://github.com/hecc-blot)。

## 接口定义

```go
import (
	iCoreApi "github.com/hecc-blot/framework/contract/api"
	coreError "github.com/hecc-blot/framework/contract/error"
)

type IApiHandle interface {
	Get(apiPath string, api interface{})
	Post(apiPath string, api interface{})
	Middleware(middlewares ...IMiddleware) IApiHandle
	Group(relativePath string, middlewares ...IMiddleware) IApiHandle
	// Handle 注册原生 http.Handler（如 Prometheus 采集端点、SSE 流），
	// 不经过参数绑定与响应包装；中间件/分组语义同 Get/Post。
	Handle(method, path string, handler http.Handler)
	// SetResponseSvc 注入 API 响应渲染实现：只服务 Get/Post 注册的路由，
	// 用 Handle 挂原生 handler 的内核（纯 WS / SSE 服务）不必调用。
	// 未注入就注册 Get/Post 会在注册期 panic；传 nil 忽略。
	SetResponseSvc(response IResponse)
	// OnShutdown 注册关闭钩子：收到 SIGINT/SIGTERM 后、http.Server.Shutdown 之前执行。
	// 长连接模块（sse / ws）在**构造时**注册一次，业务无需手工传参。
	// 钩子须幂等；每个钩子独立 recover，一个 panic 不影响后续钩子与 Shutdown。
	OnShutdown(fn func())
	// Listen 阻塞运行；收到 SIGINT/SIGTERM 后先依次执行通过 OnShutdown 注册的钩子，
	// 再以 5s 超时优雅关闭 HTTP 服务。
	Listen()
}

type IApi interface {
	Call(ctx IContext) (interface{}, coreError.IError)
}

type MiddlewareFunc func(ctx IContext)

type IMiddleware interface {
	Middleware() MiddlewareFunc
}

type IResponse interface {
	Regular(ctx IContext, data interface{}, err coreError.IError)
}

type IValidator interface {
	GetMessages() apiEntity.Messages
}
```

> `IApiHandle` 契约里**没有** `Engine()`。想拿底层 `*gin.Engine` 只能对具体类型 `*httpSvc.ApiHandle` 做断言，属于不受支持的用法。

### IContext

框架自有的请求上下文，屏蔽具体 HTTP 内核（当前为 gin）。业务代码只依赖它，不 import gin。

```go
type IContext interface {
	context.Context

	// 取参
	Param(key string) string     // 路径参数，如 /account/:id 中的 id
	Query(key string) string     // URL 查询参数
	GetHeader(key string) string // 请求头
	ClientIP() string
	Method() string
	Path() string                // 请求路径（不含 query）
	FullPath() string            // 路由模板（如 /account/:id），未匹配返回 ""

	// 请求级键值（跨中间件/API 传递数据）
	Set(key string, value any)
	Get(key string) (value any, exists bool)

	// 中间件链路控制
	Next()
	Abort()

	// 上下文传播
	SetRequestContext(ctx context.Context)

	// 响应
	JSON(code int, obj any)
	String(code int, format string, values ...any)
	Header(key, value string)
	Status() int
	AbortWithStatus(code int)
	AbortWithStatusJSON(code int, obj any)
}
```

## 使用

### 注册路由

```go
apiHandle.Post("account/add", &AddAccountApi{})
apiHandle.Get("account/list", &ListAccountApi{})

// 分组：不同分组用不同中间件
apiGroup := apiHandle.Group("admin", &demo.TokenMiddleware{})
apiGroup.Get("account/page", &PageListApi{})

// 原生 handler：绕过参数绑定与响应包装（metrics 端点、SSE 等）
apiHandle.Handle(http.MethodGet, "/metrics", metricsSvc.Handler())
```

### 定义 API

```go
import (
	iCoreApi "github.com/hecc-blot/framework/contract/api"
	coreError "github.com/hecc-blot/framework/contract/error"
	"github.com/hecc-blot/framework/enum/response"
	errorSvc "github.com/hecc-blot/framework/service/error"
)

type AddAccountRequest struct {
	Name   string `form:"name" binding:"required"`
	Amount int    `form:"amount" binding:"gte=0"`
}

type AddAccountApi struct {
	Db     dbContract.IDb   `inject:""`   // 注入字段在前
	LogSvc logContract.ILog `inject:""`
	AddAccountRequest                      // 请求参数（匿名嵌入）在后
}

func (a *AddAccountApi) Call(ctx iCoreApi.IContext) (interface{}, coreError.IError) {
	if a.Amount > 1000 {
		return nil, errorSvc.NewError(response.Fail, errors.New("amount too large"))
	}
	a.LogSvc.Info(ctx, "add account", zap.String("name", a.Name))
	return map[string]any{"ok": true}, nil
}
```

> **请求参数必须匿名嵌入**（不是具名字段）。参数校验失败时的自定义错误消息依赖方法提升——`GetMessages()` 定义在请求结构体上，只有它被匿名嵌入 API 结构体才能被找到（见[参数校验](#参数校验)）。

### 定义中间件

```go
type TokenMiddleware struct {
	ResponseSvc iCoreApi.IResponse `inject:""` // 中间件同样走注入
}

func (t *TokenMiddleware) Middleware() iCoreApi.MiddlewareFunc {
	return func(ctx iCoreApi.IContext) {
		if ctx.GetHeader("Authorization") == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": response.TokenInvalid})
			return
		}
		ctx.Next()
	}
}
```

中间件在注册时就会被 `Inject`，因此注册必须发生在 `container.Set` 之后。

### 请求处理流程

```
请求 → gin 中间件链 → registerAPI 闭包
                          ├─ reflect.New(具体类型)      ← 每请求独立实例
                          ├─ container.Inject(newInstance) ← 重新注入依赖
                          ├─ ShouldBind(参数绑定 + 校验)
                          ├─ api.Call(ctx)            ← 业务逻辑
                          └─ responseSvc.Regular(...)  ← 包装成 {code,message,data}
```

### 参数校验

绑定与校验发生在**每个请求**的 handler 里（不是注册时）。失败时返回 `response.ValidateError`（40002），消息按三级优先级取：

1. 自定义消息（`IValidator.GetMessages()`）
2. `validator` 默认英文消息
3. 原始 error

自定义消息：

```go
func (r AddAccountRequest) GetMessages() apiEntity.Messages {
	return apiEntity.Messages{
		"Name.required":   "账户名不能为空",
		"Amount.gte":      "金额不能为负数",
	}
}
```

key 为 `字段名.Tag`。

### 统一错误与响应

`Call` 返回 `(interface{}, IError)`，成功时 error 传 nil：

```go
// 业务错误
return nil, errorSvc.NewError(response.Fail, err)
// 自定义消息
return nil, errorSvc.Newf(response.Fail, "账户 %s 不存在", id)
```

响应统一为 `{code, message, data}`，HTTP 状态码恒为 200，`code` 见下表。

**三者的分工是契约：**

- **`code`**：客户端分支的唯一依据（按码判断，不要解析文案）。
- **`message`**：按码生成的**通用短语**，给机器看 —— 可枚举、可测、客户端可自行本地化。错误纹理见下方「core 码与对外码的映射」。
- **`data`**：成功时是业务数据；**失败时是给用户看的业务文案**（业务用 `errorSvc.New(code, "用户可见文案")` 传）。走 `errorSvc.From(err)` 的错误不带文案，`data` 为 `null`。

```go
type IError interface {
	error
	GetCode() response.Value
	GetData() interface{}
}
```

| 响应码 | 值 | 消息 |
|--------|-----|------|
| `Success` | 10000 | success |
| `Processing` | 10001 | processing |
| `Fail` | 40000 | request failed |
| `Busy` | 40001 | server busy |
| `ValidateError` | 40002 | invalid parameters |
| `TokenInvalid` | 40003 | invalid token |
| `AccessDenied` | 40004 | access denied |
| `NoDataPermission` | 40005 | no permission to process data |
| `RateLimit` | 40006 | too many requests |
| `Illegal` | 50000 | illegal request |
| `Panic` | 50001 | internal server error |

#### core 码与对外码的映射

`response.Value` 是 `core/enum/errorcode.Code` 的**别名** —— framework 段（10000 / 40000 / 50000…）与 core 段（20000–29999）是同一个类型，边界可以直接把 core 码发出去，无需类型转换。

默认**直用（透传）**：`errorSvc.From(err)` 会取 err 链上的 core 码（穿透 `fmt.Errorf` 的 `%w`），原样进入响应。不想让 core 段出现在响应里，就按需注册翻译 —— 逐码、增量，没有"翻译模式"开关，因为对外码表是对客户端的契约，不该随部署配置变化：

```go
errorSvc.MapCode(errorcode.CodeTimeout, response.Busy) // 此后 20001 对外一律显示 40001
```

`message` 按三跳回退：`response.CodeMap`（协议码文案）→ `errorcode.Message`（core 段文案）→ `Fail` 的兜底文案（业务自定义码 60000+ 框架无从知道文案，落在这里，不会发出空 message）。

业务侧因此不必再"回忆这个错该返回哪个码"：

```go
if err := locker.Lock(ctx, key, ttl); err != nil {
    return nil, errorSvc.From(err) // 带码的 sentinel → 该码；普通错误 → Fail
}
```

> `errorSvc.From(nil)` 返回 `nil`，可以放心直写 `return nil, errorSvc.From(err)`。
> 各 leaf 模块的契约型 sentinel（`lock` / `idempotent` / `circuitbreaker` / `core/util`）正在逐批改为携带 core 码；尚未带码的会落到 `Fail`。

### 分页

```go
import "github.com/hecc-blot/framework/util"

// Offset/Limit 分页
page := util.NewPage[Account](list, pageNum, pageSize, total)
// → { list, page, pageSize, total }

// 游标分页（适合大偏移量场景）
cursor := util.NewCursor[Account](list, nextCursor, hasMore)
```

### 替换组件

框架面向接口编程，替换组件只需实现对应契约并注册到容器：

```go
// 1. 实现 ILog（来自 core）
type LogrusLogSvc struct{ logger *logrus.Logger }

func (l *LogrusLogSvc) Info(ctx context.Context, msg string, fields ...interface{}) { ... }
// ... 其余三个方法

// 2. 覆盖默认实现
container.Set(new(logContract.ILog), &LogrusLogSvc{logger: logrus.New()})
```

要点：必须实现接口的**所有方法**且签名完全匹配；需要资源清理的组件应返回清理函数由调用方 `defer`。

## 配置

```yaml
server:
  env: dev               # dev / test / product → gin 的 DebugMode / TestMode / ReleaseMode
  port: "8080"
  read_timeout: 30       # 秒，<=0 用默认 30
  write_timeout: 30      # 秒；-1 = 不设写超时，0 / 未配置 = 默认 30
  idle_timeout: 60       # 秒，<=0 用默认 60
  body_size_limit: 10485760  # 请求体上限（字节），示例为 10MB
```

`env` 非法时 `NewApiSvc` 直接 panic。

## 语义与约定

- **容器并发**：`container.Set` 仅允许在启动初始化阶段（单线程）调用；此后容器只读，`Inject`/`Get` 可安全并发。**运行时禁止再 `Set`**。
- **每请求实例隔离**：`registerAPI` 对每个请求 `reflect.New` 出独立实例再注入，因此 API 结构体**不要在注入字段上保存请求态数据**。
- **panic 条件**：
  - `NewApiSvc` 的 `container` 为 nil → panic
  - `config.Env` 非法 → panic
  - 注册 `Get`/`Post` 时未 `SetResponseSvc` → panic（原生 handler 不受影响）
  - 依赖未注册 → `Inject` panic
  - 路由重复注册 → gin panic
- **`ContextWithFallback` 是 traceId 生效的前提**：`NewApiSvc` 内部设置了 `app.ContextWithFallback = true`。gin 默认不把 `*gin.Context` 的 `Value/Deadline/Done/Err` 委托给 `Request.Context()`，而 `core/enum/trace` 的 `TraceIdKey` 是定义类型（非 string 别名），两者叠加会让 traceId 在**日志、otel span、MQ 消息头**里全部静默丢失，且 `ctx.Done()` 恒为 nil（取消传播失效）。回归测试见 `service/http/http_svc_test.go` 的 `TestContextValueFallback`。
- **优雅停机**：`Listen` 收到信号后**先跑 `OnShutdown` 注册的钩子**（如通知 SSE 客户端断开），再以 5s 超时 `srv.Shutdown`。钩子同步串行执行，不要在里面做耗时操作；每个钩子独立 recover，一个 panic 不影响后续钩子与 `Shutdown`；钩子须**幂等**（Go 中函数值不可比较，框架无法去重）。
- **写超时**：`read_timeout` / `idle_timeout` 为 `<= 0`（含未配置）时取默认 30s / 60s；`write_timeout` 特殊 —— **负值表示不设写超时**（约定写 `-1`），`0` / 未配置仍取默认 30s。之所以不用 `0` 表示关闭：配置的零值就是 `0`，分不清"没配"与"配了 0"，用 `0` 当开关等于顺手拿掉慢客户端的兜底。
- **长连接与写 deadline**：`net/http` 在读完请求头时一次性设下**绝对**写 deadline，`Flush()` 不会顺延 —— 长连接会准时在 `write_timeout` 那一刻静默断流。SSE handler 内部已自行清掉该 deadline（`http.NewResponseController(w).SetWriteDeadline(time.Time{})`），**因此 SSE 不受 `write_timeout` 限制**；若你也写长连接 handler，记得同样处理（**WebSocket 是例外**：升级会劫持连接，`net/http` 在劫持处已把连接 deadline 清零，WS 天然不受它约束，见 websocket 模块 README）。

## 相关模块

| 模块 | 说明 |
|------|------|
| [core](https://github.com/hecc-blot/core) | `IContainer`（IOC 容器）与 `ILog`（日志契约）的来源 |
| [trace](https://github.com/hecc-blot/trace) | `NewHttpMiddleware` 提供请求级 span 与 traceId |
| [metrics](https://github.com/hecc-blot/metrics) | `NewHttpMiddleware` 提供 QPS/延迟/错误率指标 |
| [sse](https://github.com/hecc-blot/sse) | 复用 `IApiHandle` 与 `IMiddleware` 做流式推送 |
| [ratelimit](https://github.com/hecc-blot/ratelimit) | 限流中间件的典型实现 |
| [db](https://github.com/hecc-blot/db) / [cache](https://github.com/hecc-blot/cache) | 被注入到 API 结构体的数据层 |
| [log-sls](https://github.com/hecc-blot/log-sls) | `ILog` 的另一种实现 |

## 许可证

MIT License

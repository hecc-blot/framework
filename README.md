# hecc-blot-framework

Hecc-Blot 框架内核：接口契约 / IOC 容器 / HTTP 内核（路由 + 中间件 + 参数校验 + 统一响应）/ 统一错误 / 本地日志 / 分页工具。它是所有业务方与其它模块的公共依赖层。

## 安装

```bash
go get github.com/hecc-blot/framework
```

## 职责

| 包 | 说明 |
|----|------|
| `contract/` | 接口契约：`ILog`、`IResponse`、`IApi`、`IApiHandle`、`IMiddleware`、`IContainer`、`IError` 等 |
| `service/ioc` | IOC 容器：`Set` / `SetWithName` / `Inject` 反射注入 |
| `service/http` | HTTP 内核：路由注册、参数自动校验、返回值统一包装 |
| `service/error` | 统一错误构造：`NewError` / `New` / `NewErrorf` / `Newf` |
| `service/log` | 本地日志：Zap + lumberjack 文件滚动 |
| `enum/response` | 响应码与中文映射 |
| `entity/api` | API 实体（`Messages` 校验消息映射等） |
| `util/` | 工具：分页 `Paginator` / `Cursor`、错误消息 `GetErrorMsg` 等 |

---

## IOC 容器

IOC（控制反转）负责管理所有服务的生命周期与依赖注入：初始化阶段 `Set` 注册实例，运行阶段通过 `inject` tag 自动注入。

```go
type Container struct {
    values map[reflect.Type]map[string]reflect.Value
}

func New() *Container
```

- **外层 Map**：Key 为接口类型（`reflect.Type`）
- **内层 Map**：Key 为实例名称（区分同接口多个实现），Value 为实例的反射值

### 注册

```go
container := ioc.New()

// 注册默认实例
container.Set(new(logContract.ILog), logSvc)
container.Set(new(dbContract.IDbFactory), dbFactory)

// 同一接口多个实现时用命名注册
container.SetWithName(new(logContract.ILog), "local", localLog)
container.SetWithName(new(logContract.ILog), "remote", remoteLog)
```

### 注入

```go
type AddApi struct {
    // inject tag 标记需要注入的字段（必须放在请求参数前面）
    DbFactory dbContract.IDbFactory `inject:""`
    LogSvc    logContract.ILog      `inject:""`

    // 命名注入
    // RemoteLog logContract.ILog `inject:"remote"`

    AddRequest // 请求参数放在最后
}
```

### 并发约定（最高级约束）

> ⚠️ **最高级约定**：本约定是整个 IOC 容器使用中优先级最高的约束，违反会导致数据竞争。

`Container` 内部 `values` map **不加锁**，依赖以下约定保证并发安全：

- `Set` / `SetWithName` **仅允许在启动初始化阶段调用**（单线程，先注册后启动）；
- 初始化完成后容器进入**只读**，此时 `Get` / `Inject` 可安全并发调用；
- **运行时禁止再 `Set`**。

### 注入规则

1. **字段顺序**：注入字段必须放在结构体最前面，请求参数放在最后（遇到无 `inject` tag 的字段即停止注入）。
2. **匿名嵌套**：支持匿名嵌套结构体的注入。
3. **指针类型**：注入字段可以是指针类型。

---

## 路由与中间件

HTTP 内核基于 Gin，提供路由注册、中间件链、参数自动校验与返回值统一包装。

### 接口定义

```go
type IApiHandle interface {
    Get(apiPath string, api interface{})
    Post(apiPath string, api interface{})
    Middleware(middlewares ...IMiddleware) IApiHandle
    Group(relativePath string, middlewares ...IMiddleware) IApiHandle
    Listen(clearUps ...func())
    Engine() *gin.Engine
}

type IApi interface {
    Call(ctx *gin.Context) (interface{}, error.IError)
}

type IMiddleware interface {
    Middleware() interface{}
}
```

### 创建处理器并注册路由

```go
import (
    iCoreApi "github.com/hecc-blot/framework/contract/api"
    httpSvc "github.com/hecc-blot/framework/service/http"
    ioc "github.com/hecc-blot/framework/service/ioc"
)

responseSvc := httpSvc.NewResponseSvc()
apiHandle := httpSvc.NewApiSvc(&config.Server, responseSvc, container)

func register(apiHandle iCoreApi.IApiHandle) {
    apiHandle.Middleware(&TokenMiddleware{})

    apiHandle.Post("account/add", &AddApi{})
    apiHandle.Get("account/list", &ListApi{})

    // 分组中间件
    apiGroup := apiHandle.Group("admin", &TokenMiddleware{})
    apiGroup.Get("account/page", &PageListApi{})
}
```

### 定义中间件

```go
type TokenMiddleware struct {
    ResponseSvc iCoreApi.IResponse `inject:""`
}

func (t TokenMiddleware) Middleware() interface{} {
    return func(c *gin.Context) {
        if c.GetHeader("Authorization") == "" {
            t.ResponseSvc.Regular(c, nil, errorSvc.NewError(response.TokenInvalid, errors.New("token 为空")))
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 请求处理流程

```
请求 → [中间件链] → 参数绑定+校验 → Call() 业务逻辑 → 响应包装 → 统一响应
```

每个请求会创建独立 API 实例（避免并发共享写入），再自动 `ShouldBind` 绑定并校验参数，最后调用 `Call()` 并包装返回值。

框架在创建 API 处理器时自动注册内置中间件：

| 中间件 | 功能 |
|--------|------|
| `gin.Recovery()` | 捕获 handler panic，返回 500 而非进程崩溃 |
| `bodySizeLimit` | 限制请求体大小，防止大 payload 攻击，默认 10MB |

> 链路追踪中间件不自动注册，由组装层显式 `trace.NewHttpMiddleware(traceSvc)` 注册（见 [trace 模块](https://github.com/hecc-blot/trace)）。
> 请求限流由 [ratelimit 模块](https://github.com/hecc-blot/ratelimit) 提供，中间件由业务方实现并注册。

---

## 参数校验

基于 `go-playground/validator`，在路由注册时自动绑定请求参数并校验，校验失败返回统一格式错误响应。

```go
type AddRequest struct {
    Name     string `json:"name" binding:"required"`
    Age      int    `json:"age" binding:"required,min=1,max=150"`
    Email    string `json:"email" binding:"email"`
    Password string `json:"password" binding:"required,min=6"`
}
```

常用 tag：

| Tag | 说明 | 示例 |
|-----|------|------|
| `required` | 必填 | `binding:"required"` |
| `min` / `max` | 数值或字符串长度 | `binding:"min=1,max=150"` |
| `email` | 邮箱格式 | `binding:"email"` |
| `url` | URL 格式 | `binding:"url"` |
| `len` | 精确长度 | `binding:"len=11"` |
| `eqfield` | 等于另一个字段 | `binding:"eqfield=Password"` |
| `gt` / `gte` / `lt` / `lte` | 比较 | `binding:"gt=0"` |
| `oneof` | 枚举值 | `binding:"oneof=male female"` |

### 自定义错误信息

实现 `IValidator` 接口，返回 `字段.规则` 对应的中文提示：

```go
// framework/contract/api/validator.go
type IValidator interface {
    GetMessages() entityApi.Messages
}

// framework/entity/api/validator.go
type Messages map[string]string
```

```go
func (a AddRequest) GetMessages() entityApi.Messages {
    return entityApi.Messages{
        "Name.required":     "用户名不能为空",
        "Age.required":      "年龄不能为空",
        "Age.min":           "年龄最小为1",
        "Password.required": "密码不能为空",
        "Password.min":      "密码长度至少6位",
    }
}
```

### 错误消息获取

`util.GetErrorMsg()` 按三级优先级获取错误消息：

1. **自定义消息** — 结构体实现了 `IValidator` 且 `GetMessages()` 有对应 key
2. **validator 默认** — validator 内置英文消息
3. **原始 error** — 非 validator 错误（空 body、JSON 格式错误等）

---

## 统一错误与响应

框架自动将 API 返回值包装为 `{code, message, data}` 统一格式。

```json
{
    "code": 10000,
    "message": "请求成功",
    "data": {}
}
```

### 错误接口

```go
// framework/contract/error/error.go
type IError interface {
    error
    GetCode() response.Value
    GetData() interface{}
}
```

```go
import errorSvc "github.com/hecc-blot/framework/service/error"

err := errorSvc.NewError(response.Fail, errors.New("数据库错误"))
err = errorSvc.New(response.Fail, "用户名已存在")
err = errorSvc.NewErrorf(response.Fail, "查询用户 %d 失败", userID)
err = errorSvc.Newf(response.ValidateError, "字段 %s 不能为空", "name")
```

### 响应码一览

定义在 `framework/enum/response/index.go`：

| 常量 | 值 | 说明 |
|------|------|------|
| `Success` | 10000 | 成功 |
| `Processing` | 10001 | 处理中 |
| `Fail` | 40000 | 失败 |
| `Busy` | 40001 | 业务繁忙 |
| `ValidateError` | 40002 | 参数验证失败 |
| `TokenInvalid` | 40003 | 无效 token |
| `AccessDenied` | 40004 | 禁止访问 |
| `NoDataPermission` | 40005 | 无数据处理权限 |
| `Illegal` | 50000 | 非法请求 |
| `Panic` | 50001 | 服务器内部错误 |

---

## 分页组件

位于 `framework/util/paginator.go`，提供两种分页模式。

### Offset/Limit 分页

```go
type PageOpts struct {
    Page     int
    PageSize int
}

type Paginator[T any] struct {
    List     []T   `json:"list"`
    Page     int   `json:"page"`
    PageSize int   `json:"pageSize"`
    Total    int64 `json:"total"`
}
```

```go
total, _ := db.Count()
var list []AccountModel
offset := (opts.Page - 1) * opts.PageSize
db.Order("id desc").Limit(opts.PageSize).Offset(offset).Find(&list)

return util.NewPage(list, total, opts), nil
```

- `Page` 不传默认 1，`PageSize` 不传默认 10
- `list` 为空时返回 `[]` 而非 `null`

### 游标分页

```go
type CursorOpts struct {
    Cursor   any
    PageSize int
}

type Cursor[T any] struct {
    List       []T  `json:"list"`
    NextCursor any  `json:"nextCursor"`
    HasMore    bool `json:"hasMore"`
    PageSize   int  `json:"pageSize"`
}
```

核心约定：**查询时多取一条（`pageSize + 1`）**，`NewCursor` 自动判断 `hasMore` 并截断：

```go
var list []AccountModel
db.Where("id > ?", cursor).Order("id asc").Limit(pageSize + 1).Find(&list)

return util.NewCursor(list, pageSize, func(item *AccountModel) any {
    return item.ID
}), nil
```

| | Offset/Limit | 游标 |
|--|-------------|------|
| 入参 | `page` + `pageSize` | `cursor` + `pageSize` |
| 深翻页性能 | 递减 | 稳定（走索引） |
| 总页数/跳页 | 支持 | 不支持 |
| 适用场景 | 管理后台、报表 | 信息流、无限滚动 |

---

## 本地日志（已下沉至 core）

本地日志（Zap + lumberjack 文件滚动）已随 `ILog` 契约一并下沉至 [core 模块](https://github.com/hecc-blot/core)，framework 不再内置日志。使用方式：

```go
import (
    logContract "github.com/hecc-blot/core/contract/log"
    log "github.com/hecc-blot/core/service/log"
)

logSvc, err := log.NewLogger(&config.Log.Local)
if err != nil {
    panic(err)
}
container.Set(new(logContract.ILog), logSvc)
```

> 日志配置类型见 `core/config/log`（`LocalConfig`/`SlsConfig`），配置项、级别文件、TraceId 自动关联等详见 [core README](https://github.com/hecc-blot/core)。SLS 后端由 [log-sls 模块](https://github.com/hecc-blot/log-sls) 提供，二者通过 `ILog` 契约二选一。

---

## 配置

框架自身的配置为 `server`（HTTP 内核），类型见 `framework/config/http`。

### server

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `port` | string | 是 | 服务监听端口 |
| `env` | string | 是 | 运行环境：`dev` / `test` / `product` |
| `read_timeout` | int | 否 | 读取超时（秒），0 默认 30 |
| `write_timeout` | int | 否 | 写入超时（秒），0 默认 30 |
| `idle_timeout` | int | 否 | 空闲超时（秒），0 默认 60 |
| `body_size_limit` | int | 否 | 请求体最大字节数，0 默认 10MB |

`env` 映射到 Gin 模式：`dev`→DebugMode、`test`→TestMode、`product`→ReleaseMode。

---

## 组件替换

框架采用面向接口编程，替换组件只需实现对应接口并注册到 IOC。

```go
// 1. 实现 ILog 接口
type LogrusLogSvc struct{ logger *logrus.Logger }

func (l LogrusLogSvc) Debug(ctx context.Context, msg string, fields ...interface{}) { l.logger.Debug(msg, fields...) }
func (l LogrusLogSvc) Info(ctx context.Context, msg string, fields ...interface{})  { l.logger.Info(msg, fields...) }
func (l LogrusLogSvc) Warn(ctx context.Context, msg string, fields ...interface{})  { l.logger.Warn(msg, fields...) }
func (l LogrusLogSvc) Error(ctx context.Context, msg string, fields ...interface{}) { l.logger.Error(msg, fields...) }

// 2. 注册到 IOC 容器，覆盖默认实现
container.Set(new(logContract.ILog), &LogrusLogSvc{logger: logrus.New()})
```

同样的方式可替换数据库、缓存等任意组件。替换要点：

- 必须实现接口的**所有方法**，签名完全匹配
- 需要资源清理的组件应返回清理函数，`defer` 释放
- 新增配置字段需更新对应 `config` 结构体

---

## 相关模块

| 模块 | 职责 | 仓库 |
|------|------|------|
| [db](https://github.com/hecc-blot/db) | 数据库（GORM MySQL/PostgreSQL） |
| [cache](https://github.com/hecc-blot/cache) | 缓存（本地 + Redis） |
| [trace](https://github.com/hecc-blot/trace) | 链路追踪（OpenTelemetry） |
| [sse](https://github.com/hecc-blot/sse) | SSE 推送 |
| [ratelimit](https://github.com/hecc-blot/ratelimit) | 请求限流 |
| [log-sls](https://github.com/hecc-blot/log-sls) | 阿里云 SLS 日志 |

完整组装示例与快速开始见 [guide](https://github.com/hecc-blot/guide)。

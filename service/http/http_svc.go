package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/hecc-blot/core/contract/ioc"
	serverConfig "github.com/hecc-blot/framework/config/http"
	iCoreApi "github.com/hecc-blot/framework/contract/api"
	envEnum "github.com/hecc-blot/framework/enum/env"
	"github.com/hecc-blot/framework/enum/response"
	coreError "github.com/hecc-blot/framework/service/error"
	"github.com/hecc-blot/framework/util"

	"github.com/gin-gonic/gin"
)

// 停机打印是「进程级事件」，夹在成片的请求日志里很难被发现，上色便于一眼定位：
// 黄色=正在停机，绿色=已退出，红色=停机超时/失败。只对终端生效，见 terminalColor。
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiGreen  = "\033[32m"
)

// terminalColor 给终端输出套颜色；
// 非终端（管道、重定向到文件、容器日志采集）时原样返回 —— 那些场景下转义码只会变成乱码。
// 用 os.ModeCharDevice 判断而不再引第三方终端库： 它正是「stdout 是终端」在标准库里的可移植表达。
// 同时遵守 NO_COLOR 与 TERM=dumb 约定。
func terminalColor(color, msg string) string {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return msg
	}
	info, err := os.Stdout.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return msg
	}
	return color + msg + ansiReset
}

type ApiHandle struct {
	config    *serverConfig.Config
	engine    *gin.Engine
	group     *gin.RouterGroup
	response  *responseHolder
	container ioc.IContainer
	hooks     *shutdownHooks
}

// responseHolder 承载响应渲染实现（IResponse），由 SetResponseSvc 注入。
//
// 用**指针**在 Group 出来的实例间共享，理由与 hooks 相同：值拷贝会让「先建分组、 后在根上注入」的分组永远拿不到实现，注册 API 路由时就会 panic。
// 加锁是因为注入发生在启动期、读取发生在每个请求上 —— 与 hooks 一样，
// 宁可付一把无竞争的读写锁，也不留一个数据竞争。
type responseHolder struct {
	mu  sync.RWMutex
	svc iCoreApi.IResponse
}

func (h *responseHolder) set(svc iCoreApi.IResponse) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.svc = svc
}

func (h *responseHolder) get() iCoreApi.IResponse {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.svc
}

// shutdownHooks 关闭钩子集合。
//
// 用**指针**在 Group 出来的实例间共享：根实例才是 Listen 的那个，分组上注册的钩子
// 必须也能到达它，否则在分组上注册的钩子会被静默丢弃。
// 加锁是因为注册可能来自任意 goroutine，而读取发生在停机时 —— 竞态窗口极小，
// 但代价只在启动期与停机各一次，不值得为省这点开销留一个数据竞争。
type shutdownHooks struct {
	mu  sync.Mutex
	fns []func()
}

func (h *shutdownHooks) add(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fns = append(h.fns, fn)
}

// all 返回快照，避免执行期间钩子再注册导致遍历与写入并发。
func (h *shutdownHooks) all() []func() {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.fns)
}

// runHooks 顺序执行钩子，每个独立 recover。
//
// 必须逐个包 recover：一个钩子 panic 若逃逸出去，后续钩子与 srv.Shutdown 都不会执行 ——
// 停机流程被一个坏钩子整体带崩，代价远大于"少执行一个钩子"。
func runHooks(hooks []func()) {
	for _, fn := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("shutdown hook panic: %v\n%s\n", r, debug.Stack())
				}
			}()
			fn()
		}()
	}
}

var mapEnv = map[envEnum.Value]string{
	envEnum.DevMode:     gin.DebugMode,
	envEnum.ProductMode: gin.ReleaseMode,
	envEnum.TestMode:    gin.TestMode,
}

func (f *ApiHandle) Engine() *gin.Engine {
	return f.engine
}

func (f *ApiHandle) Middleware(middlewares ...iCoreApi.IMiddleware) iCoreApi.IApiHandle {
	for _, iMiddleware := range middlewares {
		f.container.Inject(iMiddleware)
		f.group.Use(AdaptMiddleware(iMiddleware.Middleware()))
	}

	return f
}

// Group 创建路由分组，分组内的中间件仅作用于该分组。
func (f *ApiHandle) Group(relativePath string, middlewares ...iCoreApi.IMiddleware) iCoreApi.IApiHandle {
	group := &ApiHandle{
		config:    f.config,
		engine:    f.engine,
		group:     f.group.Group(relativePath),
		response:  f.response, // 共享同一份响应实现：分组上注册的 API 路由也要能渲染响应
		container: f.container,
		hooks:     f.hooks, // 共享同一份钩子：分组上注册的钩子也要能在根实例停机时执行
	}
	group.Middleware(middlewares...)
	return group
}

// resolveTimeouts 把配置里的秒数解析成实际生效的超时值（单位：秒），
// 单独抽出来是为了能测 —— "默认值 / 显式关闭"这类语义最容易被后续改动碰坏。
//
// 语义：
//   - read_timeout / idle_timeout：<= 0（含未配置）用默认值 30s / 60s；
//   - write_timeout：**负值 = 不设写超时**（约定写 -1），0 / 未配置 = 默认 30s。
//
// 为什么 write_timeout 不用 0 表示关闭：config 的零值就是 0，分不清"没配"与"配了 0"，
// 用 0 当开关等于顺手拿掉慢客户端的兜底。负值在配置里不会自然出现，拿它当显式开关最安全。
//
// 需要"不设写超时"的场景是长耗时写入 —— 注意 SSE / WS 的长连接**不需要**靠它：
// 它们的 handler 会自己清掉 net/http 设下的绝对写 deadline（见 sse/service/sse_svc.go）。
func resolveTimeouts(cfg *serverConfig.Config) (read, write, idle int) {
	read, write, idle = cfg.ReadTimeout, cfg.WriteTimeout, cfg.IdleTimeout

	if read <= 0 {
		read = 30
	}
	if write < 0 {
		write = 0 // 不设写超时
	} else if write == 0 {
		write = 30
	}
	if idle <= 0 {
		idle = 60
	}

	return read, write, idle
}

// OnShutdown 注册关闭钩子，见 iCoreApi.IApiHandle 的约定（幂等、独立 recover、构造时注册一次）。
func (f *ApiHandle) OnShutdown(fn func()) {
	// 忽略 nil：注册一个空钩子没有意义，而 nil 函数在停机时调用会 panic。
	if fn == nil {
		return
	}
	f.hooks.add(fn)
}

func (f *ApiHandle) Get(apiPath string, apiInstance any) {
	f.registerAPI(apiPath, apiInstance, http.MethodGet)
}

func (f *ApiHandle) Post(apiPath string, apiInstance any) {
	f.registerAPI(apiPath, apiInstance, http.MethodPost)
}

// Handle 注册原生 http.Handler（如 Prometheus 采集端点、SSE 流），
// 不经过 IApi 的参数绑定与响应包装，仅复用框架的中间件/分组。
func (f *ApiHandle) Handle(method, path string, handler http.Handler) {
	f.group.Handle(method, path, gin.WrapH(handler))
}

func (f *ApiHandle) Listen() {
	readTimeout, writeTimeout, idleTimeout := resolveTimeouts(f.config)
	srv := &http.Server{
		Addr:         ":" + f.config.Port,
		Handler:      f.engine,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
		IdleTimeout:  time.Duration(idleTimeout) * time.Second,
	}

	// 先同步绑定端口，失败（端口占用、地址非法）直接在调用方协程 panic。
	// 不用 srv.ListenAndServe 正是因为它把 bind 和 serve 绑在一起、只能整体丢进子协程，错误会被吞掉：
	// 主协程继续等信号，进程活着但端口不通。
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(fmt.Sprintf("listen %s: %s", srv.Addr, err))
	}

	// 接收 SIGINT（Ctrl + C）和 SIGTERM 信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	// Serve 同步阻塞，信号只能放子协程等。
	shutdownDone := make(chan struct{})
	go func() {
		<-quit
		fmt.Println(terminalColor(ansiYellow, "Shutting down server..."))

		// 优雅关闭：先执行关闭钩子（如通知 SSE / WS 客户端优雅断开），再关闭 HTTP 服务。
		// 位置承重 —— srv.Shutdown 只会等活跃连接变 idle，长连接不会自己变 idle；
		// 先让钩子发关闭帧、取消连接，handler 才能返回，Shutdown 才能迅速收尾。
		runHooks(f.hooks.all())

		// 创建一个5秒的超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// 等待所有请求完成处理后关闭服务
		if err := srv.Shutdown(ctx); err != nil {
			fmt.Println(terminalColor(ansiRed, fmt.Sprintf("Server forced to shutdown: %v", err)))
		}
		close(shutdownDone)
	}()

	// 同步 Serve：监听期间的错误直接返回给调用方，不再只 printf 一行。
	// ErrServerClosed 只在 Shutdown 之后返回，属正常退出。
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(fmt.Sprintf("serve: %s", err))
	}

	// Shutdown 关掉 listener 后 Serve 立即返回，但此时活跃连接可能尚未清空；
	// 等它收尾，保持「Listen 返回 = 停机已完成」的语义。
	<-shutdownDone

	fmt.Println(terminalColor(ansiGreen, "Server exiting"))
}

func (f *ApiHandle) registerAPI(apiPath string, apiInstance any, method string) {
	// 注入api依赖项（仅注入共享的服务依赖，请求参数字段留空）
	f.container.Inject(apiInstance)

	if _, ok := apiInstance.(iCoreApi.IApi); ok {
		// 没注入响应渲染就不能注册 API 路由。早失败在注册期 ——
		// 否则要等第一个请求进来才 nil 崩，而且崩在 gin 的 handler 里更难定位。
		// 纯原生 handler 的内核（WS / SSE / Prometheus 采集端点）本就不需要它。
		responseSvc := f.response.get()
		if responseSvc == nil {
			panic("api: response svc is required to register Get/Post routes; call SetResponseSvc(httpSvc.NewResponseSvc()) first")
		}

		// 缓存具体类型，避免每个请求都做类型断言
		apiType := reflect.TypeOf(apiInstance).Elem()

		handler := func(c *gin.Context) {
			// 每个请求创建独立实例，避免并发请求共享写入
			newInstance := reflect.New(apiType).Interface()
			f.container.Inject(newInstance)
			api := newInstance.(iCoreApi.IApi)

			// 自动绑定参数，并进行校验
			if err := c.ShouldBind(newInstance); err != nil {
				responseSvc.Regular(NewContext(c), nil, coreError.New(response.ValidateError, util.GetErrorMsg(api, err)))
				return
			}

			ctx := NewContext(c)
			resp, err := api.Call(ctx)
			responseSvc.Regular(ctx, resp, err)
		}

		switch method {
		case http.MethodGet:
			f.group.GET(apiPath, handler)
		case http.MethodPost:
			f.group.POST(apiPath, handler)
		default:
			panic(
				fmt.Sprintf("invalid http method: %s", method),
			)
		}
	}
}

// NewApiSvc 创建 HTTP 内核。
//
// 不再接收响应渲染实现：那只是 API 路由（Get/Post）的需要，用 Handle 挂原生 handler 的
// 内核（纯 WS / SSE 服务、Prometheus 采集端口）不该被迫构造一个用不上的东西。
// 注册 Get/Post 之前请先 SetResponseSvc。
func NewApiSvc(config *serverConfig.Config, container ioc.IContainer) iCoreApi.IApiHandle {
	mode, ok := mapEnv[config.Env]
	if !ok {
		panic(fmt.Sprintf("invalid env config: %s", mode))
	}

	if container == nil {
		panic("ioc: container must not be nil")
	}

	gin.SetMode(mode)
	app := gin.New()
	// 开启 gin 的 Context 回退：gin 默认（ContextWithFallback=false）下 *gin.Context 的
	// Value/Deadline/Done/Err 不会委托给 Request.Context()，于是把 IContext 当
	// context.Context 往下传时取不到「类型化 key」——典型后果是 core/enum/trace 的
	// TraceIdKey 在业务日志与 otel span 里全部丢失（traceId 为空、MQ 消息头无 traceparent）。
	// 开启后这 4 个方法的行为与 Request.Context() 一致。
	app.ContextWithFallback = true
	app.Use(gin.Recovery())
	app.Use(bodySizeLimit(config.BodySizeLimit)) // 10MB

	apiHandle := &ApiHandle{
		config:    config,
		engine:    app,
		group:     &app.RouterGroup,
		response:  &responseHolder{},
		container: container,
		hooks:     &shutdownHooks{},
	}

	return apiHandle
}

// SetResponseSvc 注入 API 响应渲染实现，见 IApiHandle 的说明。
func (f *ApiHandle) SetResponseSvc(responseSvc iCoreApi.IResponse) {
	if responseSvc == nil {
		return
	}
	f.response.set(responseSvc)
}

// bodySizeLimit 限制请求体大小，防止大 payload 攻击
func bodySizeLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes == 0 {
		maxBytes = 10 << 20
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

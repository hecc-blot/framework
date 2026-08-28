package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	serverConfig "github.com/hecc-blot/framework/config/http"
	iCoreApi "github.com/hecc-blot/framework/contract/api"
	"github.com/hecc-blot/framework/contract/ioc"
	envEnum "github.com/hecc-blot/framework/enum/env"
	"github.com/hecc-blot/framework/enum/response"
	coreError "github.com/hecc-blot/framework/service/error"
	"github.com/hecc-blot/framework/util"

	"github.com/gin-gonic/gin"
)

type ApiHandle struct {
	config      *serverConfig.Config
	engine      *gin.Engine
	group       *gin.RouterGroup
	responseSvc iCoreApi.IResponse
	container   ioc.IContainer
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
		config:      f.config,
		engine:      f.engine,
		group:       f.group.Group(relativePath),
		responseSvc: f.responseSvc,
		container:   f.container,
	}
	group.Middleware(middlewares...)
	return group
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

func (f *ApiHandle) Listen(onShutdown ...func()) {
	readTimeout := f.config.ReadTimeout
	writeTimeout := f.config.WriteTimeout
	idleTimeout := f.config.IdleTimeout
	if readTimeout <= 0 {
		readTimeout = 30
	}
	if writeTimeout <= 0 {
		writeTimeout = 30
	}
	if idleTimeout <= 0 {
		idleTimeout = 60
	}
	srv := &http.Server{
		Addr:         ":" + f.config.Port,
		Handler:      f.engine,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
		IdleTimeout:  time.Duration(idleTimeout) * time.Second,
	}

	// 启动服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("listen: %s\n", err)
		}
	}()

	// 等待中断信号来优雅地关闭服务器，为重启做准备
	quit := make(chan os.Signal, 1)
	// 接收SIGINT（Ctrl + C）和SIGTERM信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	// 优雅关闭：先执行关闭钩子（如通知 SSE 客户端优雅断开），再关闭 HTTP 服务
	for _, fn := range onShutdown {
		fn()
	}

	// 创建一个5秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 等待所有请求完成处理后关闭服务
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server exiting")
}

func (f *ApiHandle) registerAPI(apiPath string, apiInstance any, method string) {
	// 注入api依赖项（仅注入共享的服务依赖，请求参数字段留空）
	f.container.Inject(apiInstance)

	if _, ok := apiInstance.(iCoreApi.IApi); ok {
		// 缓存具体类型，避免每个请求都做类型断言
		apiType := reflect.TypeOf(apiInstance).Elem()

		handler := func(c *gin.Context) {
			// 每个请求创建独立实例，避免并发请求共享写入
			newInstance := reflect.New(apiType).Interface()
			f.container.Inject(newInstance)
			api := newInstance.(iCoreApi.IApi)

			// 自动绑定参数，并进行校验
			if err := c.ShouldBind(newInstance); err != nil {
				f.responseSvc.Regular(NewContext(c), nil, coreError.New(response.ValidateError, util.GetErrorMsg(api, err)))
				return
			}

			ctx := NewContext(c)
			resp, err := api.Call(ctx)
			f.responseSvc.Regular(ctx, resp, err)
		}

		switch method {
		case http.MethodGet:
			f.group.GET(apiPath, handler)
		case http.MethodPost:
			f.group.POST(apiPath, handler)
		default:
			panic(
				fmt.Sprintf("无效http请求类型，%s", method),
			)
		}
	}
}

func NewApiSvc(config *serverConfig.Config, responseSvc iCoreApi.IResponse, container ioc.IContainer) iCoreApi.IApiHandle {
	mode, ok := mapEnv[config.Env]
	if !ok {
		panic(fmt.Sprintf("无效环境配置:%s", mode))
	}

	if container == nil {
		panic("ioc: 注入容器不能为空")
	}

	gin.SetMode(mode)
	app := gin.New()
	app.Use(gin.Recovery())
	app.Use(bodySizeLimit(config.BodySizeLimit)) // 10MB

	apiHandle := &ApiHandle{
		config:      config,
		engine:      app,
		group:       &app.RouterGroup,
		responseSvc: responseSvc,
		container:   container,
	}

	return apiHandle
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

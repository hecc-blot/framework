package api

import "github.com/gin-gonic/gin"

type IApiHandle interface {
	Get(apiPath string, api interface{})
	Post(apiPath string, api interface{})
	Middleware(middlewares ...IMiddleware) IApiHandle
	Group(relativePath string, middlewares ...IMiddleware) IApiHandle
	Engine() *gin.Engine
	Listen(onShutdown ...func())
}

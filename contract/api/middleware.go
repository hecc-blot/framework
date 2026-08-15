package api

type IMiddleware interface {
	Middleware() interface{}
}

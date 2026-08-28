package service

import (
	"net/http"
	"sync"

	"github.com/hecc-blot/framework/contract/api"
	coreError "github.com/hecc-blot/framework/contract/error"
	"github.com/hecc-blot/framework/enum/response"
)

type ResponseSvc struct {
	Code    response.Value `json:"code"`
	Message string         `json:"message"`
	Data    any            `json:"data"`
}

var responsePool = sync.Pool{
	New: func() any { return &ResponseSvc{} },
}

func (r *ResponseSvc) Regular(ctx api.IContext, data any, err coreError.IError) {
	code := response.Success
	if err != nil {
		code = err.GetCode()
		data = err.GetData()
	}

	resp := responsePool.Get().(*ResponseSvc)
	defer responsePool.Put(resp)

	resp.Code = code
	resp.Message = response.CodeMap[code]
	resp.Data = data

	ctx.JSON(http.StatusOK, resp)
}

func NewResponseSvc() api.IResponse {
	return &ResponseSvc{}
}

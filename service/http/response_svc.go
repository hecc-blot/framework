package service

import (
	"net/http"
	"sync"

	"github.com/hecc-blot/core/enum/errorcode"
	"github.com/hecc-blot/framework/contract/api"
	coreError "github.com/hecc-blot/framework/contract/error"
	"github.com/hecc-blot/framework/enum/response"
	errorSvc "github.com/hecc-blot/framework/service/error"
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
		// 对外码在边界这一处决定：注册过映射的 core 码按映射翻译，未注册的原码透传。
		code, _ = errorSvc.Resolve(err.GetCode())
		data = err.GetData()
	}

	resp := responsePool.Get().(*ResponseSvc)
	defer responsePool.Put(resp)

	resp.Code = code
	resp.Message = message(code)
	resp.Data = data

	ctx.JSON(http.StatusOK, resp)
}

// message 取对外文案，按三跳回退：
// 协议码文案（CodeMap）→ core 段文案（errorcode.Message）→ Fail 的兜底文案。
//
// 前两跳覆盖了所有有登记的码；最后一跳兜住业务自定义码（60000+，框架无从知道其文案）——
// 宁可与"请求失败"同文案，也不发出空 message，更不能把内部错误文本（含调试格式）外泄。
func message(code response.Value) string {
	if msg, ok := response.CodeMap[code]; ok && msg != "" {
		return msg
	}
	if msg := errorcode.Message(code); msg != "" {
		return msg
	}
	return response.CodeMap[response.Fail]
}

func NewResponseSvc() api.IResponse {
	return &ResponseSvc{}
}

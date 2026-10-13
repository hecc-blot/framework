package service

import (
	"errors"
	"fmt"

	"github.com/hecc-blot/core/enum/errorcode"
	coreError "github.com/hecc-blot/framework/contract/error"
	"github.com/hecc-blot/framework/enum/response"
)

type errorSvc struct {
	error

	code response.Value
	data interface{}
}

func (m errorSvc) Error() string {
	return fmt.Sprintf("[err: %v, code: %v, data: %v]", m.error, m.code, m.data)
}

func (m errorSvc) GetCode() response.Value {
	return m.code
}

func (m errorSvc) GetData() interface{} {
	return m.data
}

func New(code response.Value, data interface{}) coreError.IError {
	return errorSvc{
		error: fmt.Errorf("%v", data),
		code:  code,
		data:  data,
	}
}

func Newf(code response.Value, format string, args ...interface{}) coreError.IError {
	return errorSvc{
		error: fmt.Errorf(format, args...),
		code:  code,
		data:  fmt.Sprintf(format, args...),
	}
}

func NewError(code response.Value, err error) coreError.IError {
	return errorSvc{
		error: err,
		code:  code,
	}
}

func NewErrorf(code response.Value, format string, args ...interface{}) coreError.IError {
	return errorSvc{
		error: fmt.Errorf(format, args...),
		code:  code,
	}
}

// From 把 err 上携带的 core 码解析成 IError —— 业务不必再"回忆这个错该返回哪个码"。
//
// 解析顺序：
//  1. err 为 nil → 返回 nil（便于 `return nil, errorSvc.From(err)` 这样直写）；
//  2. err 链上已有 IError → 原样返回（From 因此是幂等的）；
//  3. err 链上有 core 码（errorcode.From，可穿透 fmt.Errorf 的 %w）→ 用该码。
//     注意这里**不做翻译**：对客户端最终发哪个码由渲染层统一按映射表决定
//     （见 code_map.go 的 Resolve），此处只记录"这个错在语义上是什么"；
//  4. 都没有 → response.Fail。
func From(err error) coreError.IError {
	if err == nil {
		return nil
	}
	if ie, ok := errors.AsType[coreError.IError](err); ok {
		return ie
	}

	code := response.Fail
	if c, ok := errorcode.From(err); ok {
		code = c
	}

	return errorSvc{
		error: err,
		code:  code,
	}
}

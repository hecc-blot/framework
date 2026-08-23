package service

import (
	"fmt"

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

package error

import "github.com/hecc-blot/core/enum/response"

type IError interface {
	error

	GetCode() response.Value
	GetData() interface{}
}

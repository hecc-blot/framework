package error

import "github.com/hecc-blot/framework/enum/response"

type IError interface {
	error

	GetCode() response.Value
	GetData() interface{}
}

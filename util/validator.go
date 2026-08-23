package util

import (
	"errors"

	"github.com/hecc-blot/framework/contract/api"

	"github.com/go-playground/validator/v10"
)

// GetErrorMsg 获取错误信息
// 优先匹配自定义校验消息，其次返回 validator 默认消息，最后兜底返回原始 error
func GetErrorMsg(request interface{}, err error) string {
	validationErrors, ok := errors.AsType[validator.ValidationErrors](err)
	if ok && len(validationErrors) > 0 {
		_, isValidator := request.(api.IValidator)

		for _, v := range validationErrors {
			if isValidator {
				if message, exist := request.(api.IValidator).GetMessages()[v.Field()+"."+v.Tag()]; exist {
					return message
				}
			}
			return v.Error()
		}
	}

	// 非 validator 错误（如空 body 导致的 JSON EOF、格式错误等），返回原始错误信息
	return err.Error()
}

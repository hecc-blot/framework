package util

import (
	"errors"
	"testing"

	entityApi "github.com/hecc-blot/core/entity/api"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// mockValidator 实现 IValidator，用于返回自定义校验消息。
type mockValidator struct {
	messages entityApi.Messages
}

func (m mockValidator) GetMessages() entityApi.Messages {
	return m.messages
}

// form 被校验的结构体，用于生成 validator.ValidationErrors。
type form struct {
	Name string `validate:"required"`
	Age  int    `validate:"min=1"`
}

func TestGetErrorMsg(t *testing.T) {
	validate := validator.New()

	t.Run("自定义消息优先", func(t *testing.T) {
		err := validate.Struct(form{})
		req := mockValidator{messages: entityApi.Messages{"Name.required": "名字不能为空"}}
		assert.Equal(t, "名字不能为空", GetErrorMsg(req, err))
	})

	t.Run("无自定义消息返回默认消息", func(t *testing.T) {
		err := validate.Struct(form{})
		req := mockValidator{}
		assert.NotEmpty(t, GetErrorMsg(req, err))
	})

	t.Run("非 validator 错误返回原始信息", func(t *testing.T) {
		req := mockValidator{}
		assert.Equal(t, "plain error", GetErrorMsg(req, errors.New("plain error")))
	})
}

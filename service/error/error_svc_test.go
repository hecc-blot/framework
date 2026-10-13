package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hecc-blot/core/enum/errorcode"
	"github.com/hecc-blot/framework/enum/response"
)

func TestNew(t *testing.T) {
	err := New(response.Fail, "用户名已存在")
	assert.Equal(t, response.Fail, err.GetCode())
	assert.Equal(t, "用户名已存在", err.GetData())
}

func TestNewf(t *testing.T) {
	err := Newf(response.Fail, "查询用户 %d 失败", 123)
	assert.Equal(t, response.Fail, err.GetCode())
	assert.Equal(t, "查询用户 123 失败", err.GetData())
}

func TestNewError(t *testing.T) {
	err := NewError(response.Fail, errors.New("数据库错误"))
	assert.Equal(t, response.Fail, err.GetCode())
	assert.Nil(t, err.GetData())
	assert.Contains(t, err.Error(), "数据库错误")
}

// TestFrom：把 err 上的 core 码解析成 IError 的四条路径 ——
// nil、普通错误兜底、core 码（含 %w 穿透）、已是 IError 时幂等。
func TestFrom(t *testing.T) {
	t.Run("nil 返回 nil", func(t *testing.T) {
		assert.Nil(t, From(nil))
	})

	t.Run("无码的错误落 Fail", func(t *testing.T) {
		err := From(errors.New("db error"))
		assert.Equal(t, response.Fail, err.GetCode())
		assert.Nil(t, err.GetData())
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("取到 core 码并穿透 %w", func(t *testing.T) {
		sentinel := errorcode.New(errorcode.CodeTimeout, "lock: acquire timeout")
		assert.Equal(t, errorcode.CodeTimeout, From(sentinel).GetCode())

		wrapped := fmt.Errorf("handle order: %w", sentinel)
		assert.Equal(t, errorcode.CodeTimeout, From(wrapped).GetCode())
	})

	t.Run("已是 IError 时原样返回", func(t *testing.T) {
		origin := New(response.ValidateError, "bad params")
		assert.Equal(t, origin, From(origin))
	})
}

func TestNewErrorf(t *testing.T) {
	err := NewErrorf(response.ValidateError, "字段 %s 不能为空", "name")
	assert.Equal(t, response.ValidateError, err.GetCode())
	assert.Nil(t, err.GetData())
	assert.Contains(t, err.Error(), "字段 name 不能为空")
}

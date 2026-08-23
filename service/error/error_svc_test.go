package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

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

func TestNewErrorf(t *testing.T) {
	err := NewErrorf(response.ValidateError, "字段 %s 不能为空", "name")
	assert.Equal(t, response.ValidateError, err.GetCode())
	assert.Nil(t, err.GetData())
	assert.Contains(t, err.Error(), "字段 name 不能为空")
}

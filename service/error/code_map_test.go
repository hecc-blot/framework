package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hecc-blot/core/enum/errorcode"
	"github.com/hecc-blot/framework/enum/response"
)

// resetCodeMap 清空映射表，避免用例之间互相污染（同包可直接操作内部状态）。
func resetCodeMap(t *testing.T) {
	t.Helper()
	codeMapMu.Lock()
	codeMap = map[errorcode.Code]response.Value{}
	codeMapMu.Unlock()
}

// TestResolveDefaultPassthrough：表空（默认）时 core 码原样返回 ——
// 这是「默认直用」的核心保证，任何翻译都不能在未注册时发生。
func TestResolveDefaultPassthrough(t *testing.T) {
	resetCodeMap(t)

	got, ok := Resolve(errorcode.CodeTimeout)
	assert.False(t, ok)
	assert.Equal(t, errorcode.CodeTimeout, got)
}

// TestMapCodeThenResolve：注册后按映射翻译，且只影响被注册的那一个码。
func TestMapCodeThenResolve(t *testing.T) {
	resetCodeMap(t)
	t.Cleanup(func() { resetCodeMap(t) })

	MapCode(errorcode.CodeServerBusy, response.Busy)

	mapped, ok := Resolve(errorcode.CodeServerBusy)
	assert.True(t, ok)
	assert.Equal(t, response.Busy, mapped)

	// 逐码生效：没注册的码仍透传
	untouched, ok := Resolve(errorcode.CodeTimeout)
	assert.False(t, ok)
	assert.Equal(t, errorcode.CodeTimeout, untouched)
}

// TestMapCodeOverwrites：重复注册同一码以最后一次为准（启动期配置，不应在运行时改）。
func TestMapCodeOverwrites(t *testing.T) {
	resetCodeMap(t)
	t.Cleanup(func() { resetCodeMap(t) })

	MapCode(errorcode.CodeTimeout, response.Busy)
	MapCode(errorcode.CodeTimeout, response.Illegal)

	got, ok := Resolve(errorcode.CodeTimeout)
	assert.True(t, ok)
	assert.Equal(t, response.Illegal, got)
}

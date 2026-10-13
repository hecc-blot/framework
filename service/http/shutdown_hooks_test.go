package service

import (
	"testing"

	iocSvc "github.com/hecc-blot/core/service/ioc"
	"github.com/stretchr/testify/assert"
)

// TestShutdownHooksRunInOrder：钩子按注册顺序执行。
func TestShutdownHooksRunInOrder(t *testing.T) {
	h := newTestHandle(t, iocSvc.New())

	var got []string
	h.OnShutdown(func() { got = append(got, "first") })
	h.OnShutdown(func() { got = append(got, "second") })

	runHooks(h.hooks.all())

	assert.Equal(t, []string{"first", "second"}, got)
}

// TestOnShutdownIgnoresNil：注册 nil 不 panic、不占位（nil 函数在停机时调用会 panic）。
func TestOnShutdownIgnoresNil(t *testing.T) {
	h := newTestHandle(t, iocSvc.New())

	h.OnShutdown(nil)

	assert.Empty(t, h.hooks.all())
	runHooks(h.hooks.all()) // 不应 panic
}

// TestShutdownHookPanicIsolated：一个钩子 panic 不连累后续钩子 ——
// 修的就是"坏钩子把整个停机流程带崩"这条。
func TestShutdownHookPanicIsolated(t *testing.T) {
	var ran []string

	runHooks([]func(){
		func() { ran = append(ran, "before") },
		func() { panic("hook boom") },
		func() { ran = append(ran, "after") },
	})

	assert.Equal(t, []string{"before", "after"}, ran)
}

// TestGroupSharesShutdownHooks：在 Group 出来的实例上注册的钩子，必须能被根实例看到 ——
// 根实例才是 Listen 的那个，否则分组上注册的钩子会被静默丢弃。
func TestGroupSharesShutdownHooks(t *testing.T) {
	root := newTestHandle(t, iocSvc.New())

	var called bool
	group := root.Group("/v1")
	group.OnShutdown(func() { called = true })

	assert.Len(t, root.hooks.all(), 1, "分组注册的钩子应出现在根实例上")

	runHooks(root.hooks.all())
	assert.True(t, called)
}

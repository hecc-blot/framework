package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {
	t.Run("正常分页", func(t *testing.T) {
		list := []int{1, 2, 3}
		p := NewPage(list, 100, PageOpts{Page: 2, PageSize: 10})
		assert.Equal(t, 2, p.Page)
		assert.Equal(t, 10, p.PageSize)
		assert.Equal(t, int64(100), p.Total)
		assert.Equal(t, list, p.List)
	})

	t.Run("page 默认 1", func(t *testing.T) {
		p := NewPage([]int{1}, 1, PageOpts{Page: 0, PageSize: 10})
		assert.Equal(t, 1, p.Page)
	})

	t.Run("pageSize 默认 10", func(t *testing.T) {
		p := NewPage([]int{1}, 1, PageOpts{Page: 1, PageSize: 0})
		assert.Equal(t, 10, p.PageSize)
	})

	t.Run("nil list 转空 slice", func(t *testing.T) {
		p := NewPage[int](nil, 0, PageOpts{Page: 1, PageSize: 10})
		assert.NotNil(t, p.List)
		assert.Empty(t, p.List)
	})
}

func TestNewCursor(t *testing.T) {
	lastID := func(item *int) any { return *item }

	t.Run("无更多数据", func(t *testing.T) {
		list := []int{1, 2, 3}
		c := NewCursor(list, 10, lastID)
		assert.False(t, c.HasMore)
		assert.Nil(t, c.NextCursor)
		assert.Equal(t, list, c.List)
	})

	t.Run("有更多数据截断", func(t *testing.T) {
		list := []int{1, 2, 3, 4}
		c := NewCursor(list, 3, lastID)
		assert.True(t, c.HasMore)
		assert.Equal(t, 3, c.NextCursor)
		assert.Equal(t, []int{1, 2, 3}, c.List)
	})

	t.Run("pageSize 默认 10", func(t *testing.T) {
		c := NewCursor([]int{1}, 0, lastID)
		assert.Equal(t, 10, c.PageSize)
	})

	t.Run("nil list 转空 slice", func(t *testing.T) {
		c := NewCursor[int](nil, 10, lastID)
		assert.NotNil(t, c.List)
		assert.Empty(t, c.List)
	})
}

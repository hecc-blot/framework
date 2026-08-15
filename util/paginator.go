package util

// ============================ Offset / Limit 分页 ============================

// Paginator offset/limit 分页返回结构
type Paginator[T any] struct {
	List     []T   `json:"list"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

// PageOpts offset 分页入参
type PageOpts struct {
	Page     int
	PageSize int
}

// NewPage 组装 offset/limit 分页返回数据
func NewPage[T any](list []T, total int64, opts PageOpts) *Paginator[T] {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 10
	}
	if list == nil {
		list = make([]T, 0)
	}

	return &Paginator[T]{
		List:     list,
		Page:     opts.Page,
		PageSize: opts.PageSize,
		Total:    total,
	}
}

// ============================ 游标分页 ============================

// Cursor 游标分页返回结构
type Cursor[T any] struct {
	List       []T  `json:"list"`
	NextCursor any  `json:"nextCursor"`
	HasMore    bool `json:"hasMore"`
	PageSize   int  `json:"pageSize"`
}

// CursorOpts 游标分页入参
type CursorOpts struct {
	Cursor   any // 上一页最后一条记录的游标值，首页传 nil
	PageSize int
}

// NewCursor 组装游标分页返回数据
//
// 约定：调用方查询时取 pageSize+1 条，如果返回了多余数据，
// 则用最后一条作为下次游标，并标记 hasMore。
//
//	list:    查询结果（可能包含多余的一条用于判断 hasMore）
//	pageSize: 每页条数
//	last:    提取最后一条记录游标的函数，仅当 len(list) > pageSize 时调用
func NewCursor[T any](list []T, pageSize int, last func(item *T) any) *Cursor[T] {
	if pageSize <= 0 {
		pageSize = 10
	}
	if list == nil {
		list = make([]T, 0)
	}

	hasMore := len(list) > pageSize
	var nextCursor any

	if hasMore {
		nextCursor = last(&list[pageSize-1])
		list = list[:pageSize] // 截掉多查的那条
	}

	return &Cursor[T]{
		List:       list,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		PageSize:   pageSize,
	}
}

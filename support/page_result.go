package support

// PageResult 分页查询结果
type PageResult[T any] struct {
	// 总记录条数
	Total    int64
	// 总页数
	Pages    int64
	// 当前页码
	PageNum  int
	// 每页条数
	PageSize int
	// 记录
	Data     []T
}

func NewPageResult[T any](data []T, total int64, pageNum, pageSize int) PageResult[T] {
	return PageResult[T]{
		Data:     data,
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
	}
}

func EmptyPageResult[T any]() PageResult[T] {
	return PageResult[T]{
		Data:     []T{},
		Total:    0,
		PageNum:  1,
		PageSize: 0,
	}
}

func (r PageResult[T]) IsEmpty() bool {
	return len(r.Data) == 0
}

func (r PageResult[T]) TotalPages() int {
	if r.PageSize <= 0 {
		return 0
	}
	pages := int(r.Total) / r.PageSize
	if int(r.Total)%r.PageSize > 0 {
		pages++
	}
	return pages
}

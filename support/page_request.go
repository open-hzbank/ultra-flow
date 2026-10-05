package support

// PageRequest 分页查询请求
type PageRequest struct {
	// 不传参数默认第一页
	PageNum  int
	// 不传参数默认页容量20
	PageSize int
}

func NewPageRequest(pageNum, pageSize int) PageRequest {
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return PageRequest{PageNum: pageNum, PageSize: pageSize}
}

func (r PageRequest) Offset() int {
	return (r.PageNum - 1) * r.PageSize
}

package goinnodb

import (
	"fmt"
	"io"
)

type PageReader struct {
	r io.ReaderAt
}

func NewPageReader(r io.ReaderAt) *PageReader { return &PageReader{r: r} }

func (pr *PageReader) ReadPage(pageNo uint32) (*InnerPage, error) {
	buf := make([]byte, PageSize)
	off := int64(pageNo) * int64(PageSize)
	if _, err := pr.r.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("read page %d: %w", pageNo, err)
	}
	return NewInnerPage(pageNo, buf)
}

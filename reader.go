// reader.go - Page reader for reading from InnoDB data files
package goinnodb

import (
	"fmt"
	"github.com/wilhasse/go-innodb/format"
	"github.com/wilhasse/go-innodb/page"
	"io"
)

type PageReader struct {
	r io.ReaderAt
}

func NewPageReader(r io.ReaderAt) *PageReader { return &PageReader{r: r} }

func (pr *PageReader) ReadPage(pageNo uint32) (*page.InnerPage, error) {
	buf := make([]byte, format.PageSize)
	off := int64(pageNo) * int64(format.PageSize)
	if _, err := pr.r.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("read page %d: %w", pageNo, err)
	}
	return page.NewInnerPage(pageNo, buf)
}

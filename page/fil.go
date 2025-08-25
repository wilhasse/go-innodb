// fil.go - FIL header and trailer parsing for InnoDB pages
package page

import (
	"fmt"
	"github.com/wilhasse/go-innodb/format"
)

const filNull uint32 = 0xFFFFFFFF

type FilHeader struct {
	Checksum   uint32
	PageNumber uint32
	Prev       *uint32
	Next       *uint32
	LastModLSN uint64
	PageType   format.PageType
	FlushLSN   uint64
	SpaceID    uint32
}

func ParseFilHeader(p []byte) (FilHeader, error) {
	if len(p) < format.PageSize {
		return FilHeader{}, fmt.Errorf("short page: %d", len(p))
	}
	chk, _ := format.Be32(p, 0)
	pg, _ := format.Be32(p, 4)
	prev, _ := format.Be32(p, 8)
	next, _ := format.Be32(p, 12)
	lsn, _ := format.Be64(p, 16)
	pt, _ := format.Be16(p, 24)
	flush, _ := format.Be64(p, 26)
	space, _ := format.Be32(p, 34)
	var prevPtr, nextPtr *uint32
	if prev != filNull {
		prevPtr = &prev
	}
	if next != filNull {
		nextPtr = &next
	}
	return FilHeader{
		Checksum: chk, PageNumber: pg, Prev: prevPtr, Next: nextPtr,
		LastModLSN: lsn, PageType: format.PageType(pt), FlushLSN: flush, SpaceID: space,
	}, nil
}

type FilTrailer struct {
	Checksum uint32
	Low32LSN uint32
}

func ParseFilTrailer(p []byte) (FilTrailer, error) {
	if len(p) < format.FilTrailerSize {
		return FilTrailer{}, fmt.Errorf("short trailer")
	}
	off := format.PageSize - format.FilTrailerSize
	chk, _ := format.Be32(p, off+0)
	lsn, _ := format.Be32(p, off+4)
	return FilTrailer{Checksum: chk, Low32LSN: lsn}, nil
}

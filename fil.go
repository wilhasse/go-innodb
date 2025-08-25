// fil.go - FIL header and trailer parsing for InnoDB pages
package goinnodb

import "fmt"

const filNull uint32 = 0xFFFFFFFF

type FilHeader struct {
	Checksum   uint32
	PageNumber uint32
	Prev       *uint32
	Next       *uint32
	LastModLSN uint64
	PageType   PageType
	FlushLSN   uint64
	SpaceID    uint32
}

func ParseFilHeader(p []byte) (FilHeader, error) {
	if len(p) < PageSize {
		return FilHeader{}, fmt.Errorf("short page: %d", len(p))
	}
	chk, _ := be32(p, 0)
	pg, _ := be32(p, 4)
	prev, _ := be32(p, 8)
	next, _ := be32(p, 12)
	lsn, _ := be64(p, 16)
	pt, _ := be16(p, 24)
	flush, _ := be64(p, 26)
	space, _ := be32(p, 34)
	var prevPtr, nextPtr *uint32
	if prev != filNull {
		prevPtr = &prev
	}
	if next != filNull {
		nextPtr = &next
	}
	return FilHeader{
		Checksum: chk, PageNumber: pg, Prev: prevPtr, Next: nextPtr,
		LastModLSN: lsn, PageType: PageType(pt), FlushLSN: flush, SpaceID: space,
	}, nil
}

type FilTrailer struct {
	Checksum uint32
	Low32LSN uint32
}

func ParseFilTrailer(p []byte) (FilTrailer, error) {
	if len(p) < FilTrailerSize {
		return FilTrailer{}, fmt.Errorf("short trailer")
	}
	off := PageSize - FilTrailerSize
	chk, _ := be32(p, off+0)
	lsn, _ := be32(p, off+4)
	return FilTrailer{Checksum: chk, Low32LSN: lsn}, nil
}

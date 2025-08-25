package goinnodb

import "fmt"

// InnerPage = FIL header + body + FIL trailer (exactly 16 KiB)
type InnerPage struct {
	PageNo  uint32
	FIL     FilHeader
	Trailer FilTrailer
	Data    []byte // full 16KiB page bytes
}

func NewInnerPage(pageNo uint32, page []byte) (*InnerPage, error) {
	if len(page) != PageSize {
		return nil, fmt.Errorf("expected %dB page, got %d", PageSize, len(page))
	}
	h, err := ParseFilHeader(page)
	if err != nil {
		return nil, err
	}
	t, err := ParseFilTrailer(page)
	if err != nil {
		return nil, err
	}
	if uint32(h.LastModLSN&0xffffffff) != t.Low32LSN {
		return nil, fmt.Errorf("low32 LSN mismatch: hdr=%#x trl=%#x", uint32(h.LastModLSN), t.Low32LSN)
	}
	return &InnerPage{PageNo: pageNo, FIL: h, Trailer: t, Data: page}, nil
}

func (ip *InnerPage) PageType() PageType { return ip.FIL.PageType }

package goinnodb

import (
	"bytes"
	"fmt"
)

type IndexPage struct {
	Inner    *InnerPage
	Hdr      IndexHeader
	Fseg     FsegHeader
	Infimum  GenericRecord
	Supremum GenericRecord
	DirSlots []uint16 // dirSlots[0] is the first slot (reversed from end of page)
}

func ParseIndexPage(ip *InnerPage) (*IndexPage, error) {
	if ip.FIL.PageType != PageTypeIndex {
		return nil, fmt.Errorf("not an INDEX page: type=%d", ip.FIL.PageType)
	}
	hdr, err := ParseIndexHeader(ip.Data, FilHeaderSize)
	if err != nil {
		return nil, err
	}
	if hdr.Format != FormatCompact {
		return nil, fmt.Errorf("only compact pages supported (format=%d)", hdr.Format)
	}
	fseg, err := ParseFsegHeader(ip.Data, FilHeaderSize+36)
	if err != nil {
		return nil, err
	}

	cur := FilHeaderSize + PageHeaderSize

	// INFIMUM
	infHdr, err := ParseRecordHeader(ip.Data, cur)
	if err != nil {
		return nil, err
	}
	cur += RecordHeaderSize
	if !bytes.Equal(ip.Data[cur:cur+SystemRecordBytes], LitInfimum) {
		return nil, fmt.Errorf("INFIMUM literal mismatch at %d", cur)
	}
	inf := GenericRecord{PageNumber: ip.PageNo, Header: infHdr, PrimaryKeyPos: cur}
	cur += SystemRecordBytes

	// SUPREMUM
	supHdr, err := ParseRecordHeader(ip.Data, cur)
	if err != nil {
		return nil, err
	}
	cur += RecordHeaderSize
	if !bytes.Equal(ip.Data[cur:cur+SystemRecordBytes], LitSupremum) {
		return nil, fmt.Errorf("SUPREMUM literal mismatch at %d", cur)
	}
	sup := GenericRecord{PageNumber: ip.PageNo, Header: supHdr, PrimaryKeyPos: cur}
	cur += SystemRecordBytes
	_ = cur

	// Directory slots read from the end of page and reversed
	n := int(hdr.NumDirSlots)
	dir := make([]uint16, n)
	start := PageSize - FilTrailerSize - n*PageDirSlotSize
	for i := 0; i < n; i++ {
		val, _ := be16(ip.Data, start+i*2)
		dir[n-i-1] = val
	}

	return &IndexPage{
		Inner: ip, Hdr: hdr, Fseg: fseg,
		Infimum: inf, Supremum: sup, DirSlots: dir,
	}, nil
}

func (p *IndexPage) IsLeaf() bool { return p.Hdr.PageLevel == 0 }
func (p *IndexPage) IsRoot() bool { return p.Inner.FIL.Prev == nil && p.Inner.FIL.Next == nil }

// UsedBytes matches the calculation in the Java reference project
func (p *IndexPage) UsedBytes() int {
	return int(p.Hdr.HeapTop) + FilTrailerSize + int(p.Hdr.NumDirSlots)*PageDirSlotSize - int(p.Hdr.GarbageSpace)
}

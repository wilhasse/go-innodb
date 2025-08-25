// index_page.go - INDEX page parsing with records and directory
package page

import (
	"bytes"
	"fmt"
	"github.com/wilhasse/go-innodb/format"
	"github.com/wilhasse/go-innodb/record"
)

var (
	LitInfimum  = []byte("infimum\x00")
	LitSupremum = []byte("supremum")
)

type IndexPage struct {
	Inner    *InnerPage
	Hdr      record.IndexHeader
	Fseg     FsegHeader
	Infimum  record.GenericRecord
	Supremum record.GenericRecord
	DirSlots []uint16 // dirSlots[0] is the first slot (reversed from end of page)
}

func ParseIndexPage(ip *InnerPage) (*IndexPage, error) {
	if ip.FIL.PageType != format.PageTypeIndex {
		return nil, fmt.Errorf("not an INDEX page: type=%d", ip.FIL.PageType)
	}
	hdr, err := record.ParseIndexHeader(ip.Data, format.FilHeaderSize)
	if err != nil {
		return nil, err
	}
	if hdr.Format != format.FormatCompact {
		return nil, fmt.Errorf("only compact pages supported (format=%d)", hdr.Format)
	}
	fseg, err := ParseFsegHeader(ip.Data, format.FilHeaderSize+36)
	if err != nil {
		return nil, err
	}

	cur := format.FilHeaderSize + format.PageHeaderSize

	// INFIMUM
	infHdr, err := record.ParseRecordHeader(ip.Data, cur)
	if err != nil {
		return nil, err
	}
	cur += format.RecordHeaderSize
	if !bytes.Equal(ip.Data[cur:cur+format.SystemRecordBytes], LitInfimum) {
		return nil, fmt.Errorf("INFIMUM literal mismatch at %d", cur)
	}
	inf := record.GenericRecord{PageNumber: ip.PageNo, Header: infHdr, PrimaryKeyPos: cur, Data: ip.Data[cur : cur+format.SystemRecordBytes]}
	cur += format.SystemRecordBytes

	// SUPREMUM
	supHdr, err := record.ParseRecordHeader(ip.Data, cur)
	if err != nil {
		return nil, err
	}
	cur += format.RecordHeaderSize
	if !bytes.Equal(ip.Data[cur:cur+format.SystemRecordBytes], LitSupremum) {
		return nil, fmt.Errorf("SUPREMUM literal mismatch at %d", cur)
	}
	sup := record.GenericRecord{PageNumber: ip.PageNo, Header: supHdr, PrimaryKeyPos: cur, Data: ip.Data[cur : cur+format.SystemRecordBytes]}
	cur += format.SystemRecordBytes
	_ = cur

	// Directory slots read from the end of page and reversed
	n := int(hdr.NumDirSlots)
	dir := make([]uint16, n)
	start := format.PageSize - format.FilTrailerSize - n*format.PageDirSlotSize
	for i := 0; i < n; i++ {
		val, _ := format.Be16(ip.Data, start+i*2)
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
	return int(p.Hdr.HeapTop) + format.FilTrailerSize + int(p.Hdr.NumDirSlots)*format.PageDirSlotSize - int(p.Hdr.GarbageSpace)
}

// WalkRecords walks records on a page following the compact record header's relative next offset.
// If skipSystem is true, INFIMUM and SUPREMUM are not returned.
// max limits the number of records to traverse (safety).
func (p *IndexPage) WalkRecords(max int, skipSystem bool) ([]record.GenericRecord, error) {
	if p.Hdr.Format != format.FormatCompact {
		return nil, fmt.Errorf("only compact format supported in WalkRecords")
	}
	return record.WalkRecordsFromData(p.Inner.PageNo, p.Inner.Data, p.Infimum, max, skipSystem)
}

// record_header.go - Compact record format header parsing (5 bytes)
package record

import (
	"fmt"
	"github.com/wilhasse/go-innodb/format"
)

// Compact record 5-byte header
type RecordHeader struct {
	FlagsMinRec   bool
	FlagsDeleted  bool
	NumOwned      uint8
	HeapNumber    uint16
	Type          format.RecordType
	NextRecOffset int // relative offset from this record's content start to next record's content start
}

func ParseRecordHeader(p []byte, off int) (RecordHeader, error) {
	if off+format.RecordHeaderSize > len(p) {
		return RecordHeader{}, fmt.Errorf("short record header")
	}
	b1 := p[off]
	flags := (b1 & 0xF0) >> 4
	nOwned := b1 & 0x0F
	b2, _ := format.Be16(p, off+1)
	rtype := format.RecordType(b2 & 0x0007)
	heap := (b2 & 0xFFF8) >> 3
	nxtU, _ := format.Be16(p, off+3)
	next := int(int16(nxtU))
	return RecordHeader{
		FlagsMinRec:   (flags & 0x1) != 0,
		FlagsDeleted:  (flags & 0x2) != 0,
		NumOwned:      uint8(nOwned),
		HeapNumber:    heap,
		Type:          rtype,
		NextRecOffset: next,
	}, nil
}

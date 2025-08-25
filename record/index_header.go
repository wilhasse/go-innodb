// index_header.go - Index-specific header parsing within a page
package record

import (
	"fmt"
	"github.com/wilhasse/go-innodb/format"
)

// 36-byte index header (compact/redundant flag in high bit of num-of-heap)
type IndexHeader struct {
	NumDirSlots           uint16
	HeapTop               uint16
	NumHeapRecs           uint16 // low 15 bits
	Format                format.PageFormat
	FirstGarbageOff       uint16
	GarbageSpace          uint16
	LastInsertPos         uint16
	Direction             format.PageDirection
	NumInsertsInDirection uint16
	NumUserRecs           uint16
	MaxTrxID              uint64
	PageLevel             uint16
	IndexID               uint64
}

func ParseIndexHeader(p []byte, off int) (IndexHeader, error) {
	if off+36 > len(p) {
		return IndexHeader{}, fmt.Errorf("short index header")
	}
	nSlots, _ := format.Be16(p, off+0)
	heapTop, _ := format.Be16(p, off+2)
	flag, _ := format.Be16(p, off+4)
	firstGarbage, _ := format.Be16(p, off+6)
	garbage, _ := format.Be16(p, off+8)
	lastIns, _ := format.Be16(p, off+10)
	dir, _ := format.Be16(p, off+12)
	nDir, _ := format.Be16(p, off+14)
	nRecs, _ := format.Be16(p, off+16)
	maxTrx, _ := format.Be64(p, off+18)
	level, _ := format.Be16(p, off+26)
	indexID, _ := format.Be64(p, off+28)

	fmt := format.FormatRedundant
	if (flag & 0x8000) != 0 {
		fmt = format.FormatCompact
	}

	return IndexHeader{
		NumDirSlots:           nSlots,
		HeapTop:               heapTop,
		NumHeapRecs:           flag & 0x7fff,
		Format:                fmt,
		FirstGarbageOff:       firstGarbage,
		GarbageSpace:          garbage,
		LastInsertPos:         lastIns,
		Direction:             format.PageDirection(dir),
		NumInsertsInDirection: nDir,
		NumUserRecs:           nRecs,
		MaxTrxID:              maxTrx,
		PageLevel:             level,
		IndexID:               indexID,
	}, nil
}

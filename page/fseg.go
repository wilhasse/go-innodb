// fseg_header.go - File segment header parsing
package page

import (
	"fmt"
	"github.com/wilhasse/go-innodb/format"
)

// 20-byte file segment header (root uses it; others are usually zero-filled)
type FsegHeader struct {
	LeafInodeSpace    uint32
	LeafInodePage     uint32
	LeafInodeOff      uint16
	NonLeafInodeSpace uint32
	NonLeafInodePage  uint32
	NonLeafInodeOff   uint16
}

func ParseFsegHeader(p []byte, off int) (FsegHeader, error) {
	if off+20 > len(p) {
		return FsegHeader{}, fmt.Errorf("short fseg header")
	}
	lsp, _ := format.Be32(p, off+0)
	lpg, _ := format.Be32(p, off+4)
	lof, _ := format.Be16(p, off+8)
	nsp, _ := format.Be32(p, off+10)
	npg, _ := format.Be32(p, off+14)
	nof, _ := format.Be16(p, off+18)
	return FsegHeader{
		LeafInodeSpace: lsp, LeafInodePage: lpg, LeafInodeOff: lof,
		NonLeafInodeSpace: nsp, NonLeafInodePage: npg, NonLeafInodeOff: nof,
	}, nil
}

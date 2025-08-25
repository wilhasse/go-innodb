// fseg_header.go - File segment header parsing
package goinnodb

import "fmt"

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
	lsp, _ := be32(p, off+0)
	lpg, _ := be32(p, off+4)
	lof, _ := be16(p, off+8)
	nsp, _ := be32(p, off+10)
	npg, _ := be32(p, off+14)
	nof, _ := be16(p, off+18)
	return FsegHeader{
		LeafInodeSpace: lsp, LeafInodePage: lpg, LeafInodeOff: lof,
		NonLeafInodeSpace: nsp, NonLeafInodePage: npg, NonLeafInodeOff: nof,
	}, nil
}

package goinnodb

import (
	"encoding/binary"
	"errors"
)

func be16(b []byte, off int) (uint16, error) {
	if off < 0 || off+2 > len(b) {
		return 0, errors.New("be16 out of bounds")
	}
	return binary.BigEndian.Uint16(b[off : off+2]), nil
}
func be32(b []byte, off int) (uint32, error) {
	if off < 0 || off+4 > len(b) {
		return 0, errors.New("be32 out of bounds")
	}
	return binary.BigEndian.Uint32(b[off : off+4]), nil
}
func be64(b []byte, off int) (uint64, error) {
	if off < 0 || off+8 > len(b) {
		return 0, errors.New("be64 out of bounds")
	}
	return binary.BigEndian.Uint64(b[off : off+8]), nil
}

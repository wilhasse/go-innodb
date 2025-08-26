// compressed_nocgo.go - Stub implementation when cgo is not available
// +build !cgo

package goinnodb

import "fmt"

const (
	// Logical page size is always 16KB for InnoDB
	LogicalPageSize = 16384
)

// CompressedPageSizes lists the valid physical sizes for compressed pages
var CompressedPageSizes = []int{1024, 2048, 4096, 8192}

// IsPageCompressed checks if a page appears to be compressed
// Without cgo, we can only do basic heuristic checks
func IsPageCompressed(data []byte) bool {
	if len(data) >= LogicalPageSize {
		return false
	}
	
	// Basic check: compressed pages are smaller than 16KB
	// and should be one of the valid compressed sizes
	for _, size := range CompressedPageSizes {
		if len(data) == size {
			return true
		}
	}
	
	return false
}

// DecompressPage is not available without cgo
func DecompressPage(src []byte, physicalSize int) ([]byte, error) {
	return nil, fmt.Errorf("compressed page support requires cgo (libinnodb_zipdecompress)")
}

// GetCompressedSize returns 0 without cgo support
func GetCompressedSize(page []byte, physicalSize int) int {
	return physicalSize // Best guess
}

// TryDecompressPage returns error without cgo
func TryDecompressPage(data []byte) ([]byte, bool, error) {
	if IsPageCompressed(data) {
		return data, false, fmt.Errorf("compressed page detected but decompression requires cgo")
	}
	return data, false, nil
}
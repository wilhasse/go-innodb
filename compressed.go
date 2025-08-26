// compressed.go - Support for InnoDB compressed pages
// Uses cgo to call the C++ shim library for decompression

package goinnodb

// #cgo CFLAGS: -I${SRCDIR}/lib
// #cgo LDFLAGS: -L${SRCDIR}/lib -lzipshim -lstdc++ -lz -llz4
// #include <stdlib.h>
// int innodb_zip_decompress(const void* src, size_t physical, void* dst, size_t logical);
// int innodb_is_page_compressed(const void* page, size_t size);
// size_t innodb_get_compressed_size(const void* page, size_t physical);
import "C"
import (
	"fmt"
	"unsafe"
)

const (
	// Logical page size is always 16KB for InnoDB
	LogicalPageSize = 16384
)

// CompressedPageSizes lists the valid physical sizes for compressed pages
var CompressedPageSizes = []int{1024, 2048, 4096, 8192}

// AllValidPageSizes includes both compressed and uncompressed sizes
var AllValidPageSizes = []int{1024, 2048, 4096, 8192, 16384}

// IsPageCompressed checks if a page appears to be compressed
// This is a heuristic check based on page header patterns
func IsPageCompressed(data []byte) bool {
	if len(data) < 38 {
		return false
	}

	// Check FIL header for compression indicators
	// Compressed pages have different header patterns

	// Simple heuristic: if page size is less than 16KB and has valid FIL header
	if len(data) < LogicalPageSize {
		// Could be compressed
		// Additional checks would go here
		return true
	}

	// Call C function for more sophisticated check
	// (though our simple implementation just returns 0 for now)
	result := C.innodb_is_page_compressed(
		unsafe.Pointer(&data[0]),
		C.size_t(len(data)),
	)

	return result != 0
}

// DecompressPage decompresses an InnoDB compressed page
// src: compressed page data
// physicalSize: size of compressed page (1K, 2K, 4K, 8K)
// Returns: decompressed 16KB page or error
func DecompressPage(src []byte, physicalSize int) ([]byte, error) {
	// Validate physical size
	validSize := false
	for _, size := range CompressedPageSizes {
		if physicalSize == size {
			validSize = true
			break
		}
	}
	if !validSize {
		return nil, fmt.Errorf("invalid physical page size: %d", physicalSize)
	}

	if len(src) < physicalSize {
		return nil, fmt.Errorf("source data too small: %d < %d", len(src), physicalSize)
	}

	// Allocate output buffer for logical page
	dst := make([]byte, LogicalPageSize)

	// Call the decompression function (following Percona/Oracle approach)
	rc := C.innodb_zip_decompress(
		unsafe.Pointer(&src[0]),
		C.size_t(physicalSize),
		unsafe.Pointer(&dst[0]),
		C.size_t(LogicalPageSize),
	)

	switch rc {
	case 0:
		return dst, nil
	case -1:
		return nil, fmt.Errorf("invalid parameters")
	case -2:
		return nil, fmt.Errorf("invalid logical page size")
	case -3:
		return nil, fmt.Errorf("invalid physical page size")
	case -4:
		return nil, fmt.Errorf("decompression failed")
	default:
		return nil, fmt.Errorf("unknown error: %d", rc)
	}
}

// GetCompressedSize returns the actual compressed data size from a compressed page
func GetCompressedSize(page []byte, physicalSize int) int {
	if len(page) < physicalSize {
		return 0
	}

	size := C.innodb_get_compressed_size(
		unsafe.Pointer(&page[0]),
		C.size_t(physicalSize),
	)

	return int(size)
}

// TryDecompressPage attempts to decompress a page if it appears compressed
// Returns the decompressed page or the original if not compressed
func TryDecompressPage(data []byte) ([]byte, bool, error) {
	// If already 16KB, probably not compressed
	if len(data) == LogicalPageSize {
		return data, false, nil
	}

	// Try to detect compression
	if !IsPageCompressed(data) {
		return data, false, nil
	}

	// Try different physical sizes
	for _, size := range CompressedPageSizes {
		if len(data) >= size {
			decompressed, err := DecompressPage(data, size)
			if err == nil {
				return decompressed, true, nil
			}
		}
	}

	// Couldn't decompress, return original
	return data, false, fmt.Errorf("unable to decompress page")
}

// decompress.go - Go wrapper for InnoDB page decompression
// This provides a clean Go API for decompressing InnoDB pages
// without any MySQL dependencies

package goinnodb

// #cgo CFLAGS: -I${SRCDIR}/lib
// #cgo LDFLAGS: -L${SRCDIR}/lib -linnodb_decompress -lstdc++ -lz -llz4
// #include <stdlib.h>
// #include "innodb_decompress.h"
//
// // Wrapper functions to handle const unsigned char* properly
// static int wrap_innodb_is_page_compressed(unsigned char* data, size_t size) {
//     return innodb_is_page_compressed((const unsigned char*)data, size);
// }
//
// static int wrap_innodb_get_page_info(unsigned char* data, size_t size, innodb_page_info_t* info) {
//     return innodb_get_page_info((const unsigned char*)data, size, info);
// }
//
// static int wrap_innodb_decompress_page(unsigned char* in, size_t in_size, unsigned char* out, size_t out_size, size_t* written) {
//     return innodb_decompress_page((const unsigned char*)in, in_size, out, out_size, written);
// }
//
// static int wrap_innodb_process_page(unsigned char* in, size_t in_size, unsigned char* out, size_t out_size, size_t* written) {
//     return innodb_process_page((const unsigned char*)in, in_size, out, out_size, written);
// }
import "C"
import (
	"fmt"
	"unsafe"
)

// Error codes from the C library
const (
	DecompressSuccess          = 0
	DecompressErrorInvalidSize = -1
	DecompressErrorNotCompressed = -2
	DecompressErrorFailed      = -3
	DecompressErrorBufferSmall = -4
	DecompressErrorInvalidPage = -5
)

// PageInfo contains metadata about an InnoDB page
type PageInfo struct {
	PageNumber    uint32 // Page number from header
	PageType      uint16 // Page type (FIL_PAGE_INDEX, etc.)
	SpaceID       uint32 // Tablespace ID
	IsCompressed  bool   // Whether the page is compressed
	PhysicalSize  int    // Size on disk
	LogicalSize   int    // Size when uncompressed (always 16KB)
}

// DecompressError represents an error from the decompression library
type DecompressError struct {
	Code    int
	Message string
}

func (e *DecompressError) Error() string {
	return fmt.Sprintf("decompress error %d: %s", e.Code, e.Message)
}

// newDecompressError creates a DecompressError from a C error code
func newDecompressError(code C.int) error {
	if code == 0 {
		return nil
	}
	errStr := C.GoString(C.innodb_decompress_error_string(code))
	return &DecompressError{
		Code:    int(code),
		Message: errStr,
	}
}

// IsPageCompressedV2 checks if a page appears to be compressed
// (V2 suffix to avoid conflict with existing IsPageCompressed in compressed.go)
func IsPageCompressedV2(data []byte) (bool, error) {
	if len(data) == 0 {
		return false, fmt.Errorf("empty page data")
	}

	var dataPtr *C.uchar = (*C.uchar)(unsafe.Pointer(&data[0]))
	result := C.wrap_innodb_is_page_compressed(
		dataPtr,
		C.size_t(len(data)))

	switch result {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, fmt.Errorf("invalid page format")
	}
}

// GetPageInfo retrieves metadata about an InnoDB page
func GetPageInfo(data []byte) (*PageInfo, error) {
	if len(data) < 38 { // Minimum size for FIL header
		return nil, fmt.Errorf("page too small: %d bytes", len(data))
	}

	var cInfo C.innodb_page_info_t
	var dataPtr *C.uchar = (*C.uchar)(unsafe.Pointer(&data[0]))
	code := C.wrap_innodb_get_page_info(
		dataPtr,
		C.size_t(len(data)),
		&cInfo,
	)

	if code != 0 {
		return nil, newDecompressError(code)
	}

	return &PageInfo{
		PageNumber:   uint32(cInfo.page_number),
		PageType:     uint16(cInfo.page_type),
		SpaceID:      uint32(cInfo.space_id),
		IsCompressed: cInfo.is_compressed != 0,
		PhysicalSize: int(cInfo.physical_size),
		LogicalSize:  int(cInfo.logical_size),
	}, nil
}

// DecompressPageV2 decompresses a compressed InnoDB page
// (V2 suffix to avoid conflict with existing DecompressPage in compressed.go)
func DecompressPageV2(compressedData []byte) ([]byte, error) {
	if len(compressedData) == 0 {
		return nil, fmt.Errorf("empty compressed data")
	}

	// Allocate output buffer (16KB for decompressed page)
	outputSize := 16384
	output := make([]byte, outputSize)
	var bytesWritten C.size_t

	var inPtr *C.uchar = (*C.uchar)(unsafe.Pointer(&compressedData[0]))
	var outPtr *C.uchar = (*C.uchar)(unsafe.Pointer(&output[0]))
	code := C.wrap_innodb_decompress_page(
		inPtr,
		C.size_t(len(compressedData)),
		outPtr,
		C.size_t(outputSize),
		&bytesWritten,
	)

	if code != 0 {
		return nil, newDecompressError(code)
	}

	// Resize to actual bytes written
	return output[:bytesWritten], nil
}

// ProcessPage handles both compressed and uncompressed pages
// It automatically detects if decompression is needed
func ProcessPage(pageData []byte) ([]byte, error) {
	if len(pageData) == 0 {
		return nil, fmt.Errorf("empty page data")
	}

	// Allocate output buffer
	outputSize := 16384
	if len(pageData) > outputSize {
		outputSize = len(pageData)
	}
	output := make([]byte, outputSize)
	var bytesWritten C.size_t

	var inPtr *C.uchar = (*C.uchar)(unsafe.Pointer(&pageData[0]))
	var outPtr *C.uchar = (*C.uchar)(unsafe.Pointer(&output[0]))
	code := C.wrap_innodb_process_page(
		inPtr,
		C.size_t(len(pageData)),
		outPtr,
		C.size_t(outputSize),
		&bytesWritten,
	)

	if code != 0 {
		return nil, newDecompressError(code)
	}

	return output[:bytesWritten], nil
}

// GetDecompressVersion returns the version of the decompression library
func GetDecompressVersion() string {
	return C.GoString(C.innodb_decompress_version())
}

// Helper function to detect compressed page size from file size pattern
func DetectCompressedSize(size int64) (int, bool) {
	// Check if size is a multiple of common compressed page sizes
	sizes := []int{1024, 2048, 4096, 8192}
	for _, pageSize := range sizes {
		if size%int64(pageSize) == 0 {
			return pageSize, true
		}
	}
	
	// Check for uncompressed
	if size%16384 == 0 {
		return 16384, false
	}
	
	return 0, false
}
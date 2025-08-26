// reader_compressed.go - Enhanced page reader with compression support
package goinnodb

import (
	"fmt"
	"github.com/wilhasse/go-innodb/format"
	"github.com/wilhasse/go-innodb/page"
	"io"
)

// CompressedPageReader extends PageReader with compression support
type CompressedPageReader struct {
	r                  io.ReaderAt
	enableDecompression bool
	physicalPageSize   int // Physical page size for compressed tables (0 = auto-detect)
}

// NewCompressedPageReader creates a reader with compression support
func NewCompressedPageReader(r io.ReaderAt) *CompressedPageReader {
	return &CompressedPageReader{
		r:                  r,
		enableDecompression: true,
		physicalPageSize:   0, // Auto-detect
	}
}

// SetPhysicalPageSize sets the physical page size for compressed tables
// Valid sizes are 1024, 2048, 4096, 8192 bytes
func (pr *CompressedPageReader) SetPhysicalPageSize(size int) error {
	validSize := false
	for _, s := range CompressedPageSizes {
		if size == s {
			validSize = true
			break
		}
	}
	if !validSize && size != 0 {
		return fmt.Errorf("invalid physical page size: %d", size)
	}
	pr.physicalPageSize = size
	return nil
}

// ReadPage reads a page and automatically decompresses if needed
func (pr *CompressedPageReader) ReadPage(pageNo uint32) (*page.InnerPage, error) {
	// Determine read size
	readSize := format.PageSize
	if pr.physicalPageSize > 0 && pr.physicalPageSize < format.PageSize {
		readSize = pr.physicalPageSize
	}
	
	// Read the page data
	buf := make([]byte, readSize)
	off := int64(pageNo) * int64(readSize)
	if _, err := pr.r.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("read page %d: %w", pageNo, err)
	}
	
	// Try to decompress if enabled and page appears compressed
	if pr.enableDecompression {
		decompressed, wasCompressed, err := pr.tryDecompress(buf)
		if err != nil {
			// Log warning but continue with original data
			// Some pages might not be compressed even in compressed tables
			fmt.Printf("Warning: decompression failed for page %d: %v\n", pageNo, err)
		} else if wasCompressed {
			buf = decompressed
		}
	}
	
	// Parse the page (now guaranteed to be logical size if decompressed)
	return page.NewInnerPage(pageNo, buf)
}

// ReadCompressedPage explicitly reads and decompresses a compressed page
func (pr *CompressedPageReader) ReadCompressedPage(pageNo uint32, physicalSize int) (*page.InnerPage, error) {
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
	
	// Read compressed data
	buf := make([]byte, physicalSize)
	off := int64(pageNo) * int64(physicalSize)
	if _, err := pr.r.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("read compressed page %d: %w", pageNo, err)
	}
	
	// Decompress
	decompressed, err := DecompressPage(buf, physicalSize)
	if err != nil {
		return nil, fmt.Errorf("decompress page %d: %w", pageNo, err)
	}
	
	// Parse decompressed page
	return page.NewInnerPage(pageNo, decompressed)
}

// tryDecompress attempts to decompress a page if it appears compressed
func (pr *CompressedPageReader) tryDecompress(data []byte) ([]byte, bool, error) {
	// If we have a specific physical size set, use it
	if pr.physicalPageSize > 0 && pr.physicalPageSize < format.PageSize {
		if len(data) == pr.physicalPageSize {
			decompressed, err := DecompressPage(data, pr.physicalPageSize)
			if err == nil {
				return decompressed, true, nil
			}
			return data, false, err
		}
	}
	
	// Otherwise, try auto-detection
	return TryDecompressPage(data)
}

// DisableDecompression turns off automatic decompression
func (pr *CompressedPageReader) DisableDecompression() {
	pr.enableDecompression = false
}

// EnableDecompression turns on automatic decompression
func (pr *CompressedPageReader) EnableDecompression() {
	pr.enableDecompression = true
}
# API Documentation

This document provides detailed API documentation for the InnoDB Page Parser library.

## Package: `goinnodb`

### Constants

```go
const (
    PageSize          = 16384  // InnoDB page size (16KB)
    FilHeaderSize     = 38     // FIL header size in bytes
    FilTrailerSize    = 8      // FIL trailer size in bytes
    RecordHeaderSize  = 5      // Compact record header size
    SystemRecordBytes = 8      // Size of system records (INFIMUM/SUPREMUM)
    PageDirSlotSize   = 2      // Page directory slot size
    PageHeaderSize    = 56     // Total page header size (index + FSEG)
    PageDataOff       = 94     // Offset where data begins (after headers)
)
```

### Page Types

```go
type PageType uint16

const (
    PageTypeAllocated PageType = 0      // Freshly allocated page
    PageTypeIndex     PageType = 17855  // B-tree index page
    PageTypeUndoLog   PageType = 2      // Undo log page
    PageTypeSDI       PageType = 17853  // Serialized Dictionary Information
)
```

### Page Formats

```go
type PageFormat uint8

const (
    FormatRedundant PageFormat = 0  // Old redundant format
    FormatCompact   PageFormat = 1  // Compact format (supported)
)
```

### Record Types

```go
type RecordType uint8

const (
    RecConventional RecordType = 0  // Regular user record
    RecNodePointer  RecordType = 1  // Node pointer in non-leaf page
    RecInfimum      RecordType = 2  // System infimum record
    RecSupremum     RecordType = 3  // System supremum record
)
```

## Core Types

### PageReader

```go
type PageReader struct {
    // Private fields
}

// NewPageReader creates a new page reader from an io.ReaderAt
func NewPageReader(r io.ReaderAt) *PageReader

// ReadPage reads a page at the specified page number
func (pr *PageReader) ReadPage(pageNo uint32) (*InnerPage, error)
```

**Usage Example:**
```go
file, _ := os.Open("data.ibd")
reader := goinnodb.NewPageReader(file)
page, err := reader.ReadPage(3)
```

### InnerPage

```go
type InnerPage struct {
    PageNo  uint32      // Page number
    FIL     FilHeader   // FIL header
    Trailer FilTrailer  // FIL trailer
    Data    []byte      // Full 16KB page data
}

// NewInnerPage creates and validates a new inner page
func NewInnerPage(pageNo uint32, page []byte) (*InnerPage, error)

// PageType returns the type of this page
func (ip *InnerPage) PageType() PageType
```

### FilHeader

```go
type FilHeader struct {
    Checksum   uint32    // Page checksum
    PageNumber uint32    // Page number
    Prev       *uint32   // Previous page (nil if none)
    Next       *uint32   // Next page (nil if none)
    LastModLSN uint64    // Last modification LSN
    PageType   PageType  // Type of page
    FlushLSN   uint64    // Flush LSN
    SpaceID    uint32    // Tablespace ID
}

// ParseFilHeader parses a FIL header from page bytes
func ParseFilHeader(p []byte) (FilHeader, error)
```

### FilTrailer

```go
type FilTrailer struct {
    Checksum uint32  // Checksum (old format)
    Low32LSN uint32  // Low 32 bits of LSN
}

// ParseFilTrailer parses a FIL trailer from page bytes
func ParseFilTrailer(p []byte) (FilTrailer, error)
```

### IndexPage

```go
type IndexPage struct {
    Inner    *InnerPage     // Underlying page
    Hdr      IndexHeader    // Index header
    Fseg     FsegHeader     // File segment header
    Infimum  GenericRecord  // Infimum system record
    Supremum GenericRecord  // Supremum system record
    DirSlots []uint16       // Page directory slots
}

// ParseIndexPage parses an index page from an inner page
func ParseIndexPage(ip *InnerPage) (*IndexPage, error)

// IsLeaf returns true if this is a leaf page
func (p *IndexPage) IsLeaf() bool

// IsRoot returns true if this is the root page
func (p *IndexPage) IsRoot() bool

// UsedBytes returns the number of used bytes in the page
func (p *IndexPage) UsedBytes() int

// WalkRecords traverses records in the page
// max: maximum number of records to return
// skipSystem: if true, skip INFIMUM and SUPREMUM records
func (p *IndexPage) WalkRecords(max int, skipSystem bool) ([]GenericRecord, error)
```

### IndexHeader

```go
type IndexHeader struct {
    NumDirSlots           uint16        // Number of directory slots
    HeapTop               uint16        // Heap top position
    NumHeapRecs           uint16        // Number of heap records
    Format                PageFormat    // Page format
    FirstGarbageOff       uint16        // First garbage record offset
    GarbageSpace          uint16        // Total garbage space
    LastInsertPos         uint16        // Last insert position
    Direction             PageDirection // Insert direction
    NumInsertsInDirection uint16        // Consecutive inserts in direction
    NumUserRecs           uint16        // Number of user records
    MaxTrxID              uint64        // Maximum transaction ID
    PageLevel             uint16        // B-tree level (0=leaf)
    IndexID               uint64        // Index identifier
}

// ParseIndexHeader parses an index header from page bytes
func ParseIndexHeader(p []byte, off int) (IndexHeader, error)
```

### GenericRecord

```go
type GenericRecord struct {
    PageNumber      uint32        // Page number containing record
    Header          RecordHeader  // Record header
    PrimaryKeyPos   int          // Offset where content starts
    ChildPageNumber uint32       // Child page (for internal pages)
}

// NextRecordPos returns the position of the next record
func (r GenericRecord) NextRecordPos() int
```

### RecordHeader

```go
type RecordHeader struct {
    FlagsMinRec   bool       // Minimum record flag
    FlagsDeleted  bool       // Deleted flag
    NumOwned      uint8      // Number of records owned
    HeapNumber    uint16     // Heap number
    Type          RecordType // Record type
    NextRecOffset int        // Relative offset to next record
}

// ParseRecordHeader parses a compact record header
func ParseRecordHeader(p []byte, off int) (RecordHeader, error)
```

### FsegHeader

```go
type FsegHeader struct {
    LeafInodeSpace    uint32  // Leaf inode space ID
    LeafInodePage     uint32  // Leaf inode page number
    LeafInodeOff      uint16  // Leaf inode offset
    NonLeafInodeSpace uint32  // Non-leaf inode space ID
    NonLeafInodePage  uint32  // Non-leaf inode page number
    NonLeafInodeOff   uint16  // Non-leaf inode offset
}

// ParseFsegHeader parses a file segment header
func ParseFsegHeader(p []byte, off int) (FsegHeader, error)
```

## Helper Functions

### Endian Conversion

```go
// Read big-endian values from byte slices
func be16(b []byte, off int) (uint16, error)  // Read 16-bit big-endian
func be32(b []byte, off int) (uint32, error)  // Read 32-bit big-endian
func be64(b []byte, off int) (uint64, error)  // Read 64-bit big-endian
```

## Error Handling

All parsing functions return errors for:
- Invalid page sizes
- Corrupted headers
- Checksum mismatches
- Out-of-bounds offsets
- Unsupported formats

Example error handling:
```go
page, err := reader.ReadPage(3)
if err != nil {
    // Handle specific error types
    switch {
    case strings.Contains(err.Error(), "LSN mismatch"):
        // Page corruption detected
    case strings.Contains(err.Error(), "short page"):
        // Invalid page size
    default:
        // Other errors
    }
}
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "os"
    goinnodb "github.com/wilhasse/go-innodb"
)

func analyzeTablespace(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    reader := goinnodb.NewPageReader(file)

    // Read root page (usually page 3 for user tables)
    rootPage, err := reader.ReadPage(3)
    if err != nil {
        return fmt.Errorf("failed to read root page: %w", err)
    }

    // Verify it's an index page
    if rootPage.PageType() != goinnodb.PageTypeIndex {
        return fmt.Errorf("page 3 is not an index page")
    }

    // Parse as index page
    indexPage, err := goinnodb.ParseIndexPage(rootPage)
    if err != nil {
        return fmt.Errorf("failed to parse index page: %w", err)
    }

    // Analyze page structure
    fmt.Printf("Index ID: %d\n", indexPage.Hdr.IndexID)
    fmt.Printf("Page Level: %d\n", indexPage.Hdr.PageLevel)
    fmt.Printf("Number of Records: %d\n", indexPage.Hdr.NumUserRecs)
    fmt.Printf("Page Usage: %d/%d bytes\n", 
        indexPage.UsedBytes(), goinnodb.PageSize)

    // Walk through records
    records, err := indexPage.WalkRecords(10, true)
    if err != nil {
        return fmt.Errorf("failed to walk records: %w", err)
    }

    for i, rec := range records {
        fmt.Printf("Record %d: heap_no=%d, type=%d, deleted=%v\n",
            i, rec.Header.HeapNumber, rec.Header.Type, 
            rec.Header.FlagsDeleted)
    }

    // Follow page links
    if rootPage.FIL.Next != nil {
        fmt.Printf("Next page: %d\n", *rootPage.FIL.Next)
    }

    return nil
}

func main() {
    if err := analyzeTablespace("data.ibd"); err != nil {
        log.Fatal(err)
    }
}
```
# InnoDB Compressed Page Support

This document describes how to use the InnoDB compressed page decompression feature in go-innodb.

## Overview

InnoDB supports table compression using the `ROW_FORMAT=COMPRESSED` option with `KEY_BLOCK_SIZE` settings. Compressed tables store pages in a smaller physical size (1K, 2K, 4K, or 8K) instead of the standard 16K logical size.

## Prerequisites

### Required Library

You need the InnoDB decompression library from MySQL:
- `libinnodb_zipdecompress.a` - Static library from MySQL source

Place this file in the `lib/` directory of the project.

### System Dependencies

The compression support requires:
- `liblz4` - LZ4 compression library
- `zlib` - Standard compression library
- C++ compiler (g++ or clang++)
- CGO enabled in your Go environment

Install dependencies:
```bash
# Ubuntu/Debian
sudo apt-get install liblz4-dev zlib1g-dev g++

# macOS
brew install lz4 zlib

# RHEL/CentOS
sudo yum install lz4-devel zlib-devel gcc-c++
```

## Building

### Quick Build

```bash
# Make the build script executable
chmod +x build_compression.sh

# Build compression support
./build_compression.sh
```

### Manual Build

```bash
# Build the C++ shim library
cd lib
g++ -fPIC -O2 -Wall -std=c++11 -c zipshim.cpp
g++ -shared -o libzipshim.so zipshim.o libinnodb_zipdecompress.a -lz -llz4
cd ..

# Build Go code with cgo
go build -tags cgo
```

## Usage

### Basic Usage

```go
import (
    "os"
    innodb "github.com/wilhasse/go-innodb"
)

func main() {
    file, _ := os.Open("compressed_table.ibd")
    defer file.Close()
    
    // Create reader with compression support
    reader := innodb.NewCompressedPageReader(file)
    
    // Read page 4 (auto-detects compression)
    page, err := reader.ReadPage(4)
    if err != nil {
        panic(err)
    }
    
    // Page is automatically decompressed if needed
    fmt.Printf("Page type: %d\n", page.PageType())
}
```

### Explicit Compressed Page Reading

```go
// For tables with KEY_BLOCK_SIZE=8K
reader := innodb.NewCompressedPageReader(file)

// Read compressed page with explicit physical size
page, err := reader.ReadCompressedPage(pageNo, 8192) // 8K physical size
```

### Setting Physical Page Size

```go
reader := innodb.NewCompressedPageReader(file)

// Set physical size for all reads (e.g., for 4K compressed tables)
reader.SetPhysicalPageSize(4096)

// Now all ReadPage calls will read 4K and decompress to 16K
page, err := reader.ReadPage(pageNo)
```

### Detecting Compressed Pages

```go
// Check if page data appears to be compressed
if innodb.IsPageCompressed(pageData) {
    // Try to decompress
    decompressed, err := innodb.DecompressPage(pageData, 8192)
    if err != nil {
        fmt.Printf("Decompression failed: %v\n", err)
    }
}
```

## Compressed Table Formats

InnoDB compressed tables use these KEY_BLOCK_SIZE values:

| KEY_BLOCK_SIZE | Physical Page Size | Compression Ratio |
|----------------|-------------------|-------------------|
| 1K | 1024 bytes | 16:1 |
| 2K | 2048 bytes | 8:1 |
| 4K | 4096 bytes | 4:1 |
| 8K | 8192 bytes | 2:1 |

## How Compression Works

1. **Logical vs Physical Size**:
   - Logical size: Always 16KB (how InnoDB sees the page internally)
   - Physical size: 1K, 2K, 4K, or 8K (how it's stored on disk)

2. **Compression Algorithm**:
   - Uses zlib compression with LZ77 algorithm
   - May use LZ4 for certain operations

3. **Page Structure**:
   - Compressed pages have modified headers
   - Include compression metadata
   - Store both compressed and uncompressed data in memory (for modification log)

## Example: Reading Compressed Table

```go
package main

import (
    "fmt"
    "os"
    innodb "github.com/wilhasse/go-innodb"
    "github.com/wilhasse/go-innodb/schema"
)

func main() {
    // Parse table schema
    tableDef, _ := schema.ParseTableDefFromSQLFile("table.sql")
    
    // Open compressed table file
    file, _ := os.Open("compressed_table.ibd")
    defer file.Close()
    
    // Create compressed page reader
    reader := innodb.NewCompressedPageReader(file)
    
    // For KEY_BLOCK_SIZE=4K tables
    reader.SetPhysicalPageSize(4096)
    
    // Read and decompress page
    page, err := reader.ReadPage(4)
    if err != nil {
        panic(err)
    }
    
    // Parse as index page
    indexPage, _ := innodb.ParseIndexPage(page)
    
    // Extract records with schema
    records, _ := indexPage.WalkRecordsWithSchema(tableDef, true)
    
    for _, rec := range records {
        fmt.Printf("Record: %v\n", rec.Values)
    }
}
```

## Troubleshooting

### Build Errors

**"libinnodb_zipdecompress.a not found"**
- Ensure the library is in `lib/` directory
- Check file permissions

**"undefined reference to lz4_*"**
- Install liblz4-dev: `sudo apt-get install liblz4-dev`

**"cgo not enabled"**
- Enable cgo: `CGO_ENABLED=1 go build`

### Runtime Errors

**"decompression failed"**
- Check if page is actually compressed
- Verify correct physical page size
- Page might be corrupted

**"invalid physical page size"**
- Use only 1024, 2048, 4096, or 8192
- Check table's KEY_BLOCK_SIZE setting

## Without CGO

If you can't use cgo, the library provides stub functions that return errors for compressed pages. You can still read uncompressed pages normally:

```go
// Build without cgo
// go build -tags !cgo

reader := innodb.NewPageReader(file)
page, err := reader.ReadPage(4) // Works for uncompressed pages only
```

## Performance Considerations

1. **Memory Usage**: Decompression creates a 16KB page from smaller compressed data
2. **CPU Overhead**: Decompression adds CPU cost
3. **I/O Benefits**: Smaller physical pages mean less disk I/O
4. **Caching**: Consider caching decompressed pages in your application

## Limitations

- Read-only support (cannot create compressed pages)
- Requires MySQL's libinnodb_zipdecompress library
- CGO required for compression support
- No support for transparent page compression (different from ROW_FORMAT=COMPRESSED)

## References

- [MySQL Documentation: InnoDB Table Compression](https://dev.mysql.com/doc/refman/8.0/en/innodb-compression.html)
- [InnoDB Compressed Page Format](https://dev.mysql.com/doc/internals/en/innodb-compressed-format.html)
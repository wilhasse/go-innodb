# InnoDB Compressed Page Support

This document describes how to use the InnoDB compressed page decompression feature in go-innodb.

## Overview

InnoDB supports table compression using the `ROW_FORMAT=COMPRESSED` option with `KEY_BLOCK_SIZE` settings. Compressed tables store pages in a smaller physical size (1K, 2K, 4K, or 8K) instead of the standard 16K logical size.

## Prerequisites

### Required Library

You need the InnoDB decompression library from MySQL:
- `libinnodb_zipdecompress.a` - Static library from MySQL source

**Obtaining the Library:**

1. **From MySQL Binary Distribution:**
   ```bash
   # Download MySQL server (tarball version)
   wget https://dev.mysql.com/get/Downloads/MySQL-8.0/mysql-8.0.XX-linux-glibc2.17-x86_64.tar.xz
   tar -xf mysql-8.0.XX-linux-glibc2.17-x86_64.tar.xz
   find mysql-8.0.XX-linux-glibc2.17-x86_64 -name "*zipdecompress*" -o -name "*innodb*compress*"
   ```

2. **From MySQL Source Build:**
   ```bash
   # Clone MySQL source
   git clone https://github.com/mysql/mysql-server.git
   cd mysql-server
   
   # Build with compression support
   mkdir build && cd build
   cmake .. -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_ZLIB=bundled
   make -j4
   
   # Look for the library in build directories
   find . -name "*zipdecompress*" -o -name "*innodb*compress*"
   ```

3. **Alternative Sources:**
   - Some Linux distributions package development files separately
   - Check for mysql-server-dev or similar packages
   - The library may be part of libmysqlclient-dev

**Installation:**
```bash
# Copy to project lib directory
cp /path/to/libinnodb_zipdecompress.a /path/to/go-innodb/lib/

# Verify the library
file lib/libinnodb_zipdecompress.a
nm lib/libinnodb_zipdecompress.a | grep decompress
```

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
make clean && make
cd ..

# Build Go code with cgo
go build -tags cgo
```

### Detailed Build Process

The compression support requires several components to work together:

1. **MySQL Symbol Stubs** (`mysql_stubs.cpp`)
   - Provides stub implementations for InnoDB internal symbols
   - Includes ib:: namespace classes (logger, error, warn, fatal)
   - Implements ut_dbg_assertion_failed and memory allocation functions
   - Required because libinnodb_zipdecompress.a has dependencies on MySQL internals

2. **C++ Shim Layer** (`zipshim.cpp`)
   - Provides extern "C" interface for Go's cgo to call
   - Wraps InnoDB's page_zip_decompress_low function
   - Handles page size conversions and error handling

3. **Shared Library Build** (`libzipshim.so`)
   - Links zipshim.o, mysql_stubs.o, and libinnodb_zipdecompress.a
   - Uses --whole-archive to include all InnoDB symbols
   - Statically links zlib and lz4 dependencies

## Implementation Architecture

### Integration Pattern

The compression support uses a three-layer approach:

```
Go Code (compressed.go)
    ↓ cgo calls
C Wrapper (zipshim.cpp) 
    ↓ C++ calls
InnoDB Library (libinnodb_zipdecompress.a)
    ↓ depends on  
MySQL Stubs (mysql_stubs.cpp)
```

### Key Technical Challenges Solved

1. **MySQL Symbol Dependencies**
   - InnoDB library expects MySQL server context (logging, memory management, etc.)
   - Solution: Implement minimal stub versions of required symbols
   - Critical symbols: `srv_page_size`, `ib::` logging classes, `ut_dbg_assertion_failed`

2. **C++ Name Mangling and Linkage**
   - InnoDB functions use C++ linkage with mangled names
   - Go cgo expects C-style functions
   - Solution: C++ wrapper layer with extern "C" interface

3. **Virtual Table Generation**
   - InnoDB library expects C++ vtables for ib:: logging classes
   - Virtual destructors must be defined out-of-line to generate vtables
   - Solution: Define destructor bodies separately from class declarations

4. **Library Linking Order**
   - Static library must be fully included to resolve all symbols
   - Solution: Use `--whole-archive` linker flag

### Oracle Engineers' Guidance Applied

This implementation follows guidance from Oracle MySQL engineers, adapted for environments without full MySQL source access:

**Oracle's Recommendations:**
- Use proper InnoDB types from MySQL headers (`my_config.h`, `page0zip.h`, etc.)
- Update `srv_page_size` globals at runtime for consistency
- Implement complete logger classes with ostringstream buffering
- Use `[[noreturn]]` attribute for assertion handlers
- Call InnoDB utility functions like `page_zip_des_init()` when available

**Our Adaptation (Header-Free Approach):**
- Forward-declare InnoDB types instead of including full MySQL headers
- Implement runtime srv_page_size updates with proper log2 calculations
- Use ostringstream-based loggers following Oracle's pattern
- Apply C++11/14 best practices (brace initialization, proper attributes)
- Manual page size calculations when InnoDB utilities aren't available

This approach provides production-quality compression support without requiring the full MySQL development environment.

### File Structure
```
lib/
├── libinnodb_zipdecompress.a    # InnoDB decompression library (user provided)
├── zipshim.cpp                  # C++ wrapper for cgo interface  
├── mysql_stubs.cpp              # MySQL internal symbol stubs
├── Makefile                     # Build configuration
├── libzipshim.so               # Generated shared library
└── *.o                         # Compiled object files
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
- Library must be extracted from MySQL source or binary distribution

**"undefined reference to lz4_*" or "undefined reference to uncompress"**
- Install compression libraries:
  ```bash
  # Ubuntu/Debian
  sudo apt-get install liblz4-dev zlib1g-dev
  
  # RHEL/CentOS/Fedora  
  sudo dnf install lz4-devel zlib-devel
  
  # macOS
  brew install lz4 zlib
  ```

**"undefined reference to srv_page_size" or InnoDB symbols**
- This means mysql_stubs.cpp is not being compiled/linked
- Ensure `make` is run from the `lib/` directory
- Check that both zipshim.o and mysql_stubs.o are being linked:
  ```bash
  cd lib
  make clean && make
  nm libzipshim.so | grep srv_page_size  # Should show the symbol
  ```

**"undefined reference to ib::error::~error()" or vtable errors**
- C++ vtable/destructor linking issues
- Ensure mysql_stubs.cpp defines destructors outside class declarations
- Verify C++ linking with -lstdc++:
  ```bash
  nm mysql_stubs.o | grep ib  # Should show ib:: symbols
  ```

**"undefined reference to ut_dbg_assertion_failed"**
- Function signature mismatch between extern "C" and C++ linkage
- The function must use C++ linkage (not extern "C")
- Check mangled symbol: `nm libinnodb_zipdecompress.a | grep ut_dbg`

**"multiple definition of ut_crc32"**
- Symbol conflict between stubs and InnoDB library
- Don't implement CRC functions in stubs (InnoDB library already has them)
- Remove conflicting functions from mysql_stubs.cpp

**"cgo not enabled"**
- Enable cgo: `CGO_ENABLED=1 go build`
- Verify cgo is available: `go env CGO_ENABLED`

**"cannot find -lzipshim" (build time)**
- Library not found in library path during linking
- Set LD_LIBRARY_PATH: `export LD_LIBRARY_PATH=/path/to/lib:$LD_LIBRARY_PATH`
- Or install library system-wide: `sudo cp lib/libzipshim.so /usr/local/lib/ && sudo ldconfig`

**"error while loading shared libraries: libzipshim.so: cannot open shared object file" (runtime)**
- Binary can't find shared library at runtime
- **Quick fix:** Run with library path: `LD_LIBRARY_PATH=/path/to/go-innodb/lib ./go-innodb`
- **Permanent fix:** Install system-wide:
  ```bash
  sudo cp lib/libzipshim.so /usr/local/lib/
  sudo ldconfig
  ```
- **Alternative:** Add to your shell profile:
  ```bash
  echo 'export LD_LIBRARY_PATH=/home/user/go-innodb/lib:$LD_LIBRARY_PATH' >> ~/.bashrc
  source ~/.bashrc
  ```

### Advanced Build Issues

**Symbol Resolution Problems**
Check what symbols are undefined in the InnoDB library:
```bash
nm -u lib/libinnodb_zipdecompress.a | grep -v GLIBC | sort | uniq
```

Check what symbols your stubs provide:
```bash
nm lib/mysql_stubs.o | grep -E "(srv_|ib::|ut_)"
```

**C++ Name Mangling Issues**
Demangle symbol names to understand requirements:
```bash
echo "_ZN2ib5errorD1Ev" | c++filt  # Should show: ib::error::~error()
```

**Linker Problems**
Debug linking with verbose output:
```bash
g++ -shared -o libzipshim.so zipshim.o mysql_stubs.o \
    -Wl,--whole-archive libinnodb_zipdecompress.a -Wl,--no-whole-archive \
    -lz -llz4 -lstdc++ -Wl,--verbose
```

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
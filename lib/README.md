# InnoDB Decompression Library

This directory contains a minimal C++ library for decompressing InnoDB compressed pages.

## Files

- `innodb_decompress.cpp` - Main decompression implementation
- `innodb_decompress.h` - C interface header for Go integration
- `mysql_stubs.cpp` - Minimal stubs for InnoDB symbols (logging, errors)
- `innodb_constants.h` - InnoDB page format constants
- `libinnodb_zipdecompress.a` - MySQL/Percona static library with core decompression
- `Makefile` - Build configuration

## Building

```bash
make clean
make
```

This produces `libinnodb_decompress.so` which can be used from Go via CGo.

## Dependencies

- `libinnodb_zipdecompress.a` - From MySQL/Percona build
- `libz` - zlib compression
- `liblz4` - LZ4 compression  
- `libstdc++` - C++ standard library

## Current Status

⚠️ **ABI Mismatch Issue**: The library builds but decompression fails because the `page_zip_des_t` structure layout doesn't match the one expected by `libinnodb_zipdecompress.a`.

## Solution

To fix this, the library needs to be built against the actual MySQL/Percona headers that match the version used to create `libinnodb_zipdecompress.a`. See `../percona-parser/INNODB_DECOMPRESSION_LIBRARY.md` for detailed development plan.

## Usage from Go

```go
// In Go code
// #cgo CFLAGS: -I${SRCDIR}/lib
// #cgo LDFLAGS: -L${SRCDIR}/lib -linnodb_decompress -lstdc++ -lz -llz4
```

See `../decompress.go` for the complete Go wrapper implementation.
# InnoDB Page Parser

A Go library and command-line tool for parsing and analyzing InnoDB database pages. This tool allows you to inspect the internal structure of InnoDB data files (.ibd files) at the page level, providing insights into page headers, records, and metadata.

## Features

- **Page Structure Analysis**: Parse and inspect InnoDB page headers, trailers, and internal structures
- **Multiple Page Types**: Support for INDEX, ALLOCATED, UNDO_LOG, and SDI page types
- **Record Traversal**: Walk through records in index pages following the linked list structure
- **Flexible Output**: Text, JSON, or summary output formats
- **Library & CLI**: Use as a Go library or standalone command-line tool
- **Compact Format Support**: Full support for InnoDB compact record format

## Installation

### Prerequisites
- Go 1.20 or higher

### Building from Source

```bash
# Clone the repository
git clone https://github.com/wilhasse/go-innodb.git
cd go-innodb

# Build the library and CLI tool
make build

# Or build only the CLI tool
make build-tool

# Install to $GOPATH/bin
make install
```

## Usage

### Quick Start with Test Data

```bash
# Build the tool
make build

# Try with included test data
./go-innodb -file testdata/users/users.ibd -page 4 -records -v
```

### Command-Line Tool

The `go-innodb` command-line tool provides a user-friendly interface for analyzing InnoDB pages:

```bash
# Parse page 0 (root page) from a data file
./go-innodb -file /path/to/table.ibd

# Parse a specific page number
./go-innodb -file /path/to/table.ibd -page 3

# Show records in a page with actual data (verbose mode)
./go-innodb -file /path/to/table.ibd -page 3 -records -v

# Output in JSON format
./go-innodb -file /path/to/table.ibd -page 3 -format json

# Get a quick summary
./go-innodb -file /path/to/table.ibd -page 3 -format summary
```

#### Command-Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `-file` | Path to InnoDB data file (.ibd) | Required |
| `-page` | Page number to read | 0 |
| `-format` | Output format: text, json, or summary | text |
| `-records` | Show all records in the page | false |
| `-max-records` | Maximum records to display | 100 |
| `-v` | Verbose output with additional details | false |

### Using as a Go Library

```go
package main

import (
    "fmt"
    "os"
    goinnodb "github.com/wilhasse/go-innodb"
)

func main() {
    // Open an InnoDB data file
    file, err := os.Open("table.ibd")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    // Create a page reader
    reader := goinnodb.NewPageReader(file)

    // Read page 3
    page, err := reader.ReadPage(3)
    if err != nil {
        panic(err)
    }

    // Check page type
    fmt.Printf("Page type: %d\n", page.PageType())

    // Parse as index page if applicable
    if page.PageType() == goinnodb.PageTypeIndex {
        indexPage, err := goinnodb.ParseIndexPage(page)
        if err != nil {
            panic(err)
        }

        fmt.Printf("Number of records: %d\n", indexPage.Hdr.NumUserRecs)
        fmt.Printf("Page level: %d\n", indexPage.Hdr.PageLevel)
        fmt.Printf("Is leaf: %v\n", indexPage.IsLeaf())

        // Walk through records
        records, err := indexPage.WalkRecords(100, true)
        if err != nil {
            panic(err)
        }

        for i, rec := range records {
            fmt.Printf("Record %d: heap_no=%d, deleted=%v\n", 
                i, rec.Header.HeapNumber, rec.Header.FlagsDeleted)
        }
    }
}
```

## InnoDB Page Structure

### Page Layout (16KB)

```
+------------------+ 0
| FIL Header       | 38 bytes
+------------------+ 38
| Page Header      | 56 bytes (includes FSEG header)
+------------------+ 94
| Infimum Record   | System record
+------------------+
| Supremum Record  | System record
+------------------+
| User Records     | Variable size
| ...              |
+------------------+
| Free Space       |
+------------------+
| Page Directory   | Variable size (grows backwards)
+------------------+ 16376
| FIL Trailer      | 8 bytes
+------------------+ 16384
```

### Key Components

#### FIL Header (38 bytes)
- **Checksum**: Page checksum for integrity verification
- **Page Number**: Unique page identifier within the tablespace
- **Previous/Next**: Pointers for doubly-linked list of pages
- **LSN**: Log Sequence Number for recovery
- **Page Type**: Type of page (INDEX, UNDO_LOG, etc.)
- **Space ID**: Tablespace identifier

#### Index Header (36 bytes)
- **Number of Records**: Count of user records in the page
- **Page Level**: 0 for leaf pages, >0 for internal nodes
- **Index ID**: Unique identifier for the index
- **Format**: Compact or Redundant format indicator
- **Garbage Space**: Bytes occupied by deleted records

#### Records
- Stored as a linked list using relative offsets
- Each record has a 5-byte header in compact format
- INFIMUM and SUPREMUM are system records marking boundaries

## Development

### Project Structure

```
go-innodb/
├── cmd/
│   └── go-innodb/         # CLI tool source
│       └── main.go
├── docs/                  # Documentation
│   ├── API.md            # API documentation
│   ├── EXAMPLES.md       # Usage examples
│   └── INTERNALS.md      # InnoDB internals
├── testdata/              # Test data files
│   ├── users/            # Sample users table
│   └── README.md         # Test data documentation
├── *.go                   # Library source files (see ARCHITECTURE.md)
├── doc.go                # Package documentation
├── go.mod                # Go module definition
├── Makefile              # Build automation
├── ARCHITECTURE.md       # Code organization guide
├── CLAUDE.md             # AI assistant context
└── README.md             # This file
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for details on code organization and design decisions.

### Building and Testing

```bash
# Format code
make fmt

# Run static analysis
make vet

# Run linters
make lint

# Run tests
make test

# Run tests with coverage
make coverage

# Clean build artifacts
make clean

# Show all available targets
make help
```

### Core Types

- `InnerPage`: Base page structure with FIL header/trailer
- `IndexPage`: Parsed INDEX page with header and records
- `PageReader`: Utility for reading pages from files
- `GenericRecord`: Record structure with header and position
- `RecordHeader`: 5-byte compact record header

## Output Examples

### Text Format
```
=== Page 3 ===

FIL Header:
  Checksum:    0x12345678
  Page Number: 3
  Page Type:   INDEX (17855)
  Space ID:    1
  LSN:         123456789
  Prev Page:   2
  Next Page:   4

Index Header:
  Format:      COMPACT
  Records:     42 user records
  Page Level:  0 (leaf)
  Index ID:    140

Page Usage:  8432 / 16384 bytes (51.5%)
```

### JSON Format
```json
{
  "page_number": 3,
  "fil_header": {
    "checksum": 305419896,
    "page_number": 3,
    "page_type": 17855,
    "page_type_name": "INDEX",
    "space_id": 1,
    "lsn": 123456789
  },
  "index_page": {
    "format": 1,
    "format_name": "COMPACT",
    "user_records": 42,
    "page_level": 0,
    "is_leaf": true,
    "index_id": 140,
    "used_bytes": 8432
  }
}
```

## Limitations

- Only supports InnoDB compact record format (not redundant format)
- Read-only operations (no modification capabilities)
- Requires understanding of InnoDB internals for advanced analysis
- Does not decode actual record data (only headers and metadata)

## Contributing

Contributions are welcome! Please ensure:
1. Code follows Go conventions
2. All tests pass
3. Documentation is updated for new features

## License

This project is open source. Please check the license file for details.

## Acknowledgments

This project is inspired by and based on:
- [innodb-java-reader](https://github.com/alibaba/innodb-java-reader) - Alibaba's Java library for parsing InnoDB files

## References

- [Jeremy Cole's InnoDB Ruby Tools](https://github.com/jeremycole/innodb_ruby)
- [Nextgres OSS Embedded InnoDB](https://github.com/nextgres/oss-embedded-innodb)
- MySQL/MariaDB InnoDB source code

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/wilhasse/go-innodb).
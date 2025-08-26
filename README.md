# InnoDB Page Parser

A Go library and command-line tool for parsing InnoDB database pages (.ibd files) and extracting actual column data using table schemas.

## Features

- **Page Structure Analysis**: Parse InnoDB page headers, records, and metadata
- **Column Data Extraction**: Extract actual column values using CREATE TABLE schemas
- **Multiple Output Formats**: Text, JSON, or summary output
- **Compact Format Support**: Full support for InnoDB compact record format
- **Schema-Aware Parsing**: Parse records using table definitions from SQL files

## Installation

```bash
# Clone the repository
git clone https://github.com/wilhasse/go-innodb.git
cd go-innodb

# Build the library and CLI tool
make build

# Or install to $GOPATH/bin
make install
```

## Usage

### Quick Start

```bash
# Parse page with hex dump only (no schema)
./go-innodb -file testdata/users/users.ibd -page 4 -records

# Parse with schema to extract actual column values
./go-innodb -file testdata/users/users.ibd -page 4 -sql testdata/users/users.sql -parse -records
```

### Column Data Extraction (NEW!)

With a CREATE TABLE schema, the parser can extract actual column values:

```bash
# Provide the table schema via SQL file
./go-innodb -file data.ibd -page 4 -sql schema.sql -parse -records
```

Output with column parsing:
```
Records:
  #  id  name     email                created_at           
  0  1   Alice    alice@example.com    2023-10-31 02:24:56  
  1  2   Bob      bob@example.com      2022-03-27 13:24:08  
  2  3   Charlie  charlie@example.com  2023-10-31 02:24:56
```

### Command-Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `-file` | Path to InnoDB data file (.ibd) | Required |
| `-page` | Page number to read | 0 |
| `-sql` | Path to SQL file with CREATE TABLE | Optional |
| `-parse` | Parse column data using schema | false |
| `-records` | Show all records in the page | false |
| `-format` | Output format: text, json, summary | text |
| `-v` | Verbose output | false |

### Using as a Go Library

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
    tableDef, _ := schema.ParseTableDefFromSQLFile("users.sql")
    
    // Open InnoDB file
    file, _ := os.Open("users.ibd")
    defer file.Close()
    
    // Read page
    reader := innodb.NewPageReader(file)
    page, _ := reader.ReadPage(4)
    
    // Parse as index page
    indexPage, _ := innodb.ParseIndexPage(page)
    
    // Extract records with column data
    records, _ := indexPage.WalkRecordsWithSchema(tableDef, true)
    
    // Access column values
    for _, rec := range records {
        fmt.Printf("ID: %v, Name: %v\n", 
            rec.Values["id"], 
            rec.Values["name"])
    }
}
```

## Documentation

- **[InnoDB Page Parsing Guide](docs/INNODB_PAGE_PARSING.md)** - Complete parsing process and pitfalls
- **[Compact Format Details](docs/COMPACT_FORMAT_DETAILS.md)** - Binary layout specifications  
- **[Debugging Guide](docs/DEBUGGING_GUIDE.md)** - Troubleshooting common issues
- **[Architecture Overview](docs/ARCHITECTURE.md)** - Project structure and design

## Supported Data Types

Currently supported MySQL column types:
- Integer types: TINYINT, SMALLINT, INT, BIGINT (signed/unsigned)
- String types: CHAR, VARCHAR
- Date/Time types: DATE, DATETIME, TIMESTAMP, YEAR
- Work in progress: DECIMAL, FLOAT, DOUBLE, TEXT, BLOB

## Output Examples

### Without Schema (Hex Dump)
```
Record 0: InnerOffset=128, Type=CONVENTIONAL
  DATA (50 bytes): 80 00 00 01 00 00 00 01 ae b3 81 00 ...
```

### With Schema (Parsed Columns)
```
Records:
  #  id  name     email                created_at           
  0  1   Alice    alice@example.com    2023-10-31 02:24:56
```

## Development

```bash
# Format code
make fmt

# Run tests  
make test

# Run linters
make lint

# Show all targets
make help
```

## Key Implementation Details

The parser handles several InnoDB format complexities:
- **Variable-length headers** are stored in reverse column order
- **Transaction fields** (13 bytes) are placed after primary key columns
- **Signed integers** use XOR transformation with the sign bit
- **NULL bitmap** calculation: (nullable_columns + 7) / 8 bytes

See the [documentation](docs/) for detailed technical information.

## Contributing

Contributions are welcome! Please ensure:
1. Code follows Go conventions
2. All tests pass
3. Documentation is updated for new features

## Acknowledgments

Based on:
- [innodb-java-reader](https://github.com/alibaba/innodb-java-reader) - Alibaba's Java implementation
- [Jeremy Cole's InnoDB Internals](https://blog.jcole.us/innodb/)

## License

This project is open source. Please check the license file for details.

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/wilhasse/go-innodb).
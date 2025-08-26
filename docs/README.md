# go-innodb Documentation

Complete documentation for parsing InnoDB database pages and extracting column data.

## Documentation Index

### Core Documentation

1. **[Installation Guide](INSTALLATION.md)** 📍 **NEW**  
   Complete installation instructions including:
   - Basic setup and dependencies
   - Compressed page support setup
   - Runtime library configuration
   - Platform-specific notes and troubleshooting

2. **[Architecture Overview](ARCHITECTURE.md)**  
   High-level design and structure of the go-innodb library

3. **[InnoDB Page Parsing](INNODB_PAGE_PARSING.md)** 📍 **NEW**  
   Complete guide to parsing InnoDB pages, including:
   - Page structure breakdown
   - Variable-length header handling  
   - Transaction field placement
   - Common pitfalls and solutions

4. **[Compact Format Details](COMPACT_FORMAT_DETAILS.md)** 📍 **NEW**  
   Technical specification of the compact record format:
   - Exact binary layout
   - Byte-level encoding details
   - Parsing algorithm
   - Test cases and examples

5. **[Debugging Guide](DEBUGGING_GUIDE.md)** 📍 **NEW**  
   Practical debugging techniques:
   - Symptom-based troubleshooting
   - Hex dump analysis
   - Common patterns recognition
   - Debug output templates

## Quick Start

### Basic Usage

```bash
# Parse a page and show records (hex dump only)
./go-innodb -file data.ibd -page 4 -records

# Parse with schema to show actual column values
./go-innodb -file data.ibd -page 4 -sql schema.sql -parse -records
```

### Understanding the Output

Without schema (`-sql`):
```
Record 0: InnerOffset=128, Type=CONVENTIONAL
  DATA (50 bytes): 80 00 00 01 00 00 00 01 ae b3 81 00 ...
```

With schema (`-sql schema.sql -parse`):
```
Records:
  #  id  name     email                created_at           
  0  1   Alice    alice@example.com    2023-10-31 02:24:56  
  1  2   Bob      bob@example.com      2022-03-27 13:24:08
```

## Key Concepts

### What Makes InnoDB Parsing Challenging?

1. **Variable-length headers are stored in reverse order** - The most common source of bugs
2. **Transaction fields are in the middle** - Not at the end as documentation suggests  
3. **Signed integers use XOR transformation** - Must flip the sign bit
4. **Big-endian encoding** - All multi-byte values

### The Parsing Process

```
1. Read Page (16KB) → 2. Parse Index Header → 3. Walk Records → 4. Extract Columns
                                                                          ↑
                                                              Requires Table Schema
```

## Learning Path

1. Start with [ARCHITECTURE.md](ARCHITECTURE.md) for overview
2. Read [INNODB_PAGE_PARSING.md](INNODB_PAGE_PARSING.md) for the complete process
3. Refer to [COMPACT_FORMAT_DETAILS.md](COMPACT_FORMAT_DETAILS.md) for binary specifications
4. Use [DEBUGGING_GUIDE.md](DEBUGGING_GUIDE.md) when things go wrong

## Common Issues and Solutions

| Problem | Solution | Documentation |
|---------|----------|---------------|
| Garbled string data | Fix variable header reading order | [INNODB_PAGE_PARSING.md#variable-length-headers](INNODB_PAGE_PARSING.md#variable-length-headers) |
| Binary data in strings | Skip 13-byte transaction fields after PK | [INNODB_PAGE_PARSING.md#transaction-fields](INNODB_PAGE_PARSING.md#transaction-fields) |
| Negative primary keys | Apply XOR transformation | [COMPACT_FORMAT_DETAILS.md#integer-types](COMPACT_FORMAT_DETAILS.md#integer-types) |
| Concatenated columns | Check variable length array order | [DEBUGGING_GUIDE.md#symptom-column-values-concatenated](DEBUGGING_GUIDE.md#symptom-column-values-concatenated) |

## Implementation Status

✅ **Completed:**
- FIL header/trailer parsing
- INDEX page structure  
- Record iteration via linked list
- Compact record format parsing
- Schema parsing from SQL
- Column value extraction (INT, VARCHAR, TIMESTAMP, DATE)
- NULL bitmap handling
- Variable-length header parsing (fixed!)
- Transaction field handling
- Compressed page support (ROW_FORMAT=COMPRESSED with KEY_BLOCK_SIZE 1K/2K/4K/8K)

🚧 **TODO:**
- Overflow page support
- More data types (DECIMAL, BLOB, JSON)
- Secondary index parsing
- Recovery from corrupted pages
- Transparent page compression support

## Contributing

When debugging or extending the parser:

1. Always test with simple, known data first
2. Use hex dumps to verify your assumptions
3. Compare with the Java reference implementation
4. Document any format discoveries

## References

- [MySQL Internals: InnoDB Page Structure](https://dev.mysql.com/doc/internals/en/innodb-page-structure.html) (has some inaccuracies)
- [Jeremy Cole's InnoDB Internals](https://blog.jcole.us/innodb/)
- [Alibaba's innodb-java-reader](https://github.com/alibaba/innodb-java-reader)

---
*This documentation was created during the debugging journey of fixing the go-innodb parser. It represents hard-won knowledge about InnoDB's actual binary format.*
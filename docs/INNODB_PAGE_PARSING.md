# InnoDB Page Parsing: A Deep Dive

This document describes the complete process of parsing InnoDB pages and extracting actual column data from records. This is based on our implementation journey and the bugs we discovered and fixed.

## Table of Contents
- [Overview](#overview)
- [Page Structure](#page-structure)
- [Compact Record Format](#compact-record-format)
- [Variable-Length Headers](#variable-length-headers)
- [Transaction Fields](#transaction-fields)
- [Parsing Process](#parsing-process)
- [Common Pitfalls](#common-pitfalls)
- [Debugging Tips](#debugging-tips)

## Overview

InnoDB stores data in 16KB pages (16384 bytes). Each page contains records in a B+ tree structure. To extract actual column values from these pages, you need:

1. **Table Schema**: The CREATE TABLE definition to know column types and order
2. **Page Parser**: To read the page structure and locate records
3. **Record Parser**: To extract column values from compact format records

## Page Structure

```
┌─────────────────────────┐ Offset 0
│     FIL Header          │ 38 bytes
├─────────────────────────┤ 
│     Page Header         │ 56 bytes (includes FSEG)
├─────────────────────────┤
│     INFIMUM Record      │ System record
├─────────────────────────┤
│     SUPREMUM Record     │ System record  
├─────────────────────────┤
│     User Records        │ Linked list of actual data
│     ...                 │
├─────────────────────────┤
│     Free Space          │ 
├─────────────────────────┤
│     Page Directory      │ Slots for quick access
├─────────────────────────┤ Offset 16376
│     FIL Trailer         │ 8 bytes
└─────────────────────────┘ Offset 16384
```

### Key Points:
- All multi-byte values use **big-endian** encoding
- Records are connected via relative offsets in their headers
- Page type 17855 (0x45BF) indicates an INDEX page

## Compact Record Format

The compact record format is the modern InnoDB format (vs. the older redundant format). Here's the actual layout:

```
[Variable-length headers] [NULL bitmap] [5-byte record header] [Record data]
     ← Read direction           →                     Reading position →
```

### Detailed Structure:

1. **Variable-length headers** (stored before record header)
   - 1 or 2 bytes per variable-length column
   - Stored in REVERSE column order
   - Read backwards from the record header

2. **NULL bitmap** (only for tables with nullable columns)
   - Size: (nullable_columns + 7) / 8 bytes
   - Each bit indicates if a column is NULL

3. **Record header** (5 bytes)
   - Contains record type, next record offset, etc.
   - This is where `recordPos - 5` points to

4. **Record data** (actual column values)
   - Stored at `recordPos`
   - Primary key columns first
   - Then transaction fields (for clustered index)
   - Then remaining columns

## Variable-Length Headers

This was the trickiest part to get right! Here's how they actually work:

### Storage Order (CRITICAL!)

Variable-length headers are stored in **reverse column order**:

```
Memory layout (addresses increasing →):
[header_for_last_col][header_for_second_last]...[header_for_first_col][NULL bitmap][record header]
```

### Reading Process

When reading backwards from the record header position:
1. The RIGHTMOST header (first one you encounter) is for the FIRST variable column
2. The next header is for the second variable column
3. And so on...

### Example:
For columns `[name VARCHAR, email VARCHAR]`:
- Memory: `[email_length][name_length]` 
- But when reading right-to-left, you get: name_length first, then email_length
- So you need to iterate columns forward while reading memory backward!

### Code Solution:
```go
// Correct approach:
for i := 0; i < len(varColumns); i++ {  // Forward through columns
    col := varColumns[i]
    varHeaderPos--  // Read backward through memory
    length := pageData[varHeaderPos]
    varLengths = append(varLengths, length)  // Append to maintain order
}
```

## Transaction Fields

In clustered index leaf pages, InnoDB stores transaction metadata:
- **Transaction ID** (6 bytes): Last transaction that modified this row
- **Roll Pointer** (7 bytes): Points to undo log for MVCC

### Placement (IMPORTANT!)

Despite what some documentation suggests, transaction fields are placed:
- **AFTER** the primary key columns
- **BEFORE** other columns
- Total: 13 bytes to skip

```
[Primary Key Columns] [6-byte trx_id] [7-byte roll_ptr] [Other Columns]
```

## Parsing Process

Here's the complete parsing flow:

1. **Load Table Schema**
   ```go
   tableDef := ParseTableDefFromSQL(sqlFile)
   ```

2. **Read Page**
   ```go
   page := ReadPage(file, pageNum)
   indexPage := ParseIndexPage(page)
   ```

3. **Walk Records**
   ```go
   records := WalkRecords(indexPage, tableDef, true) // isLeafPage = true
   ```

4. **For Each Record:**
   - Calculate header position: `recordPos - 5`
   - Parse NULL bitmap if present
   - Read variable-length headers (backwards, but matching forward column order)
   - Parse primary key columns
   - Skip 13 bytes for transaction fields
   - Parse remaining columns

## Common Pitfalls

### 1. Variable Header Order Confusion
**Wrong assumption**: Headers are in the same order as columns
**Reality**: Headers are in reverse order, but the rightmost corresponds to the first column

### 2. Transaction Field Placement
**Wrong assumption**: Transaction fields are at the end of the record
**Reality**: They're after the primary key, before other columns

### 3. Signed Integer Storage
**Important**: Signed integers use XOR transformation
```go
// For signed INT:
storedValue := actualValue ^ 0x80000000
// To read:
actualValue := storedValue ^ 0x80000000
```

### 4. String Length Calculation
VARCHAR can use 1 or 2 byte length headers depending on max length and actual length.

## Debugging Tips

### 1. Hex Dump Analysis
When debugging, always look at the actual hex data:
```bash
# Example for record starting at position 128:
80 00 00 01  # Primary key (id=1 with XOR)
00 00 00 01 ae b3 81 00 00 00 8e 01 10  # Transaction fields (13 bytes)
41 6c 69 63 65  # "Alice"
```

### 2. Variable Length Verification
Add debug output to verify lengths:
```go
fmt.Printf("Column %s: expected %d bytes, got %d\n", 
    col.Name, len("Alice"), varLength)
```

### 3. Common Patterns to Check
- Primary keys often start with `80 00 00 XX` (XOR'd integers)
- Transaction ID is often `00 00 00 01 ae b3` or similar
- Strings should be readable ASCII/UTF-8

### 4. Testing with Known Data
Always test with simple, known data first:
```sql
INSERT INTO users VALUES (1, 'Alice', 'alice@example.com');
INSERT INTO users VALUES (2, 'Bob', 'bob@example.com');
```

## Implementation Checklist

When implementing an InnoDB parser:

- [ ] Parse FIL header and verify checksums
- [ ] Parse INDEX page header
- [ ] Implement record iteration via linked list
- [ ] Parse CREATE TABLE for schema
- [ ] Calculate NULL bitmap size correctly
- [ ] **Read variable headers in correct order** (most common bug!)
- [ ] Handle transaction fields placement
- [ ] Support both 1-byte and 2-byte variable lengths
- [ ] Handle XOR transformation for signed integers
- [ ] Test with various data types

## Example Output

Correct parsing should produce:
```
Records:
  #  id  name     email                created_at           
  0  1   Alice    alice@example.com    2023-10-31 02:24:56  
  1  2   Bob      bob@example.com      2022-03-27 13:24:08  
  2  3   Charlie  charlie@example.com  2023-10-31 02:24:56
```

Not:
```
# Wrong variable header order:
0  1   Alicealice@exampl    e.com    2023-10-31 02:24:56

# Or missing transaction field skip:
0  1      ���   �Alic    ealic    2023-10-31 02:24:56
```

## References

- InnoDB source code: `storage/innobase/rem/rem0rec.cc`
- MySQL Internals Manual (though it has some inaccuracies)
- Jeremy Cole's InnoDB Ruby implementation
- Alibaba's innodb-java-reader (our reference implementation)

## Contributors

This documentation is based on the debugging session where we fixed the go-innodb parser. Special thanks to the engineer who provided the initial diff (even though it had some incorrect assumptions about transaction field placement).

---
*Last Updated: After fixing the variable-length header ordering bug*
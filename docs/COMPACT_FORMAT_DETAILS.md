# InnoDB Compact Record Format: Technical Details

## Binary Layout Specification

This document provides the exact binary layout of InnoDB's compact record format, including byte offsets and encoding details.

## Record Structure Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                          BEFORE recordPos                         │
├────────────────┬──────────────┬───────────────┬──────────────────┤
│ Var-len headers│  NULL bitmap  │ Record Header │   Record Data    │
│ (reverse order)│  (if needed)  │   (5 bytes)   │   (at recordPos) │
└────────────────┴──────────────┴───────────────┴──────────────────┘
                                  ↑
                            recordPos - 5
```

## Detailed Component Breakdown

### 1. Variable-Length Headers

**Location**: Before NULL bitmap, read backwards from record header

**Size**: 
- 1 byte for lengths ≤ 127
- 2 bytes for lengths > 127 (for columns that support it)

**Encoding for 1-byte**:
```
Bit 7: Always 0
Bits 6-0: Length (0-127)
```

**Encoding for 2-byte**:
```
First byte (read second):
  Bit 7: Always 1 (indicates 2-byte length)
  Bit 6: Overflow flag (1 = data on overflow page)
  Bits 5-0: High 6 bits of length

Second byte (read first):
  Bits 7-0: Low 8 bits of length

Total length = ((first_byte & 0x3F) << 8) | second_byte
```

**Order** (CRITICAL):
```
Columns: [col1_varchar, col2_varchar, col3_varchar]
Memory:  [len_col3][len_col2][len_col1][null_bitmap][header]
         ← Read this direction
```

### 2. NULL Bitmap

**Location**: Immediately before record header

**Size**: `(nullable_column_count + 7) / 8` bytes

**Encoding**:
- Bit 0 of byte 0: First nullable column
- Bit 1 of byte 0: Second nullable column
- ...
- Bit is 1 if column is NULL, 0 if not NULL

**Example** for 3 nullable columns:
```
Byte 0: [bit7][bit6][bit5][bit4][bit3][col3][col2][col1]
```

### 3. Record Header (5 bytes)

**Location**: At `recordPos - 5`

**Structure**:
```
Byte 0-1: Info bits and number of records owned
  Bits 15-10: Unused (reserved)
  Bit 9: Deleted flag
  Bit 8: Min_rec flag
  Bits 7-0: Number of records owned

Byte 2-3: Order in heap
  Bits 15-13: Record type (000 = conventional, 001 = node pointer, etc.)
  Bits 12-0: Heap number

Byte 4: Next record offset (high byte)
Byte 5: Next record offset (low byte)
  - 16-bit signed offset to next record in page
  - Measured from current record start
```

### 4. Record Data Layout

**For Clustered Index Leaf Pages**:

```
[Primary Key Columns]
[Transaction ID - 6 bytes]
[Roll Pointer - 7 bytes]
[Remaining Columns in Table Definition Order]
```

**For Secondary Index Leaf Pages**:
```
[Secondary Key Columns]
[Primary Key Columns]
```

**For Non-Leaf (Node Pointer) Pages**:
```
[Key Columns]
[Child Page Number - 4 bytes]
```

## Data Type Encodings

### Integer Types

**Signed integers** use XOR transformation with sign bit:
- INT (4 bytes): `stored = value ^ 0x80000000`
- BIGINT (8 bytes): `stored = value ^ 0x8000000000000000`
- SMALLINT (2 bytes): `stored = value ^ 0x8000`
- TINYINT (1 byte): `stored = value ^ 0x80`

**Unsigned integers** stored as-is in big-endian

### Variable-Length Strings (VARCHAR)

- Length stored in variable-length header
- Actual data follows in record data section
- UTF-8 encoded for utf8mb4 charset

### Fixed-Length Types

- CHAR: Padded with spaces to full length
- DATE: 3 bytes (YYYY*16*32 + MM*32 + DD)
- TIMESTAMP: 4 bytes (Unix timestamp)
- DATETIME: 8 bytes

## Parsing Algorithm

```python
def parse_compact_record(page_data, record_pos, table_def):
    # 1. Read record header
    header_pos = record_pos - 5
    header = read_record_header(page_data, header_pos)
    
    # 2. Calculate NULL bitmap size and position
    null_bitmap_size = (table_def.nullable_count + 7) // 8
    null_bitmap_pos = header_pos - null_bitmap_size
    null_bitmap = read_null_bitmap(page_data, null_bitmap_pos, null_bitmap_size)
    
    # 3. Read variable-length headers
    var_lengths = []
    var_header_pos = null_bitmap_pos
    
    # CRITICAL: Iterate forward through columns
    for col in table_def.variable_columns:
        var_header_pos -= 1  # Move backward in memory
        length = page_data[var_header_pos]
        
        if needs_two_bytes(col, length):
            var_header_pos -= 1
            length = decode_two_byte_length(page_data[var_header_pos:var_header_pos+2])
        
        var_lengths.append(length)
    
    # 4. Parse column data
    data_pos = record_pos
    values = {}
    var_idx = 0
    
    # Parse primary key
    for col in table_def.primary_key_columns:
        if is_null(col, null_bitmap):
            values[col.name] = None
        else:
            if col.is_variable:
                length = var_lengths[var_idx]
                var_idx += 1
                values[col.name] = read_bytes(page_data, data_pos, length)
                data_pos += length
            else:
                values[col.name] = read_fixed(page_data, data_pos, col.size)
                data_pos += col.size
    
    # Skip transaction fields (leaf pages only)
    if is_leaf_page:
        data_pos += 13  # 6 bytes trx_id + 7 bytes roll_ptr
    
    # Parse remaining columns
    for col in table_def.non_pk_columns:
        # ... similar to above
    
    return values
```

## Common Binary Patterns

### Identifying Record Types

```
INFIMUM: 69 6e 66 69 6d 75 6d 00  ("infimum\0")
SUPREMUM: 73 75 70 72 65 6d 75 6d  ("supremum")
```

### Transaction Fields Pattern
```
00 00 00 01 ae b3  # Transaction ID (6 bytes)
81 00 00 00 8e 01 10  # Roll Pointer (7 bytes)
```

### Signed INT Primary Keys
```
80 00 00 01  # ID = 1 (with XOR)
80 00 00 02  # ID = 2
80 00 00 03  # ID = 3
```

## Debugging Checklist

When records show corrupted data:

1. **Check variable header order**: Most common issue!
2. **Verify NULL bitmap size**: `(nullable_cols + 7) / 8`
3. **Check transaction field skip**: 13 bytes after PK
4. **Verify XOR for signed integers**: Especially primary keys
5. **Check charset multipliers**: UTF8MB4 = up to 4 bytes per char

## Test Cases

### Minimal Test Table
```sql
CREATE TABLE test (
    id INT PRIMARY KEY,
    name VARCHAR(50)
) ENGINE=InnoDB;

INSERT INTO test VALUES (1, 'Alice');
```

Expected binary for record:
```
Position  | Hex           | Description
----------|---------------|-------------
-6        | 05            | Var header: length('Alice') = 5
-5 to -1  | [rec header]  | 5-byte record header
0-3       | 80 00 00 01   | id = 1 (XOR'd)
4-9       | [trx_id]      | 6 bytes
10-16     | [roll_ptr]    | 7 bytes  
17-21     | 41 6c 69 63 65| 'Alice'
```

## Edge Cases

1. **NULL columns**: Check bit in bitmap, skip in data
2. **Empty strings**: Length = 0 in header, no bytes in data
3. **Large VARCHAR**: 2-byte length header
4. **Overflow pages**: Bit 6 set in length header (not fully supported)

---
*This document represents the actual InnoDB compact format as discovered through implementation and debugging.*
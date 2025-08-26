# Debugging InnoDB Parser Issues

A practical guide for debugging common problems when parsing InnoDB pages.

## Quick Diagnostic Commands

```bash
# Test basic parsing
./go-innodb -file users.ibd -page 4 -records

# Test with schema parsing
./go-innodb -file users.ibd -page 4 -sql users.sql -parse -records

# Show hex dump of a page
./go-innodb -file users.ibd -page 4 -hex
```

## Symptom-Based Troubleshooting

### Symptom: Garbled String Data

**Example**: `���   �Alic    ealic` instead of `Alice    alice@example.com`

**Cause**: Variable-length headers being read in wrong order

**Debug Steps**:
1. Add debug output for variable lengths:
```go
fmt.Printf("Column %s: varLen=%d\n", col.Name, varLen)
```

2. Check the hex dump at record position:
```go
fmt.Printf("Data at %d: % x\n", dataPos, pageData[dataPos:dataPos+20])
```

3. Verify header reading direction:
- Headers are in REVERSE order in memory
- But rightmost header = FIRST column!

**Fix**: Ensure forward iteration through columns while reading backward through memory

### Symptom: Column Values Concatenated

**Example**: `Alicealice@exampl    e.com` 

**Cause**: Variable lengths array has wrong values

**Debug Steps**:
1. Print variable lengths array:
```go
fmt.Printf("varLengths: %v\n", varLengths)
```

2. Compare with actual data:
- "Alice" = 5 bytes
- "alice@example.com" = 17 bytes
- If you see [17, 5], they're swapped!

### Symptom: Binary Data in Middle of Strings

**Example**: `Alice����������Bob`

**Cause**: Not skipping transaction fields (13 bytes)

**Debug Steps**:
1. Check position after primary key:
```go
fmt.Printf("After PK, pos=%d, next 13 bytes: % x\n", 
    dataPos, pageData[dataPos:dataPos+13])
```

2. Look for pattern like:
```
00 00 00 01 ae b3  # Transaction ID
81 00 00 00 8e 01 10  # Roll pointer
```

**Fix**: Add 13-byte skip after primary key columns in leaf pages

### Symptom: Negative Primary Key Values

**Example**: ID showing as -2147483647 instead of 1

**Cause**: Not handling XOR transformation for signed integers

**Debug Steps**:
1. Check raw bytes:
```go
// For ID=1, raw bytes should be: 80 00 00 01
```

2. Apply XOR:
```go
value := binary.BigEndian.Uint32(bytes) ^ 0x80000000
```

### Symptom: Wrong Timestamp Values

**Example**: Dates showing year 5000+ or negative

**Debug Steps**:
1. Check byte length (should be 4 for TIMESTAMP, 8 for DATETIME)
2. Verify endianness (should be big-endian)
3. For TIMESTAMP, it's Unix timestamp in seconds

## Advanced Debugging Techniques

### 1. Binary Comparison with Known Good Parser

Use the Java implementation as reference:
```bash
# Run Java parser
./demo-java-innodb-reader.sh

# Run Go parser
./go-innodb -file testdata/users/users.ibd -page 4 -sql testdata/users/users.sql -parse -records

# Compare outputs
```

### 2. Hex Dump Analysis

Create a hex dump visualization:
```go
func debugRecord(pageData []byte, recordPos int) {
    fmt.Println("=== RECORD DEBUG ===")
    
    // Show 5 bytes before (record header)
    fmt.Printf("Header (-5 to -1): % x\n", 
        pageData[recordPos-5:recordPos])
    
    // Show first 50 bytes of record
    fmt.Printf("Data (0-50): % x\n", 
        pageData[recordPos:recordPos+50])
    
    // Try to show as string
    fmt.Printf("As string: %q\n", 
        pageData[recordPos:recordPos+50])
}
```

### 3. Step-by-Step Parser Trace

Add trace points at each parsing step:
```go
type ParserTrace struct {
    Step           string
    Position       int
    BytesRead      []byte
    Interpretation string
}

var traces []ParserTrace

// In parser:
traces = append(traces, ParserTrace{
    Step:           "Read NULL bitmap",
    Position:       nullBitmapPos,
    BytesRead:      nullBytes,
    Interpretation: fmt.Sprintf("Nulls: %v", nullBitmap),
})
```

### 4. Validate Against Schema

```go
func validateParsedRecord(record map[string]interface{}, tableDef *TableDef) {
    for _, col := range tableDef.Columns {
        val, exists := record[col.Name]
        if !exists && !col.Nullable {
            log.Printf("ERROR: Missing non-nullable column %s", col.Name)
        }
        
        if val != nil {
            switch col.Type {
            case "VARCHAR":
                if str, ok := val.(string); ok {
                    if len(str) > col.Length {
                        log.Printf("ERROR: %s too long: %d > %d", 
                            col.Name, len(str), col.Length)
                    }
                }
            }
        }
    }
}
```

## Common Hex Patterns

Learn to recognize these patterns in hex dumps:

```
# Primary key INT (1,2,3...)
80 00 00 01  # 1
80 00 00 02  # 2
80 00 00 03  # 3

# Transaction fields (usually similar across records)
00 00 00 01 ae b3 81 00 00 00 8e 01 10

# ASCII strings
41 6c 69 63 65  # "Alice"
42 6f 62        # "Bob"

# Variable length headers (small values)
05  # Length 5
11  # Length 17
```

## Testing Strategy

### 1. Start Simple
```sql
CREATE TABLE simple (
    id INT PRIMARY KEY,
    name VARCHAR(10)
);
INSERT INTO simple VALUES (1, 'Test');
```

### 2. Add Complexity Gradually
- Add nullable column
- Add multiple varchar columns  
- Add different data types
- Add larger data

### 3. Create Known Test Cases
```sql
-- Predictable data for debugging
INSERT INTO test VALUES 
    (1, 'AAAAA', 'BBBBB'),  -- Same length strings
    (2, 'CC', 'DDDDDDDDD'),  -- Different lengths
    (3, NULL, 'EEEEE');      -- NULL handling
```

## Debug Output Template

```go
func (p *CompactParser) debugParseRecord(pageData []byte, recordPos int) {
    fmt.Printf(`
=== PARSING RECORD at position %d ===
1. Record Header (5 bytes before): % x
2. NULL Bitmap Size: %d bytes
3. NULL Bitmap Bytes: % x
4. Variable Headers Count: %d
5. Variable Lengths: %v
6. Data Section First 30 bytes: % x
   As String: %q

Column Parsing:
`, recordPos, 
   pageData[recordPos-5:recordPos],
   p.tableDef.NullBitmapSize(),
   nullBitmapBytes,
   len(p.tableDef.VariableLengthColumns()),
   varLengths,
   pageData[recordPos:recordPos+30],
   pageData[recordPos:recordPos+30])
   
   // Then for each column...
}
```

## Error Recovery

When encountering parse errors:

1. **Don't panic** - Return error with context:
```go
return nil, fmt.Errorf("parse column %s at pos %d: %w", 
    col.Name, dataPos, err)
```

2. **Log intermediate state**:
```go
if err != nil {
    log.Printf("Parse failed. State: pos=%d, varLengths=%v, parsed=%d cols",
        dataPos, varLengths, len(record.Values))
}
```

3. **Provide hex context**:
```go
if err != nil {
    fmt.Printf("Error context - bytes at failure point: % x\n",
        pageData[dataPos:min(dataPos+20, len(pageData))])
}
```

## Performance Debugging

If parser is slow:

1. **Profile allocations**:
```go
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

2. **Reuse buffers**:
```go
type CompactParser struct {
    varLengthBuf []int  // Reuse instead of allocating
}
```

3. **Batch operations**:
- Read all variable headers at once
- Parse NULL bitmap in one operation

---
*Remember: Most parsing issues come from incorrect assumptions about data layout, not bugs in reading bytes!*
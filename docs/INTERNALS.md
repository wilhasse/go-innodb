# InnoDB Internals Documentation

This document explains the internal structure of InnoDB pages and how this library parses them.

## InnoDB Page Architecture

### Page Size and Organization

InnoDB uses a fixed page size of 16KB (16384 bytes). Each page is a self-contained unit with headers, data, and trailers that ensure data integrity and enable efficient navigation.

```
┌─────────────────────────────┐ Offset 0
│      FIL Header (38B)       │
├─────────────────────────────┤ Offset 38
│    Page Header (36B)        │
├─────────────────────────────┤ Offset 74
│    FSEG Header (20B)        │
├─────────────────────────────┤ Offset 94
│    System Records           │
│  - INFIMUM (13B)           │
│  - SUPREMUM (13B)          │
├─────────────────────────────┤ Offset 120
│                             │
│      User Records           │
│    (Variable Length)        │
│                             │
├─────────────────────────────┤
│                             │
│      Free Space             │
│                             │
├─────────────────────────────┤
│    Page Directory           │
│  (Grows Backwards)          │
├─────────────────────────────┤ Offset 16376
│    FIL Trailer (8B)         │
└─────────────────────────────┘ Offset 16384
```

## FIL Header Structure (38 bytes)

The FIL (File) header appears at the beginning of every InnoDB page:

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0 | 4 | Checksum | CRC32 or other checksum of page contents |
| 4 | 4 | Page Number | Offset of this page within the tablespace |
| 8 | 4 | Previous Page | Page number of previous page in list (0xFFFFFFFF if none) |
| 12 | 4 | Next Page | Page number of next page in list (0xFFFFFFFF if none) |
| 16 | 8 | LSN | Log Sequence Number of last modification |
| 24 | 2 | Page Type | Type identifier (INDEX=17855, UNDO_LOG=2, etc.) |
| 26 | 8 | Flush LSN | LSN at the time of last flush |
| 34 | 4 | Space ID | Tablespace identifier |

### Page Types

- `0x0000` (0): Freshly allocated page
- `0x45BD` (17855): B-tree index page
- `0x0002` (2): Undo log page
- `0x0003` (3): Inode page
- `0x0004` (4): Insert buffer free list
- `0x0005` (5): Insert buffer bitmap
- `0x0006` (6): System page
- `0x0007` (7): Transaction system page
- `0x0008` (8): File space header
- `0x45BF` (17853): SDI (Serialized Dictionary Information)

## Index Page Header (36 bytes)

For INDEX pages, immediately after the FIL header:

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 38 | 2 | Number of Directory Slots | Count of slots in page directory |
| 40 | 2 | Heap Top | Byte offset of free space start |
| 42 | 2 | Heap Records | Number of records in heap (high bit = format flag) |
| 44 | 2 | First Garbage | Offset to first deleted record |
| 46 | 2 | Garbage Space | Total bytes in deleted records |
| 48 | 2 | Last Insert | Offset of last inserted record |
| 50 | 2 | Direction | Direction of consecutive inserts |
| 52 | 2 | N Direction | Number of consecutive inserts in direction |
| 54 | 2 | User Records | Number of user records |
| 56 | 8 | Max Transaction ID | Maximum transaction ID on page |
| 64 | 2 | Page Level | Level in B-tree (0 = leaf) |
| 66 | 8 | Index ID | Identifier of the index |

### Format Flag

The high bit of the Heap Records field indicates the record format:
- Bit set (0x8000): Compact format
- Bit clear: Redundant format

## FSEG Header (20 bytes)

File segment header, used primarily by the root page:

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 74 | 4 | Leaf Inode Space | Space ID for leaf pages |
| 78 | 4 | Leaf Inode Page | Page number of leaf inode |
| 82 | 2 | Leaf Inode Offset | Byte offset of leaf inode |
| 84 | 4 | Non-leaf Inode Space | Space ID for internal pages |
| 88 | 4 | Non-leaf Inode Page | Page number of internal inode |
| 92 | 2 | Non-leaf Inode Offset | Byte offset of internal inode |

## Record Structure

### Compact Record Format

Each record consists of:
1. **Record Header** (5 bytes)
2. **Variable-length fields** (if any)
3. **Fixed-length fields**

#### Record Header Layout (5 bytes)

```
Byte 0: [4 bits: flags] [4 bits: n_owned]
  - Bit 0: min_rec flag
  - Bit 1: deleted flag
  - Bits 2-3: Reserved
  - Bits 4-7: Number of records owned

Bytes 1-2: [13 bits: heap_no] [3 bits: type]
  - Bits 0-12: Heap number (record position)
  - Bits 13-15: Record type

Bytes 3-4: Next record offset (signed 16-bit)
  - Relative offset from current record to next
```

### System Records

Every index page contains two system records:

1. **INFIMUM**: 
   - Always the first record
   - Contains "infimum\0" (8 bytes)
   - Points to first user record
   - Total size: 5 (header) + 8 (data) = 13 bytes

2. **SUPREMUM**:
   - Always the second record  
   - Contains "supremum" (8 bytes)
   - Last record in chain (next = 0)
   - Total size: 5 (header) + 8 (data) = 13 bytes

### Record Linked List

Records form a singly-linked list in ascending key order:
```
INFIMUM → Record1 → Record2 → ... → RecordN → SUPREMUM
```

The `next_record` field in each header contains the relative offset (in bytes) from the current record's data start to the next record's data start.

## Page Directory

The page directory provides fast binary search access to records:

- Located at the end of the page (before FIL trailer)
- Grows backwards from offset 16376
- Each slot is 2 bytes (big-endian)
- Points to the "largest" record in a group
- Groups contain 4-8 records (except first/last)

### Directory Structure
```
┌─────────────┐ 16376 - (n_slots * 2)
│   Slot 0    │ → Points to INFIMUM
├─────────────┤
│   Slot 1    │ → Points to a user record
├─────────────┤
│     ...     │
├─────────────┤
│   Slot n-1  │ → Points to SUPREMUM
└─────────────┘ 16376
```

## FIL Trailer (8 bytes)

Located at the very end of each page:

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 16376 | 4 | Checksum | Old-style checksum |
| 16380 | 4 | Low 32 LSN | Low 32 bits of page LSN |

The Low 32 LSN must match the low 32 bits of the LSN in the FIL header for page integrity.

## B-tree Structure

### Page Levels

- **Level 0**: Leaf pages containing actual data records
- **Level 1+**: Internal (non-leaf) pages containing only keys and child page pointers

### Root Page

The root page is identified by:
- Having both `prev` and `next` pointers as NULL (0xFFFFFFFF)
- Usually located at page 3 for user tables
- Contains FSEG header information for the entire index

### Page Splitting and Merging

When a page becomes full:
1. A new page is allocated
2. Records are distributed between old and new pages
3. Parent page is updated with new child pointer
4. Previous/next pointers maintain leaf page chain

## Checksums and Validation

### Checksum Types

InnoDB supports multiple checksum algorithms:
- CRC32C (default in modern versions)
- InnoDB checksum
- None (for temporary tables)

### LSN Validation

The library validates page integrity by checking:
1. Low 32 bits of FIL header LSN matches FIL trailer Low32 LSN
2. Page number in FIL header matches expected page number
3. Checksum validation (when implemented)

## Data Encoding

### Byte Order

All multi-byte values in InnoDB pages use **big-endian** (network) byte order.

### Variable-Length Encoding

Some fields use variable-length encoding:
- Small values: 1 byte
- Medium values: 2 bytes  
- Large values: 3-4 bytes

## Garbage Collection

Deleted records are not immediately removed:
1. Record is marked as deleted (flag in header)
2. Space becomes "garbage" (tracked in page header)
3. During reorganization, garbage space is reclaimed
4. Links are updated to skip deleted records

## Usage Analysis

The library calculates page usage as:
```
UsedBytes = HeapTop + TrailerSize + (NumDirSlots × SlotSize) - GarbageSpace
```

This represents the actual bytes used for storing valid data.

## Performance Considerations

### Page Directory

- Enables O(log n) search within a page
- Binary search to find record group
- Linear search within group (4-8 records)

### Record Access Patterns

1. **Sequential**: Follow next pointers (efficient)
2. **Random**: Use page directory for faster lookup
3. **Range**: Start at position, follow chain

### Memory Alignment

InnoDB ensures critical structures are aligned for performance:
- Pages: 16KB aligned
- Records: Generally 8-byte aligned
- Directory slots: 2-byte aligned

## Limitations and Assumptions

This library:
1. Only supports compact record format
2. Assumes 16KB page size
3. Does not decode actual field data
4. Read-only operations only
5. No support for compressed pages
6. No support for encrypted pages

## Further Reading

- [MySQL Internals Manual](https://dev.mysql.com/doc/internals/en/)
- [InnoDB Source Code](https://github.com/mysql/mysql-server)
- [Jeremy Cole's Blog](https://blog.jcole.us/) - Excellent InnoDB internals articles
- [MariaDB Knowledge Base](https://mariadb.com/kb/en/innodb/)
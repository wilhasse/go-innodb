# Project Architecture

## File Organization

The go-innodb library uses a flat package structure with files logically grouped by functionality. Each file has a clear, single responsibility with a descriptive header comment.

### Core Components

#### Type Definitions and Constants
- **types.go** - Core type definitions and constants (PageSize, PageType, RecordType, etc.)
- **endian.go** - Big-endian byte reading utilities for parsing binary data

#### Page Structure Components
- **fil.go** - FIL (File) header and trailer parsing for page metadata
- **inner_page.go** - Base 16KB page structure with FIL header/trailer
- **index_page.go** - INDEX page parsing with user records and page directory
- **index_header.go** - Index-specific header within INDEX pages
- **fseg_header.go** - File segment header parsing (FSEG)

#### Record Handling
- **record_header.go** - Compact record format header parsing (5 bytes)
- **generic_record.go** - Generic record structure with header and position
- **iter.go** - Record iteration and traversal utilities

#### I/O Operations
- **reader.go** - Page reader for reading from InnoDB data files (.ibd)

### Design Principles

1. **Single Responsibility**: Each file handles one specific aspect of InnoDB page parsing
2. **Clear Naming**: File names directly indicate their purpose
3. **Minimal Dependencies**: Files depend only on what they need
4. **Public API**: All exported types and functions are part of the public API
5. **Documentation**: Each file starts with a descriptive comment explaining its purpose

### Data Flow

```
.ibd file → PageReader → InnerPage → ParseIndexPage → IndexPage
                ↓             ↓                           ↓
            ReadPage()    FIL Header/              Records, Directory
                          Trailer                    Slots, Headers
```

### Why Flat Structure?

The flat package structure was chosen for several reasons:

1. **Simplicity**: Easy to understand and navigate
2. **Go Idioms**: Follows Go's preference for simple, flat packages
3. **Clear Dependencies**: No circular dependencies or complex import paths
4. **Library Size**: The library is focused enough that subpackages would add unnecessary complexity
5. **Testing**: Easier to test with all components in the same package

### Future Considerations

If the library grows significantly, consider organizing into packages like:
- `page/` - Page-related types and parsing
- `record/` - Record types and handling
- `io/` - I/O operations
- `format/` - Format constants and utilities

However, the current flat structure works well for the library's scope and makes it easy for users to import and use.
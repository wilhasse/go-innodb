#!/bin/bash

# Script to generate a single file with all Go source code for LLM prompting
# This creates a consolidated view of the go-innodb codebase for analysis
# Excludes test files, vendor, testdata, and the Java implementation

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Output configuration
OUTPUT_DIR="."
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUTPUT_FILE="go-innodb-codebase-${TIMESTAMP}.txt"

# Function to print colored output
print_color() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

echo "📦 Generating consolidated go-innodb codebase for LLM prompting..."

# Function to add a file with header
add_file() {
    local filepath=$1
    local description=$2
    
    if [ -f "$filepath" ]; then
        echo "" >> "$OUTPUT_FILE"
        echo "================================================================================" >> "$OUTPUT_FILE"
        echo "// File: $filepath" >> "$OUTPUT_FILE"
        echo "// Description: $description" >> "$OUTPUT_FILE"
        echo "================================================================================" >> "$OUTPUT_FILE"
        echo "" >> "$OUTPUT_FILE"
        cat "$filepath" >> "$OUTPUT_FILE"
    fi
}

# Start with a header
cat > "$OUTPUT_FILE" << 'EOF'
# go-innodb Codebase
# InnoDB page parser library for Go
# Generated for LLM analysis and understanding
# 
# This library parses InnoDB database pages (16KB pages) and can extract
# actual column data using table schema information.
#
# ⚠️ THIS IS A GENERATED FILE - DO NOT EDIT
# ⚠️ Generated from actual source files

## Project Structure Overview:
- format/: Basic types, constants, and endian utilities
- page/: Page-level structures (InnerPage, IndexPage, etc.)
- record/: Record-level structures and iteration
- schema/: Table schema parsing from SQL
- column/: Column data parsers for different types
- cmd/go-innodb/: Command-line tool

## Key Features:
- Parse InnoDB page structure (FIL headers, INDEX pages, etc.)
- Walk records in B+ tree pages
- Parse CREATE TABLE statements to get schema
- Extract actual column values using schema
- Support for INT, VARCHAR, TIMESTAMP, DATE types
- Handle NULL bitmap and variable-length columns

================================================================================

EOF

# Add main documentation files
add_file "README.md" "Project documentation and usage examples"
add_file "docs/ARCHITECTURE.md" "Detailed architecture documentation"
add_file "CLAUDE.md" "Instructions for AI assistants"

# Add go.mod for dependencies
add_file "go.mod" "Go module definition with dependencies"

# Add format package files
print_color $BLUE "Adding format package..."
add_file "format/types.go" "Core type definitions and constants"
add_file "format/endian.go" "Big-endian byte reading utilities"

# Add page package files
print_color $BLUE "Adding page package..."
add_file "page/inner.go" "Base InnoDB page structure (16KB)"
add_file "page/index.go" "INDEX page parsing with records"
add_file "page/fil.go" "FIL header and trailer parsing"
add_file "page/fseg.go" "File segment header parsing"

# Add record package files
print_color $BLUE "Adding record package..."
add_file "record/header.go" "Compact record format header (5 bytes)"
add_file "record/generic.go" "Generic record structure with values"
add_file "record/iterator.go" "Record iteration and traversal"
add_file "record/index_header.go" "Index-specific header parsing"
add_file "record/compact_parser.go" "Parser for InnoDB compact record format"

# Add schema package files
print_color $BLUE "Adding schema package..."
add_file "schema/column.go" "Column definition for table schema"
add_file "schema/table_def.go" "Table definition with columns"
add_file "schema/parser.go" "Parse CREATE TABLE SQL statements"

# Add column package files
print_color $BLUE "Adding column package..."
add_file "column/parser.go" "Column parser interface and base"
add_file "column/factory.go" "Factory for getting appropriate parser"
add_file "column/int_parser.go" "Parser for integer column types"
add_file "column/string_parser.go" "Parser for string/text columns"
add_file "column/datetime_parser.go" "Parser for date and time columns"

# Add compression support files
print_color $BLUE "Adding compression support..."
add_file "compressed.go" "InnoDB compressed page support with cgo bindings"
add_file "reader_compressed.go" "Compressed page reader implementation"
add_file "lib/zipshim.cpp" "C++ wrapper implementing Oracle/Percona approach"
add_file "lib/mysql_stubs.cpp" "MySQL stubs following Oracle engineers' guidance"
add_file "lib/Makefile" "Build configuration for C++ compression library"

# Add main package files
print_color $BLUE "Adding main package files..."
add_file "reader.go" "Page reader for InnoDB data files"
add_file "exports.go" "Re-exports for main package API"
add_file "doc.go" "Package documentation"

# Add command-line tool
print_color $BLUE "Adding CLI tool..."
add_file "cmd/go-innodb/main.go" "Command-line tool for parsing InnoDB files"

# Add examples
print_color $BLUE "Adding examples..."
add_file "examples/compressed_example.go" "Example using compressed page support"

# Add example SQL for testing
add_file "testdata/users/users.sql" "Example CREATE TABLE statement"

# Add footer with statistics
echo "" >> "$OUTPUT_FILE"
echo "================================================================================" >> "$OUTPUT_FILE"
echo "// Code Statistics" >> "$OUTPUT_FILE"
echo "================================================================================" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

# Count lines and files
TOTAL_LINES=$(wc -l < "$OUTPUT_FILE")
GO_FILES=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./innodb-java-reader/*" -not -path "./*_test.go" 2>/dev/null | wc -l)
PACKAGE_COUNT=$(find . -type d -name "*" -not -path "./.*" -not -path "./vendor/*" -not -path "./innodb-java-reader/*" -not -path "./testdata/*" -maxdepth 1 | wc -l)

echo "Total lines in this document: $TOTAL_LINES" >> "$OUTPUT_FILE"
echo "Total Go files in project: $GO_FILES" >> "$OUTPUT_FILE"
echo "Total packages: $PACKAGE_COUNT" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"
echo "Generated at: $(date)" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

echo "⚠️ IMPORTANT NOTES FOR LLM:" >> "$OUTPUT_FILE"
echo "================================" >> "$OUTPUT_FILE"
echo "1. This is a SINGLE GENERATED FILE containing multiple source files concatenated" >> "$OUTPUT_FILE"
echo "2. Each section marked with '// File:' is a DIFFERENT source file" >> "$OUTPUT_FILE"
echo "3. These are NOT multiple versions - they are DIFFERENT files from the project" >> "$OUTPUT_FILE"
echo "4. The actual source code is in the directories shown in each file header" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

echo "Key Implementation Details:" >> "$OUTPUT_FILE"
echo "1. InnoDB pages are always 16KB logical (compressed pages are 1K/2K/4K/8K physical)" >> "$OUTPUT_FILE"
echo "2. Multi-byte values use big-endian encoding" >> "$OUTPUT_FILE"
echo "3. Signed integers are stored with XOR transformation (sign bit flipped)" >> "$OUTPUT_FILE"
echo "4. Compact record format: [var headers][null bitmap][5-byte header][data]" >> "$OUTPUT_FILE"
echo "5. Variable-length headers are stored in reverse order before record header" >> "$OUTPUT_FILE"
echo "6. Primary key columns come first in record data" >> "$OUTPUT_FILE"
echo "7. Leaf pages have 13-byte transaction fields after primary key" >> "$OUTPUT_FILE"
echo "8. Compression: Only INDEX pages (17855) are decompressed, others copied as-is" >> "$OUTPUT_FILE"
echo "9. Uses Oracle/Percona approach with libinnodb_zipdecompress.a via cgo" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

echo "Example Usage:" >> "$OUTPUT_FILE"
echo "  go run cmd/go-innodb -file users.ibd -page 4 -sql users.sql -parse -records" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"
echo "This will:" >> "$OUTPUT_FILE"
echo "  1. Load the table schema from users.sql" >> "$OUTPUT_FILE"
echo "  2. Read page 4 from users.ibd" >> "$OUTPUT_FILE"
echo "  3. Parse and display the actual column values" >> "$OUTPUT_FILE"

print_color $GREEN ""
print_color $GREEN "✅ Generated: $OUTPUT_FILE"
print_color $GREEN "📊 File size: $(du -h "$OUTPUT_FILE" | cut -f1)"
print_color $GREEN "📝 Total lines: $TOTAL_LINES"
print_color $GREEN ""
print_color $GREEN "You can now use this file to prompt an LLM with the complete codebase context."
print_color $YELLOW ""
print_color $YELLOW "💡 Tip: Add this file to .gitignore to avoid committing it:"
print_color $YELLOW "    echo 'go-innodb-codebase-*.txt' >> .gitignore"
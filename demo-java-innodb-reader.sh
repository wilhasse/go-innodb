#!/bin/bash

# InnoDB Java Reader Demo Script
# This script demonstrates how to read MySQL InnoDB files directly without a MySQL server
# using the Java innodb-java-reader tool from https://github.com/alibaba/innodb-java-reader
#
# IMPORTANT: This requires the innodb-java-reader project to be built first!
# Run: cd innodb-java-reader && mvn clean install -DskipTests

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JAR_FILE="$SCRIPT_DIR/innodb-java-reader/innodb-java-reader-cli/target/innodb-java-reader-cli.jar"
TESTDATA_DIR="$SCRIPT_DIR/testdata/users"
IBD_FILE="$TESTDATA_DIR/users.ibd"
SQL_FILE="$TESTDATA_DIR/users.sql"
OUTPUT_DIR="$TESTDATA_DIR/output"

# Function to print colored output
print_header() {
    echo -e "\n${BLUE}===================================================${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${BLUE}===================================================${NC}\n"
}

print_info() {
    echo -e "${YELLOW}➜${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if JAR file exists
if [ ! -f "$JAR_FILE" ]; then
    print_error "JAR file not found at: $JAR_FILE"
    echo ""
    echo -e "${YELLOW}The innodb-java-reader project needs to be built first.${NC}"
    echo -e "${YELLOW}This is the Java implementation from: https://github.com/alibaba/innodb-java-reader${NC}"
    echo ""
    echo "Please build the project:"
    echo "  1. cd innodb-java-reader"
    echo "  2. mvn clean install -DskipTests -Dpmd.skip=true -Dcheckstyle.skip=true"
    echo ""
    echo "Requirements:"
    echo "  - Java 17 or later"
    echo "  - Maven"
    exit 1
fi

# Check if test data exists
if [ ! -f "$IBD_FILE" ]; then
    print_error "InnoDB data file not found at: $IBD_FILE"
    echo "Please ensure the test data exists in testdata/users/"
    exit 1
fi

if [ ! -f "$SQL_FILE" ]; then
    print_error "SQL file not found at: $SQL_FILE"
    echo "Please ensure the SQL file exists in testdata/users/"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Main demo
clear
echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}     ${GREEN}InnoDB Java Reader Demo${NC}                              ${BLUE}║${NC}"
echo -e "${BLUE}║${NC}     Reading MySQL InnoDB files without MySQL Server       ${BLUE}║${NC}"
echo -e "${BLUE}║${NC}     Using: https://github.com/alibaba/innodb-java-reader  ${BLUE}║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"

# Show table structure
print_header "1. Table Structure"
print_info "Reading from: $SQL_FILE"
cat "$SQL_FILE"
print_success "Table structure loaded"

# Show all pages in the InnoDB file
print_header "2. InnoDB File Page Structure"
print_info "Analyzing pages in: $IBD_FILE"
java -jar "$JAR_FILE" \
    -ibd-file-path "$IBD_FILE" \
    -create-table-sql-file-path "$SQL_FILE" \
    -c show-all-pages
print_success "Page analysis complete"

# Query all records
print_header "3. Table Data (All Records)"
print_info "Extracting all records from InnoDB file..."
java -jar "$JAR_FILE" \
    -ibd-file-path "$IBD_FILE" \
    -create-table-sql-file-path "$SQL_FILE" \
    -c query-all \
    -showheader
print_success "Data extraction complete"

# Export to file with headers
print_header "4. Export Data to Files"
print_info "Exporting to CSV format..."
java -jar "$JAR_FILE" \
    -ibd-file-path "$IBD_FILE" \
    -create-table-sql-file-path "$SQL_FILE" \
    -c query-all \
    -showheader \
    -delimiter "," \
    -o "$OUTPUT_DIR/users_export.csv" 2>/dev/null
print_success "Exported to: $OUTPUT_DIR/users_export.csv"

print_info "Exporting to TSV format..."
java -jar "$JAR_FILE" \
    -ibd-file-path "$IBD_FILE" \
    -create-table-sql-file-path "$SQL_FILE" \
    -c query-all \
    -showheader \
    -o "$OUTPUT_DIR/users_export.tsv" 2>/dev/null
print_success "Exported to: $OUTPUT_DIR/users_export.tsv"

# Query specific record by primary key
print_header "5. Query by Primary Key"
print_info "Querying record with id=2..."
java -jar "$JAR_FILE" \
    -ibd-file-path "$IBD_FILE" \
    -create-table-sql-file-path "$SQL_FILE" \
    -c query-by-pk \
    -args 2
print_success "Primary key query complete"

# Range query
print_header "6. Range Query"
print_info "Querying records with id >= 1 and id <= 2..."
java -jar "$JAR_FILE" \
    -ibd-file-path "$IBD_FILE" \
    -create-table-sql-file-path "$SQL_FILE" \
    -c range-query-by-pk \
    -args ">=;1;<=;2"
print_success "Range query complete"

# Query specific page
print_header "7. Query Specific Page"
print_info "Reading records from page 4 (INDEX page)..."
java -jar "$JAR_FILE" \
    -ibd-file-path "$IBD_FILE" \
    -create-table-sql-file-path "$SQL_FILE" \
    -c query-by-page-number \
    -args 4
print_success "Page query complete"

# Summary
print_header "Demo Complete!"
echo -e "${GREEN}Successfully demonstrated the following capabilities:${NC}"
echo "  • Reading InnoDB file structure"
echo "  • Extracting all table data"
echo "  • Querying by primary key"
echo "  • Range queries"
echo "  • Exporting data to various formats"
echo "  • Direct page access"
echo ""
echo -e "${YELLOW}Output files created in: $OUTPUT_DIR${NC}"
ls -la "$OUTPUT_DIR" 2>/dev/null || true
echo ""
echo -e "${BLUE}This tool is the Java implementation from:${NC}"
echo -e "${BLUE}https://github.com/alibaba/innodb-java-reader${NC}"
echo ""
echo -e "${GREEN}It allows you to read MySQL InnoDB files directly${NC}"
echo -e "${GREEN}without needing a running MySQL server!${NC}"
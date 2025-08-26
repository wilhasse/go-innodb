#!/bin/bash
# Build script for InnoDB compression support

set -e

echo "Building InnoDB compression support..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if the InnoDB library exists
if [ ! -f "lib/libinnodb_zipdecompress.a" ]; then
    echo -e "${RED}Error: lib/libinnodb_zipdecompress.a not found!${NC}"
    echo "Please add the InnoDB compression library to the lib/ directory"
    exit 1
fi

# Build the C++ shim library
echo "Building C++ shim library..."
cd lib

# Check for required dependencies
echo "Checking dependencies..."
MISSING_DEPS=""

# Check for lz4
if ! pkg-config --exists liblz4 2>/dev/null; then
    # Try alternative check if ldconfig is available
    if command -v ldconfig >/dev/null 2>&1; then
        if ! ldconfig -p 2>/dev/null | grep -q liblz4; then
            MISSING_DEPS="$MISSING_DEPS liblz4"
        fi
    else
        # Just check if the library file exists
        if ! ls /usr/lib*/liblz4.* /usr/local/lib/liblz4.* 2>/dev/null | grep -q .; then
            MISSING_DEPS="$MISSING_DEPS liblz4"
        fi
    fi
fi

# Check for zlib
if ! pkg-config --exists zlib 2>/dev/null; then
    # Try alternative check
    if ! ls /usr/lib*/libz.* /usr/local/lib/libz.* 2>/dev/null | grep -q .; then
        MISSING_DEPS="$MISSING_DEPS zlib"
    fi
fi

if [ -n "$MISSING_DEPS" ]; then
    echo -e "${YELLOW}Warning: Missing dependencies:${MISSING_DEPS}${NC}"
    echo "You may need to install: sudo apt-get install liblz4-dev zlib1g-dev"
    echo "Or on Mac: brew install lz4 zlib"
fi

# Compile the shim
echo "Compiling zipshim.cpp..."
g++ -fPIC -O2 -Wall -std=c++11 -c zipshim.cpp -o zipshim.o

echo "Compiling mysql_stubs.cpp..."
g++ -fPIC -O2 -Wall -std=c++11 -c mysql_stubs.cpp -o mysql_stubs.o

# Create shared library
echo "Creating shared library..."
g++ -shared -o libzipshim.so zipshim.o mysql_stubs.o -Wl,--whole-archive libinnodb_zipdecompress.a -Wl,--no-whole-archive -lz -llz4 -lstdc++

# Also create static library for static linking
echo "Creating static library..."
ar rcs libzipshim.a zipshim.o mysql_stubs.o

cd ..

# Test compilation with Go
echo "Testing Go compilation..."
go build -tags cgo ./...

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Compression support built successfully!${NC}"
    echo ""
    echo "IMPORTANT: To run binaries, you need to set the library path:"
    echo "  Option 1 (Quick): LD_LIBRARY_PATH=\$PWD/lib ./go-innodb"
    echo "  Option 2 (Permanent): sudo cp lib/libzipshim.so /usr/local/lib/ && sudo ldconfig"
    echo ""
    echo "You can now use compressed page support in your Go code:"
    echo "  reader := goinnodb.NewCompressedPageReader(file)"
    echo "  page, err := reader.ReadCompressedPage(pageNo, 8192) // for 8K compressed pages"
else
    echo -e "${RED}✗ Go compilation failed${NC}"
    echo "Check that cgo is enabled and dependencies are installed"
    exit 1
fi
# Installation Guide

Complete installation instructions for go-innodb, including optional features.

## Prerequisites

- **Go 1.19+**: Required for building the library
- **Git**: For cloning the repository  
- **Make**: For using the build system

## Basic Installation

### 1. Clone Repository

```bash
git clone https://github.com/wilhasse/go-innodb.git
cd go-innodb
```

### 2. Basic Build

```bash
# Build library and CLI tool
make build

# Verify installation
./go-innodb --help
```

### 3. Optional: Install to PATH

```bash
# Install to $GOPATH/bin
make install

# Or copy to system PATH
sudo cp go-innodb /usr/local/bin/
```

## Compressed Page Support

For reading InnoDB compressed tables (tables created with `ROW_FORMAT=COMPRESSED` and `KEY_BLOCK_SIZE`).

### System Dependencies

Install required development libraries:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install liblz4-dev zlib1g-dev g++ make

# RHEL/CentOS/Fedora
sudo dnf install lz4-devel zlib-devel gcc-c++ make

# macOS (with Homebrew)
brew install lz4 zlib

# Alpine Linux
apk add lz4-dev zlib-dev g++ make
```

### InnoDB Decompression Library

You need `libinnodb_zipdecompress.a` from MySQL:

#### Option 1: From MySQL Binary Distribution

```bash
# Download MySQL server tarball
wget https://dev.mysql.com/get/Downloads/MySQL-8.0/mysql-8.0.35-linux-glibc2.17-x86_64.tar.xz

# Extract and find the library
tar -xf mysql-8.0.35-linux-glibc2.17-x86_64.tar.xz
find mysql-8.0.35-linux-glibc2.17-x86_64 -name "*zipdecompress*" -o -name "*innodb*compress*"

# Copy to project lib directory
cp /path/to/libinnodb_zipdecompress.a lib/
```

#### Option 2: From MySQL Source

```bash
# Clone MySQL source
git clone https://github.com/mysql/mysql-server.git
cd mysql-server

# Build MySQL with compression support
mkdir build && cd build
cmake .. -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_ZLIB=bundled
make -j$(nproc)

# Find and copy the library
find . -name "*zipdecompress*" -o -name "*innodb*compress*"
cp /path/to/libinnodb_zipdecompress.a /path/to/go-innodb/lib/
```

#### Option 3: Package Managers

```bash
# Some distributions may include it in development packages
# Ubuntu/Debian
apt-cache search mysql | grep dev
sudo apt-get install libmysqlclient-dev

# Look for the library
find /usr -name "*zipdecompress*" 2>/dev/null
find /usr -name "*innodb*compress*" 2>/dev/null
```

### Build Compression Support

```bash
# Verify library is in place
ls -la lib/libinnodb_zipdecompress.a

# Build compression support
chmod +x build_compression.sh
./build_compression.sh
```

Expected output:
```
Building InnoDB compression support...
Building C++ shim library...
Checking dependencies...
Compiling zipshim.cpp...
Compiling mysql_stubs.cpp...
Creating shared library...
Creating static library...
Testing Go compilation...
✓ Compression support built successfully!
```

### Runtime Library Installation

The binary needs access to `libzipshim.so` at runtime:

#### Recommended: System-wide Installation

```bash
# Install shared library system-wide
sudo cp lib/libzipshim.so /usr/local/lib/
sudo ldconfig

# Verify installation
ldd ./go-innodb | grep zipshim
# Should show: libzipshim.so => /usr/local/lib/libzipshim.so
```

#### Alternative: Environment Variable

```bash
# Set library path for current session
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH

# Add to shell profile for persistence
echo 'export LD_LIBRARY_PATH=/path/to/go-innodb/lib:$LD_LIBRARY_PATH' >> ~/.bashrc
source ~/.bashrc
```

#### Alternative: Per-execution

```bash
# Run with library path each time
LD_LIBRARY_PATH=$PWD/lib ./go-innodb -file compressed.ibd
```

### Verification

Test compression support:

```bash
# Test basic functionality
./go-innodb --help

# Test with compressed table (if you have one)
./go-innodb -file compressed_table.ibd -page 4

# Check library dependencies
ldd ./go-innodb
```

## Build Options

### CGO Control

```bash
# Build with compression support
CGO_ENABLED=1 go build -tags cgo

# Build without compression (smaller binary, no compression support)
CGO_ENABLED=0 go build -tags !cgo
```

### Static Linking

For completely static binaries:

```bash
# Create static library version
cd lib && make static

# Build statically linked binary
CGO_ENABLED=1 go build -tags cgo -ldflags "-extldflags=-static"
```

## Troubleshooting

### Build Issues

**"build_compression.sh: Permission denied"**
```bash
chmod +x build_compression.sh
```

**"libinnodb_zipdecompress.a not found"**
```bash
# Check file exists and permissions
ls -la lib/libinnodb_zipdecompress.a
# Should show readable file, not empty
```

**"undefined reference to lz4_*"**
```bash
# Install development packages
sudo apt-get install liblz4-dev zlib1g-dev
```

### Runtime Issues

**"error while loading shared libraries: libzipshim.so"**
```bash
# Install library system-wide (recommended fix)
sudo cp lib/libzipshim.so /usr/local/lib/
sudo ldconfig

# Or use LD_LIBRARY_PATH
LD_LIBRARY_PATH=$PWD/lib ./go-innodb
```

**"cgo not enabled"**
```bash
# Enable cgo
CGO_ENABLED=1 go build -tags cgo
```

### Verification Commands

```bash
# Check Go environment
go env CGO_ENABLED

# Check library dependencies
ldd ./go-innodb

# Check symbols in library
nm lib/libzipshim.so | grep -E "(decompress|srv_page)"

# Check library installation
ldconfig -p | grep zipshim
```

## Platform-Specific Notes

### Linux
- Most distributions work with the standard installation
- Use package manager for dependencies when possible
- ldconfig requires sudo privileges

### macOS
- Use Homebrew for dependencies: `brew install lz4 zlib`
- May need to set C compiler: `export CC=clang`
- Shared libraries use `.dylib` extension (handled automatically)

### Docker/Containers
```dockerfile
# Install dependencies in container
RUN apt-get update && apt-get install -y \
    liblz4-dev zlib1g-dev g++ make \
    && rm -rf /var/lib/apt/lists/*

# Copy library and build
COPY lib/libinnodb_zipdecompress.a /app/lib/
RUN ./build_compression.sh && \
    cp lib/libzipshim.so /usr/local/lib/ && \
    ldconfig
```

## Next Steps

After installation:

1. **Test with sample data**: Use files in `testdata/` directory
2. **Read documentation**: Start with [COMPRESSED_PAGES.md](COMPRESSED_PAGES.md) for compression features
3. **Try examples**: Run the examples in `examples/` directory
4. **Explore API**: Check [API.md](API.md) for programmatic usage

## Support

For installation issues:

1. Check [COMPRESSED_PAGES.md](COMPRESSED_PAGES.md) troubleshooting section
2. Verify prerequisites are met
3. Run build with verbose output: `make build V=1`
4. Open issue with full error output and system information
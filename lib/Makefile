# Makefile for building the InnoDB decompression library
# This provides a minimal library for decompressing InnoDB pages without MySQL dependencies

CXX = g++
CXXFLAGS = -fPIC -O2 -Wall -std=c++11
LDFLAGS = -shared

# InnoDB static library (from MySQL/Percona build)
INNODB_LIB = libinnodb_zipdecompress.a

# Decompression library
TARGET = libinnodb_decompress.so
SOURCES = innodb_decompress.cpp mysql_stubs.cpp
OBJECTS = $(SOURCES:.cpp=.o)

# Default target
all: $(TARGET)

# Build the decompression library
$(TARGET): $(OBJECTS)
	$(CXX) $(LDFLAGS) -o $@ $(OBJECTS) \
		-Wl,--whole-archive $(INNODB_LIB) -Wl,--no-whole-archive \
		-lz -llz4

# Compile source files
%.o: %.cpp
	$(CXX) $(CXXFLAGS) -c $< -o $@

# Clean build artifacts
clean:
	rm -f $(OBJECTS) $(TARGET)

# Install (optional)
install: $(TARGET)
	@echo "Copy $(TARGET) to your library path or set LD_LIBRARY_PATH"

.PHONY: all clean install
# Makefile for building the InnoDB compression shim library

CXX = g++
CXXFLAGS = -fPIC -O2 -Wall -std=c++11
LDFLAGS = -shared

# Output library name
TARGET = libzipshim.so

# Source files
SOURCES = zipshim.cpp mysql_stubs.cpp

# Object files
OBJECTS = $(SOURCES:.cpp=.o)

# InnoDB library
INNODB_LIB = libinnodb_zipdecompress.a

# Build the shared library
# Note: We need to include the whole static library to resolve all symbols
$(TARGET): $(OBJECTS)
	$(CXX) $(LDFLAGS) -o $@ $(OBJECTS) -Wl,--whole-archive $(INNODB_LIB) -Wl,--no-whole-archive -lz -llz4 -lstdc++

# Compile source files
%.o: %.cpp
	$(CXX) $(CXXFLAGS) -c $< -o $@

# Build static library (alternative)
static: $(OBJECTS)
	ar rcs libzipshim.a $(OBJECTS)

# New minimal decompression library
DECOMPRESS_TARGET = libinnodb_decompress.so
DECOMPRESS_SOURCES = innodb_decompress_v3.cpp mysql_stubs.cpp
DECOMPRESS_OBJECTS = innodb_decompress_v3.o mysql_stubs.o

# Build the minimal decompression library
# Include the static library directly for standalone operation
$(DECOMPRESS_TARGET): $(DECOMPRESS_OBJECTS)
	$(CXX) $(LDFLAGS) -o $@ $(DECOMPRESS_OBJECTS) -Wl,--whole-archive $(INNODB_LIB) -Wl,--no-whole-archive -lz -llz4

# Build both libraries
all: $(TARGET) $(DECOMPRESS_TARGET)

# Build only the minimal decompression library
decompress: $(DECOMPRESS_TARGET)

clean:
	rm -f $(OBJECTS) $(TARGET) libzipshim.a $(DECOMPRESS_OBJECTS) $(DECOMPRESS_TARGET)

.PHONY: clean static all decompress
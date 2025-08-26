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

clean:
	rm -f $(OBJECTS) $(TARGET) libzipshim.a

.PHONY: clean static
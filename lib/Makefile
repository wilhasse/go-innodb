# Makefile for building the InnoDB compression shim library

CXX = g++
CXXFLAGS = -fPIC -O2 -Wall -std=c++11
LDFLAGS = -shared

# Output library name
TARGET = libzipshim.so

# Source files
SOURCES = zipshim.cpp

# Object files
OBJECTS = $(SOURCES:.cpp=.o)

# InnoDB library
INNODB_LIB = libinnodb_zipdecompress.a

# Build the shared library
$(TARGET): $(OBJECTS)
	$(CXX) $(LDFLAGS) -o $@ $(OBJECTS) $(INNODB_LIB) -lz -llz4 -lstdc++

# Compile source files
%.o: %.cpp
	$(CXX) $(CXXFLAGS) -c $< -o $@

# Build static library (alternative)
static: $(OBJECTS)
	ar rcs libzipshim.a $(OBJECTS)

clean:
	rm -f $(OBJECTS) $(TARGET) libzipshim.a

.PHONY: clean static
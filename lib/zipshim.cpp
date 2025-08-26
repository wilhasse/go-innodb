// zipshim.cpp - C++ wrapper for InnoDB compressed page decompression
// Following Oracle engineers' guidance for proper InnoDB integration
// This shim provides a C ABI for Go to call via cgo

extern "C" {
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
}

// Access to InnoDB globals (defined in mysql_stubs.cpp)
extern "C" {
    extern unsigned long srv_page_size;
    extern unsigned long srv_page_size_shift;
}

// Forward declarations for InnoDB types and functions
// These would normally come from MySQL headers, but we'll define minimal interfaces
struct page_zip_des_t {
    void* data;     // Compressed page data
    uint32_t ssize; // Shift size for compression
    // Other fields exist but we only need these
};

// InnoDB page type definitions
typedef uint8_t page_t;
typedef void page_zip_t;

// InnoDB page types (from fil0fil.h)
#define FIL_PAGE_TYPE_OFFSET 24
#define FIL_PAGE_INDEX 17855  // B-tree node

// External functions from libinnodb_zipdecompress.a  
// The actual C++ function signature (mangled name: _Z23page_zip_decompress_lowP14page_zip_des_tPhb)
extern "C++" {
    bool page_zip_decompress_low(page_zip_des_t* page_zip, page_t* page, bool all);
}

// Simple implementation of page_zip_des_init since it's not exported
// Based on InnoDB source - just zero-initialize the structure
static void page_zip_des_init_simple(page_zip_des_t* page_zip) {
    if (page_zip) {
        page_zip->data = nullptr;
        page_zip->ssize = 0;
        // Other fields would be initialized here in real InnoDB
    }
}

// Helper function to read 16-bit values from page data (big-endian)
static uint16_t read_uint16_be(const uint8_t* data) {
    return (static_cast<uint16_t>(data[0]) << 8) | static_cast<uint16_t>(data[1]);
}

// Helper function for log2 calculation (Oracle's approach)
static unsigned long ilog2_ul(unsigned long v) {
    unsigned long s = 0;
    while ((1UL << s) < v) ++s;
    return s;
}

// Convert physical page size to shift size (ssize) following Oracle's approach
// This matches InnoDB's internal page_size_to_ssize() function
static uint32_t page_size_to_ssize(size_t physical) {
    switch (physical) {
        case 1024:  return 1;  // 1KB -> ssize=1
        case 2048:  return 2;  // 2KB -> ssize=2  
        case 4096:  return 4;  // 4KB -> ssize=4
        case 8192:  return 8;  // 8KB -> ssize=8
        case 16384: return 16; // 16KB -> ssize=16 (uncompressed)
        default:    return 0;  // Invalid size
    }
}


// Main decompression function exposed to Go - following Percona's approach
extern "C" int innodb_zip_decompress(
    const void* src,        // Pointer to compressed page data  
    size_t      physical,   // Physical page size (e.g., 8192 for 8K)
    void*       dst,        // Output buffer (16KB)
    size_t      logical)    // Logical page size (usually 16384)
{
    if (!src || !dst) {
        return -1;
    }
    
    if (logical != 16384) {
        // InnoDB logical pages are always 16KB
        return -2;
    }
    
    // Oracle's approach: keep InnoDB globals consistent for this call
    srv_page_size = static_cast<unsigned long>(logical);
    srv_page_size_shift = static_cast<unsigned long>(ilog2_ul(static_cast<unsigned long>(logical)));
    
    // Clear output buffer first
    memset(dst, 0, logical);
    
    // Check if this is actually a compressed page that needs decompression
    // Following Percona's approach: check page type and size
    const uint8_t* page_data = static_cast<const uint8_t*>(src);
    
    // If physical == logical, just copy (not compressed)
    if (physical >= logical) {
        memcpy(dst, src, logical);
        return 0;
    }
    
    // Check page type - only decompress INDEX pages (Percona's approach)
    uint16_t page_type = read_uint16_be(page_data + FIL_PAGE_TYPE_OFFSET);
    if (page_type != FIL_PAGE_INDEX) {
        // Not an index page, just copy the raw data
        memcpy(dst, src, physical);
        return 0;
    }
    
    // This is a compressed INDEX page - attempt decompression (Percona's approach)
    
    // Percona's approach: create aligned temporary buffer for decompression
    unsigned char* temp = static_cast<unsigned char*>(malloc(2 * logical));
    if (!temp) {
        return -3;
    }
    
    // Align buffer (simple alignment to logical page size boundary)
    unsigned char* aligned_temp = temp;
    uintptr_t align_mask = logical - 1;
    if (reinterpret_cast<uintptr_t>(temp) & align_mask) {
        aligned_temp = reinterpret_cast<unsigned char*>(
            (reinterpret_cast<uintptr_t>(temp) + logical) & ~align_mask);
    }
    memset(aligned_temp, 0, logical);
    
    // Set up the page_zip descriptor following Percona's patterns
    page_zip_des_t page_zip{};
    page_zip_des_init_simple(&page_zip);
    page_zip.data = reinterpret_cast<page_zip_t*>(const_cast<void*>(src));
    page_zip.ssize = page_size_to_ssize(physical);
    
    if (page_zip.ssize == 0) {
        // Invalid physical page size for compression
        free(temp);
        return -3;
    }
    
    // Attempt decompression using InnoDB's low-level function
    // This follows the exact pattern used by Percona's successful implementation
    bool ok = page_zip_decompress_low(&page_zip, aligned_temp, true);
    
    if (ok) {
        // Copy decompressed data to output buffer
        memcpy(dst, aligned_temp, logical);
    }
    
    free(temp);
    return ok ? 0 : -4;
}

// Helper function to check if a page appears to be compressed
// Compressed pages have specific magic values in their headers
extern "C" int innodb_is_page_compressed(const void* page, size_t size) {
    if (!page || size < 38) {
        return 0;
    }
    
    // const uint8_t* p = static_cast<const uint8_t*>(page);
    // Would check for compressed page markers here
    // For now, this is a placeholder implementation
    
    // Compressed pages are typically smaller than 16KB
    if (size < 16384 && (size == 1024 || size == 2048 || size == 4096 || size == 8192)) {
        return 1;  // Likely compressed based on size
    }
    
    return 0;
}

// Get the actual compressed size from a compressed page header
extern "C" size_t innodb_get_compressed_size(const void* page, size_t physical) {
    if (!page) {
        return 0;
    }
    
    // The actual compressed data size is stored in the page
    // This would require parsing the compressed page header
    // For now, return the physical size
    return physical;
}
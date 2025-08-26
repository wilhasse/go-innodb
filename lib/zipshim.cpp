// zipshim.cpp - C++ wrapper for InnoDB compressed page decompression
// Following Oracle engineers' guidance for proper InnoDB integration
// This shim provides a C ABI for Go to call via cgo

extern "C" {
#include <stdint.h>
#include <string.h>
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

// External functions from libinnodb_zipdecompress.a
// The actual C++ function signature (mangled name: _Z23page_zip_decompress_lowP14page_zip_des_tPhb)
extern "C++" {
    bool page_zip_decompress_low(page_zip_des_t* page_zip, page_t* page, bool all);
}

// Simple init function - we'll do it ourselves since it may not be exported
static void page_zip_des_init(page_zip_des_t* page_zip) {
    if (page_zip) {
        // Basic initialization - the important fields are set by us
        memset(page_zip, 0, sizeof(*page_zip));
    }
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

// Main decompression function exposed to Go - following Oracle's guidance
// Oracle's approach: update srv_page_size globals at runtime for consistency
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
    
    // Build page_zip descriptor following Oracle's pattern
    page_zip_des_t z{};  // C++11 brace initialization
    page_zip_des_init(&z);
    z.data = const_cast<void*>(src);
    z.ssize = page_size_to_ssize(physical);  // derives zip shift from physical size
    
    if (z.ssize == 0) {
        // Invalid physical page size
        return -3;
    }
    
    // Decompress the whole page into 'dst'
    // Oracle's approach: page_zip_decompress_low(&z, dst, /*all=*/true)
    bool ok = page_zip_decompress_low(&z, static_cast<page_t*>(dst), true);
    
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
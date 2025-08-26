// zipshim.cpp - C++ wrapper for InnoDB compressed page decompression
// This shim provides a C ABI for Go to call via cgo

extern "C" {
#include <stdint.h>
#include <string.h>
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
extern "C" {
    void page_zip_des_init(page_zip_des_t* page_zip);
    bool page_zip_decompress_low(const page_zip_des_t* page_zip, page_t* page, bool all);
}

// Convert physical page size to shift size (ssize)
// 1KB = 1 << 10, 2KB = 2 << 10, 4KB = 4 << 10, 8KB = 8 << 10
static uint32_t page_size_to_ssize(size_t physical) {
    switch (physical) {
        case 1024:  return 1;  // 1KB
        case 2048:  return 2;  // 2KB  
        case 4096:  return 4;  // 4KB
        case 8192:  return 8;  // 8KB
        case 16384: return 16; // 16KB (though this would mean no compression)
        default:    return 0;  // Invalid
    }
}

// Main decompression function exposed to Go
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
    
    // Initialize page_zip descriptor
    page_zip_des_t z;
    memset(&z, 0, sizeof(z));
    page_zip_des_init(&z);
    
    z.data = const_cast<void*>(src);
    z.ssize = page_size_to_ssize(physical);
    
    if (z.ssize == 0) {
        // Invalid physical page size
        return -3;
    }
    
    // Decompress the page
    // The 'all' parameter tells it to decompress the entire page
    bool ok = page_zip_decompress_low(&z, static_cast<page_t*>(dst), true);
    
    return ok ? 0 : -4;
}

// Helper function to check if a page appears to be compressed
// Compressed pages have specific magic values in their headers
extern "C" int innodb_is_page_compressed(const void* page, size_t size) {
    if (!page || size < 38) {
        return 0;
    }
    
    const uint8_t* p = static_cast<const uint8_t*>(page);
    
    // Check for PAGE_ZIP_MIN_SIZE marker at offset 0
    // Compressed pages start with specific patterns
    // This is a simplified check - actual detection is more complex
    
    // Look for patterns that indicate compression
    // Compressed pages typically have:
    // - Different checksum format
    // - Compression info in FIL header
    
    // For now, return 0 (not compressed) as we need actual MySQL headers
    // for proper detection
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
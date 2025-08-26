// zipshim.cpp - C++ wrapper for InnoDB compressed page decompression
// Following Oracle engineers' guidance for proper InnoDB integration
// This shim provides a C ABI for Go to call via cgo

extern "C" {
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
}

// Simple page_size_t implementation (minimal version for our needs)
class page_size_t {
public:
    page_size_t(unsigned long logical, unsigned long physical, bool compressed) 
        : m_logical(logical), m_physical(physical), m_compressed(compressed) {}
    
    void copy_from(const page_size_t& other) {
        m_logical = other.m_logical;
        m_physical = other.m_physical; 
        m_compressed = other.m_compressed;
    }
    
    unsigned long logical() const { return m_logical; }
    unsigned long physical() const { return m_physical; }
    
private:
    unsigned long m_logical;
    unsigned long m_physical;
    bool m_compressed;
};

// Access to InnoDB globals (defined in mysql_stubs.cpp)
extern "C" {
    extern unsigned long srv_page_size;
    extern unsigned long srv_page_size_shift;
}

// Global page size object (Oracle engineers' guidance: must be set correctly for compression)
page_size_t univ_page_size(16384, 16384, false);

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
#define FIL_PAGE_INDEX 17855              // B-tree node
#define FIL_PAGE_COMPRESSED 14            // Compressed page
#define FIL_PAGE_COMPRESSED_AND_ENCRYPTED 16  // Compressed and encrypted page
#define FIL_PAGE_DATA 38                  // Start of page data (after 38-byte FIL header)

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

// Percona engineers' CORRECT ZIP ssize calculation  
// ZIP formula: 1 << (10 + ssize) = physical_size
// So 8KB → ssize=3 (because 1 << (10+3) = 1 << 13 = 8192)
static inline uint32_t zip_shift_from_bytes(size_t z) {
    switch (z) {
        case 1024:  return 0; // 1KB -> ssize=0 (1 << (10+0) = 1024)
        case 2048:  return 1; // 2KB -> ssize=1 (1 << (10+1) = 2048)
        case 4096:  return 2; // 4KB -> ssize=2 (1 << (10+2) = 4096)  
        case 8192:  return 3; // 8KB -> ssize=3 (1 << (10+3) = 8192)
        case 16384: return 4; // 16KB -> ssize=4 (uncompressed/legacy)
        default:    return 0; // Invalid => let caller handle
    }
}


// Main decompression function exposed to Go - following Percona's approach
extern "C" int innodb_zip_decompress(
    const void* src,        // Pointer to compressed page data  
    size_t      physical,   // Physical page size (e.g., 8192 for 8K)
    void*       dst,        // Output buffer (16KB)
    size_t      logical)    // Logical page size (usually 16384)
{
    printf("[ENTRY-DEBUG] innodb_zip_decompress called: physical=%zu, logical=%zu\n", physical, logical);
    fflush(stdout);
    
    if (!src || !dst) {
        printf("[ENTRY-DEBUG] Invalid pointers\n");
        fflush(stdout);
        return -1;
    }
    
    if (logical != 16384) {
        // InnoDB logical pages are always 16KB
        return -2;
    }
    
    // Oracle engineers' fix #3: Set InnoDB globals correctly for compression
    // srv_page_size and srv_page_size_shift must reflect LOGICAL size (16KB)
    srv_page_size = static_cast<unsigned long>(logical);
    srv_page_size_shift = static_cast<unsigned long>(ilog2_ul(static_cast<unsigned long>(logical)));
    
    // univ_page_size must be set with correct logical/physical split for compressed tablespaces
    bool is_compressed = (physical < logical);
    univ_page_size.copy_from(page_size_t(logical, physical, is_compressed));
    
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
    
    // Follow working Percona approach: decompress INDEX pages in compressed tablespaces
    const uint16_t page_type = read_uint16_be(page_data + FIL_PAGE_TYPE_OFFSET);
    
    printf("[PAGE-DEBUG] Page type: %u (FIL_PAGE_INDEX=%u)\n", page_type, FIL_PAGE_INDEX);
    fflush(stdout);
    
    // Only attempt to decompress if it's a real index page (like working Percona code)
    if (page_type != FIL_PAGE_INDEX) {
        // Not an index page => just copy the raw page data  
        printf("[PAGE-DEBUG] Not FIL_PAGE_INDEX, copying raw data\n");
        fflush(stdout);
        memcpy(dst, src, physical);
        return 0;
    }
    
    printf("[PAGE-DEBUG] FIL_PAGE_INDEX detected - attempting decompression\n");
    fflush(stdout);
    
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
    // Correct: pass the start-of-page (including FIL header)
    page_zip.data = reinterpret_cast<page_zip_t*>(const_cast<void*>(src));
    // Correct: ssize is the ZIP exponent: 1 << (10 + ssize) == physical
    page_zip.ssize = zip_shift_from_bytes(physical);  // 8 KiB -> 3
    
    // Validate that we got a sensible ZIP ssize (0 is valid for 1KB with ZIP formula)
    if (page_zip.ssize > 4) {
        // Invalid ZIP ssize for compression  
        free(temp);
        return -3;
    }
    
    // Percona engineers' sanity check: verify ZIP ssize calculation is correct
    size_t expected_physical = static_cast<size_t>(1u) << (10 + page_zip.ssize);
    if (expected_physical != physical) {
        fprintf(stderr, "[ASSERT] ZIP ssize mismatch: expect=%zu actual_physical=%zu (ssize=%u)\n",
                expected_physical, physical, page_zip.ssize);
        free(temp);
        return -4;
    }
    
    // Oracle engineers' quick sanity checks before calling decompressor
    uint16_t type = read_uint16_be(static_cast<const unsigned char*>(src) + FIL_PAGE_TYPE_OFFSET);
    fprintf(stderr, "page_type=%u (expect 17855 for INDEX), ssize=%u (8KB should be 3)\n",
            (unsigned)type, (unsigned)page_zip.ssize);
    fflush(stderr);
    
    // Debug output to verify Oracle fixes are applied
    printf("[ORACLE-DEBUG] Before decompression:\n");
    printf("[PERCONA-DEBUG]   page_zip.data = %p (should be src, not src+38)\n", page_zip.data);
    printf("[PERCONA-DEBUG]   page_zip.ssize = %u (should be 3 for 8KB)\n", page_zip.ssize);
    printf("[ORACLE-DEBUG]   srv_page_size = %lu (should be 16384)\n", srv_page_size);
    printf("[ORACLE-DEBUG]   univ_page_size logical/physical = %lu/%lu\n", 
           univ_page_size.logical(), univ_page_size.physical());
    fflush(stdout);
    
    // Attempt decompression using InnoDB's low-level function
    // This follows the exact pattern used by Percona's successful implementation
    bool ok = page_zip_decompress_low(&page_zip, aligned_temp, true);
    
    printf("[ORACLE-DEBUG] page_zip_decompress_low returned: %s\n", ok ? "SUCCESS" : "FAILED");
    fflush(stdout);
    
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
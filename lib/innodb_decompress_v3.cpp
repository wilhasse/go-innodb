// innodb_decompress_v3.cpp - Simplified InnoDB page decompression
// Based on the working percona-parser implementation

#include <cstring>
#include <cstdlib>
#include <cstdio>
#include <cstdint>
#include <algorithm>

#include "innodb_decompress.h"

// Version string
#define INNODB_DECOMPRESS_VERSION "3.0.0"

// InnoDB page constants (from fil0fil.h)
#define FIL_PAGE_OFFSET 4
#define FIL_PAGE_TYPE 24
#define FIL_PAGE_DATA 38
#define FIL_PAGE_ARCH_LOG_NO_OR_SPACE_ID 34
#define FIL_PAGE_INDEX 17855
#define FIL_PAGE_COMPRESSED 14
#define FIL_PAGE_COMPRESSED_AND_ENCRYPTED 16
#define UNIV_PAGE_SIZE 16384
#define UNIV_ZIP_SIZE_MIN 1024

// Page types for compression
typedef void page_zip_t;

// The actual page_zip_des_t structure from InnoDB
// Based on analysis of the working code and ABI requirements
struct page_zip_des_t {
    page_zip_t* data;     // Compressed page data pointer
    uint16_t    m_start;  // Start offset of modification log
    uint16_t    m_end;    // End offset of modification log  
    uint16_t    m_nonempty; // TRUE if modification log not empty
    uint8_t     n_blobs;  // Number of externally stored BLOBs
    uint8_t     ssize;    // Shift size: 0=16KB, 1=1KB, 2=2KB, 3=4KB, 4=8KB
    // Additional padding may be needed depending on MySQL version
};

// Forward declaration - this is in libinnodb_zipdecompress.a
// Note: This is a C++ function, not extern "C"
extern bool page_zip_decompress_low(
    page_zip_des_t* page_zip,
    unsigned char* page,
    bool all
);

// Global page size settings are defined in mysql_stubs.cpp

// Read 2-byte big-endian
static inline uint16_t mach_read_from_2(const unsigned char* ptr) {
    return (uint16_t)(ptr[0] << 8 | ptr[1]);
}

// Read 4-byte big-endian
static inline uint32_t mach_read_from_4(const unsigned char* ptr) {
    return (uint32_t)(ptr[0] << 24 | ptr[1] << 16 | ptr[2] << 8 | ptr[3]);
}

// Initialize page_zip_des_t structure
static void page_zip_des_init(page_zip_des_t* page_zip) {
    memset(page_zip, 0, sizeof(page_zip_des_t));
}

// Convert page size to shift size (ssize)
// Based on percona-parser's working implementation
static uint8_t page_size_to_ssize(size_t physical_size) {
    switch(physical_size) {
        case 1024:  return 1;  // 2^10 = 1KB
        case 2048:  return 2;  // 2^11 = 2KB
        case 4096:  return 3;  // 2^12 = 4KB
        case 8192:  return 4;  // 2^13 = 8KB
        case 16384: return 0;  // Uncompressed
        default:    return 0;
    }
}

// Align pointer to boundary
static inline void* ut_align(void* ptr, size_t align) {
    return (void*)(((uintptr_t)ptr + (align - 1)) & ~(align - 1));
}

// Static helper: Validate that compressed size is valid
static bool is_valid_compressed_size(size_t size) {
    return (size == 1024 || size == 2048 || size == 4096 || size == 8192);
}

// Static helper: Detect if page is likely compressed
static bool detect_compressed_page(const unsigned char* data, size_t size) {
    // If size is less than 16KB, it's likely compressed
    if (size < UNIV_PAGE_SIZE) {
        return is_valid_compressed_size(size);
    }
    
    // Check page type for compression indicators
    if (size >= FIL_PAGE_TYPE + 2) {
        uint16_t page_type = mach_read_from_2(data + FIL_PAGE_TYPE);
        return (page_type == FIL_PAGE_COMPRESSED || 
                page_type == FIL_PAGE_COMPRESSED_AND_ENCRYPTED);
    }
    
    return false;
}

// Implementation of public API functions

extern "C" int innodb_is_page_compressed(const unsigned char* page_data, size_t page_size) {
    if (!page_data || page_size < FIL_PAGE_DATA) {
        return -1;
    }
    
    return detect_compressed_page(page_data, page_size) ? 1 : 0;
}

extern "C" int innodb_get_page_info(const unsigned char* page_data, size_t page_size,
                                    innodb_page_info_t* info) {
    if (!page_data || !info || page_size < FIL_PAGE_DATA) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    memset(info, 0, sizeof(innodb_page_info_t));
    
    info->page_number = mach_read_from_4(page_data + FIL_PAGE_OFFSET);
    info->page_type = mach_read_from_2(page_data + FIL_PAGE_TYPE);
    info->space_id = mach_read_from_4(page_data + FIL_PAGE_ARCH_LOG_NO_OR_SPACE_ID);
    
    info->is_compressed = detect_compressed_page(page_data, page_size);
    info->physical_size = page_size;
    info->logical_size = info->is_compressed ? UNIV_PAGE_SIZE : page_size;
    
    return INNODB_DECOMPRESS_SUCCESS;
}

extern "C" int innodb_decompress_page(
    const unsigned char* compressed_data,
    size_t compressed_size,
    unsigned char* output_buffer,
    size_t output_size,
    size_t* bytes_written)
{
    if (!compressed_data || !output_buffer || !bytes_written) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    if (output_size < UNIV_PAGE_SIZE) {
        return INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL;
    }
    
    if (!is_valid_compressed_size(compressed_size)) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    if (compressed_size < FIL_PAGE_TYPE + 2) {
        return INNODB_DECOMPRESS_ERROR_INVALID_PAGE;
    }
    
    uint16_t page_type = mach_read_from_2(compressed_data + FIL_PAGE_TYPE);
    
    // Only INDEX pages use zip decompression
    if (page_type != FIL_PAGE_INDEX) {
        // For non-index pages, just copy
        size_t copy_size = std::min(compressed_size, output_size);
        memcpy(output_buffer, compressed_data, copy_size);
        *bytes_written = copy_size;
        return INNODB_DECOMPRESS_SUCCESS;
    }
    
    // Allocate aligned temporary buffer (following percona-parser pattern)
    size_t alloc_size = 2 * UNIV_PAGE_SIZE;
    unsigned char* temp = (unsigned char*)malloc(alloc_size);
    if (!temp) {
        return INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL;
    }
    
    unsigned char* aligned_temp = (unsigned char*)ut_align(temp, UNIV_PAGE_SIZE);
    memset(aligned_temp, 0, UNIV_PAGE_SIZE);
    
    // Setup page_zip descriptor (following percona-parser exactly)
    page_zip_des_t page_zip;
    page_zip_des_init(&page_zip);
    
    // Set compressed data pointer (cast as in percona-parser)
    page_zip.data = reinterpret_cast<page_zip_t*>(const_cast<unsigned char*>(compressed_data));
    
    // Set the shift size (critical for decompression to work)
    page_zip.ssize = page_size_to_ssize(compressed_size);
    
    // Attempt decompression
    bool success = page_zip_decompress_low(&page_zip, aligned_temp, true);
    
    if (!success) {
        free(temp);
        return INNODB_DECOMPRESS_ERROR_DECOMPRESS_FAILED;
    }
    
    // Copy decompressed data to output
    memcpy(output_buffer, aligned_temp, UNIV_PAGE_SIZE);
    *bytes_written = UNIV_PAGE_SIZE;
    
    free(temp);
    return INNODB_DECOMPRESS_SUCCESS;
}

extern "C" int innodb_process_page(
    const unsigned char* input_data,
    size_t input_size,
    unsigned char* output_buffer,
    size_t output_size,
    size_t* bytes_written)
{
    if (!input_data || !output_buffer || !bytes_written) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    if (output_size < UNIV_PAGE_SIZE) {
        return INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL;
    }
    
    bool is_compressed = detect_compressed_page(input_data, input_size);
    
    if (!is_compressed) {
        size_t copy_size = std::min(input_size, output_size);
        memcpy(output_buffer, input_data, copy_size);
        *bytes_written = copy_size;
        return INNODB_DECOMPRESS_SUCCESS;
    }
    
    return innodb_decompress_page(input_data, input_size, output_buffer, 
                                  output_size, bytes_written);
}

extern "C" const char* innodb_decompress_error_string(int error_code) {
    switch (error_code) {
        case INNODB_DECOMPRESS_SUCCESS:
            return "Success";
        case INNODB_DECOMPRESS_ERROR_INVALID_SIZE:
            return "Invalid page size";
        case INNODB_DECOMPRESS_ERROR_NOT_COMPRESSED:
            return "Page is not compressed";
        case INNODB_DECOMPRESS_ERROR_DECOMPRESS_FAILED:
            return "Decompression failed";
        case INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL:
            return "Output buffer too small";
        case INNODB_DECOMPRESS_ERROR_INVALID_PAGE:
            return "Invalid page format";
        default:
            return "Unknown error";
    }
}

extern "C" const char* innodb_decompress_version(void) {
    return INNODB_DECOMPRESS_VERSION;
}
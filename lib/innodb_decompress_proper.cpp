// innodb_decompress_proper.cpp - InnoDB page decompression using real headers
// This version uses the actual InnoDB headers to ensure ABI compatibility

// Include the real InnoDB headers from MySQL/Percona source
// These define the actual structures and functions we need
#include "univ.i"
#include "fil0fil.h"
#include "mach0data.h"
#include "page0page.h"
#include "page0size.h"
#include "page0types.h"
#include "page0zip.h"   // Defines page_zip_t, page_zip_des_t, page_zip_des_init()
#include "ut0byte.h"

#include <cstring>
#include <cstdlib>
#include <cstdio>
#include <cstdint>
#include <algorithm>

#include "innodb_decompress.h"

// Version string
#define INNODB_DECOMPRESS_VERSION "2.0.0"

// The decompression function is in libinnodb_zipdecompress.a
// It's already declared in page0zip.h with proper signature
// extern bool page_zip_decompress_low(page_zip_des_t* page_zip, byte* page, bool all);

// Global page size settings are defined in mysql_stubs.cpp

// Static helper: Validate that compressed size is valid
static bool is_valid_compressed_size(size_t size) {
    return (size == 1024 || size == 2048 || size == 4096 || size == 8192);
}

// Static helper: Convert physical size to ssize
static unsigned int size_to_ssize(size_t physical_size) {
    switch(physical_size) {
        case 1024:  return 1;  // 2^10 = 1KB
        case 2048:  return 2;  // 2^11 = 2KB  
        case 4096:  return 3;  // 2^12 = 4KB
        case 8192:  return 4;  // 2^13 = 8KB
        case 16384: return 0;  // Uncompressed
        default:    return 0;
    }
}

// Static helper: Detect if page is likely compressed based on size and header
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
        return -1;  // Invalid input
    }
    
    return detect_compressed_page(page_data, page_size) ? 1 : 0;
}

extern "C" int innodb_get_page_info(const unsigned char* page_data, size_t page_size,
                                    innodb_page_info_t* info) {
    if (!page_data || !info || page_size < FIL_PAGE_DATA) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    // Clear the info structure
    memset(info, 0, sizeof(innodb_page_info_t));
    
    // Extract page header information using InnoDB macros
    info->page_number = mach_read_from_4(page_data + FIL_PAGE_OFFSET);
    info->page_type = mach_read_from_2(page_data + FIL_PAGE_TYPE);
    info->space_id = mach_read_from_4(page_data + FIL_PAGE_ARCH_LOG_NO_OR_SPACE_ID);
    
    // Determine if compressed
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
    // Validate inputs
    if (!compressed_data || !output_buffer || !bytes_written) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    if (output_size < UNIV_PAGE_SIZE) {
        return INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL;
    }
    
    // Check if this is actually a compressed page
    if (!is_valid_compressed_size(compressed_size)) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    // Get page type
    if (compressed_size < FIL_PAGE_TYPE + 2) {
        return INNODB_DECOMPRESS_ERROR_INVALID_PAGE;
    }
    
    uint16_t page_type = mach_read_from_2(compressed_data + FIL_PAGE_TYPE);
    
    // Only INDEX pages use the zip decompression
    if (page_type != FIL_PAGE_INDEX) {
        // For non-index pages, just copy
        size_t copy_size = std::min(compressed_size, output_size);
        memcpy(output_buffer, compressed_data, copy_size);
        *bytes_written = copy_size;
        return INNODB_DECOMPRESS_SUCCESS;
    }
    
    // Allocate aligned temporary buffer for decompression
    // Using ut_align for proper alignment as InnoDB expects
    size_t alloc_size = 2 * UNIV_PAGE_SIZE;
    byte* temp_buffer = static_cast<byte*>(ut::malloc_withkey(alloc_size));
    if (!temp_buffer) {
        return INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL;
    }
    
    // Align the buffer to page size boundary using InnoDB's ut_align
    byte* aligned_buffer = static_cast<byte*>(ut_align(temp_buffer, UNIV_PAGE_SIZE));
    memset(aligned_buffer, 0, UNIV_PAGE_SIZE);
    
    // Setup page_zip descriptor using real InnoDB structure
    page_zip_des_t page_zip;
    page_zip_des_init(&page_zip);  // Use real InnoDB init function
    
    // Set compressed data pointer and size
    // Note: page_zip.data is page_zip_t* in real structure
    page_zip.data = const_cast<page_zip_t*>(reinterpret_cast<const page_zip_t*>(compressed_data));
    
    // Set the ssize based on compressed size
    // In InnoDB: ssize 0=16KB, 1=1KB, 2=2KB, 3=4KB, 4=8KB
    page_zip.ssize = size_to_ssize(compressed_size);
    
    // Create page_size_t for the compressed page
    page_size_t page_size(compressed_size, UNIV_PAGE_SIZE, true);
    
    // Attempt decompression using real InnoDB function
    bool success = page_zip_decompress_low(&page_zip, aligned_buffer, true);
    
    if (!success) {
        ut::free(temp_buffer);
        return INNODB_DECOMPRESS_ERROR_DECOMPRESS_FAILED;
    }
    
    // Copy decompressed data to output buffer
    memcpy(output_buffer, aligned_buffer, UNIV_PAGE_SIZE);
    *bytes_written = UNIV_PAGE_SIZE;
    
    ut::free(temp_buffer);
    return INNODB_DECOMPRESS_SUCCESS;
}

extern "C" int innodb_process_page(
    const unsigned char* input_data,
    size_t input_size,
    unsigned char* output_buffer,
    size_t output_size,
    size_t* bytes_written)
{
    // Validate inputs
    if (!input_data || !output_buffer || !bytes_written) {
        return INNODB_DECOMPRESS_ERROR_INVALID_SIZE;
    }
    
    if (output_size < UNIV_PAGE_SIZE) {
        return INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL;
    }
    
    // Check if the page appears to be compressed
    bool is_compressed = detect_compressed_page(input_data, input_size);
    
    if (!is_compressed) {
        // Not compressed, just copy the data
        size_t copy_size = std::min(input_size, output_size);
        memcpy(output_buffer, input_data, copy_size);
        *bytes_written = copy_size;
        return INNODB_DECOMPRESS_SUCCESS;
    }
    
    // It's compressed, decompress it
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
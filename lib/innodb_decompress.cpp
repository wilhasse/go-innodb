// innodb_decompress.cpp - InnoDB page decompression implementation
// Minimal implementation without MySQL dependencies
// Links against libinnodb_zipdecompress.a

#include <cstring>
#include <cstdlib>
#include <cstdio>
#include <cstdint>
#include <algorithm>

#include "innodb_decompress.h"
#include "innodb_constants.h"

// Version string
#define INNODB_DECOMPRESS_VERSION "1.0.0"

// Forward declaration for the function in libinnodb_zipdecompress.a
// This function is implemented in the MySQL source and we link against it
// Note: This is a C++ function, not extern "C"
extern bool page_zip_decompress_low(
    page_zip_des_t* page_zip,  // Compression descriptor
    unsigned char* page,        // Output buffer for decompressed page
    bool all                    // Decompress all (always true for our use)
);

// Global page size settings are defined in mysql_stubs.cpp
// to avoid duplicate symbol errors

// Static helper: Validate that compressed size is valid
static bool is_valid_compressed_size(size_t size) {
    return (size == 1024 || size == 2048 || size == 4096 || size == 8192);
}

// Static helper: Detect if page is likely compressed based on size and header
static bool detect_compressed_page(const unsigned char* data, size_t size) {
    // If size is less than 16KB, it's likely compressed
    if (size < UNIV_PAGE_SIZE) {
        return is_valid_compressed_size(size);
    }
    
    // Check page type for compression indicators
    if (size >= FIL_PAGE_TYPE + 2) {
        uint16_t page_type = MACH_READ_2(data + FIL_PAGE_TYPE);
        return is_compressed_page_type(page_type);
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
    
    // Extract page header information
    info->page_number = MACH_READ_4(page_data + FIL_PAGE_OFFSET);
    info->page_type = MACH_READ_2(page_data + FIL_PAGE_TYPE);
    info->space_id = MACH_READ_4(page_data + FIL_PAGE_SPACE_ID);
    
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
    
    uint16_t page_type = MACH_READ_2(compressed_data + FIL_PAGE_TYPE);
    
    // Only INDEX pages use the zip decompression
    // Other compressed page types might use different compression
    if (page_type != FIL_PAGE_INDEX) {
        // For non-index pages, check if it's marked as compressed
        if (!is_compressed_page_type(page_type)) {
            // Not compressed, just copy
            size_t copy_size = std::min(compressed_size, output_size);
            memcpy(output_buffer, compressed_data, copy_size);
            *bytes_written = copy_size;
            return INNODB_DECOMPRESS_SUCCESS;
        }
        
        // It's a compressed non-index page - we might not support this
        // For now, copy as-is
        size_t copy_size = std::min(compressed_size, output_size);
        memcpy(output_buffer, compressed_data, copy_size);
        *bytes_written = copy_size;
        return INNODB_DECOMPRESS_SUCCESS;
    }
    
    // Allocate aligned temporary buffer for decompression
    // The decompression function requires aligned memory
    size_t alloc_size = 2 * UNIV_PAGE_SIZE;
    unsigned char* temp_buffer = (unsigned char*)malloc(alloc_size);
    if (!temp_buffer) {
        return INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL;
    }
    
    // Align the buffer to page size boundary
    unsigned char* aligned_buffer = (unsigned char*)UT_ALIGN(temp_buffer, UNIV_PAGE_SIZE);
    memset(aligned_buffer, 0, UNIV_PAGE_SIZE);
    
    // Setup page_zip descriptor
    page_zip_des_t page_zip;
    PAGE_ZIP_DES_INIT(&page_zip);
    
    // Set compressed data pointer and size
    page_zip.data = const_cast<void*>(static_cast<const void*>(compressed_data));
    page_zip.ssize = physical_size_to_ssize(compressed_size);
    
    // Attempt decompression
    bool success = page_zip_decompress_low(&page_zip, aligned_buffer, true);
    
    if (!success) {
        free(temp_buffer);
        return INNODB_DECOMPRESS_ERROR_DECOMPRESS_FAILED;
    }
    
    // Copy decompressed data to output buffer
    memcpy(output_buffer, aligned_buffer, UNIV_PAGE_SIZE);
    *bytes_written = UNIV_PAGE_SIZE;
    
    free(temp_buffer);
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
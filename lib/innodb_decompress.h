// innodb_decompress.h - C interface for InnoDB page decompression
// This provides a clean C API that can be called from Go via CGo
// No MySQL dependencies required

#ifndef INNODB_DECOMPRESS_H
#define INNODB_DECOMPRESS_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>
#include <stdint.h>

// Return codes
#define INNODB_DECOMPRESS_SUCCESS           0
#define INNODB_DECOMPRESS_ERROR_INVALID_SIZE -1
#define INNODB_DECOMPRESS_ERROR_NOT_COMPRESSED -2
#define INNODB_DECOMPRESS_ERROR_DECOMPRESS_FAILED -3
#define INNODB_DECOMPRESS_ERROR_BUFFER_TOO_SMALL -4
#define INNODB_DECOMPRESS_ERROR_INVALID_PAGE -5

// Page information structure
typedef struct {
    uint32_t page_number;      // Page number from header
    uint16_t page_type;        // Page type (FIL_PAGE_INDEX, etc.)
    uint32_t space_id;         // Tablespace ID
    int      is_compressed;    // 1 if compressed, 0 if not
    size_t   physical_size;    // Size on disk (1KB, 2KB, 4KB, 8KB, or 16KB)
    size_t   logical_size;     // Always 16KB when uncompressed
} innodb_page_info_t;

/**
 * Check if a page appears to be compressed.
 * 
 * @param page_data  Pointer to page data
 * @param page_size  Size of the page data in bytes
 * @return 1 if compressed, 0 if not, -1 on error
 */
int innodb_is_page_compressed(const unsigned char* page_data, size_t page_size);

/**
 * Get information about an InnoDB page.
 * 
 * @param page_data  Pointer to page data
 * @param page_size  Size of the page data in bytes
 * @param info       Output: Page information structure
 * @return 0 on success, negative error code on failure
 */
int innodb_get_page_info(const unsigned char* page_data, size_t page_size, 
                         innodb_page_info_t* info);

/**
 * Decompress an InnoDB compressed page.
 * 
 * This is the main decompression function. It detects whether the page
 * is compressed and decompresses it if necessary.
 * 
 * @param compressed_data  Input: Compressed page data
 * @param compressed_size  Input: Size of compressed data (1KB, 2KB, 4KB, or 8KB)
 * @param output_buffer    Output: Buffer for decompressed data
 * @param output_size      Input: Size of output buffer (must be >= 16KB)
 * @param bytes_written    Output: Actual bytes written to output buffer
 * @return 0 on success, negative error code on failure
 */
int innodb_decompress_page(
    const unsigned char* compressed_data,
    size_t compressed_size,
    unsigned char* output_buffer,
    size_t output_size,
    size_t* bytes_written
);

/**
 * Process a page that might be compressed or uncompressed.
 * This function handles both cases automatically.
 * 
 * @param input_data     Input: Page data (compressed or uncompressed)
 * @param input_size     Input: Size of input data
 * @param output_buffer  Output: Buffer for processed data
 * @param output_size    Input: Size of output buffer (must be >= 16KB)
 * @param bytes_written  Output: Actual bytes written to output buffer
 * @return 0 on success, negative error code on failure
 */
int innodb_process_page(
    const unsigned char* input_data,
    size_t input_size,
    unsigned char* output_buffer,
    size_t output_size,
    size_t* bytes_written
);

/**
 * Get a string description of an error code.
 * 
 * @param error_code  Error code returned by other functions
 * @return String description of the error
 */
const char* innodb_decompress_error_string(int error_code);

/**
 * Get the library version.
 * 
 * @return Version string
 */
const char* innodb_decompress_version(void);

#ifdef __cplusplus
}
#endif

#endif // INNODB_DECOMPRESS_H
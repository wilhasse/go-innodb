// innodb_constants.h - Minimal InnoDB constants for decompression
// Extracted from MySQL/InnoDB source to avoid dependencies
// Only includes what's needed for page decompression

#ifndef INNODB_CONSTANTS_H
#define INNODB_CONSTANTS_H

#include <stdint.h>
#include <stddef.h>

// ============================================================================
// Page Size Constants
// ============================================================================

// Default InnoDB page size
#define UNIV_PAGE_SIZE_ORIG 16384
#define UNIV_PAGE_SIZE      16384

// Compressed page sizes
#define UNIV_ZIP_SIZE_MIN   1024   // Minimum compressed page size (1KB)
#define UNIV_ZIP_SIZE_MAX   16384  // Maximum page size

// Valid compressed sizes: 1KB, 2KB, 4KB, 8KB
// Shift sizes: 10=1KB, 11=2KB, 12=4KB, 13=8KB

// ============================================================================
// FIL Header Offsets (First 38 bytes of every page)
// ============================================================================

#define FIL_PAGE_SPACE_OR_CHKSUM  0    // Checksum or space id
#define FIL_PAGE_OFFSET            4    // Page number
#define FIL_PAGE_PREV              8    // Previous page in list
#define FIL_PAGE_NEXT             12    // Next page in list  
#define FIL_PAGE_LSN              16    // LSN of page's latest log record
#define FIL_PAGE_TYPE             24    // Page type (2 bytes)
#define FIL_PAGE_FILE_FLUSH_LSN   26    // Flushed LSN (only in space 0, page 0)
#define FIL_PAGE_SPACE_ID         34    // Space ID (4 bytes)
#define FIL_PAGE_DATA             38    // Start of page data

// FIL Trailer (last 8 bytes)
#define FIL_PAGE_END_LSN_OLD_CHKSUM 8  // Size of FIL trailer

// ============================================================================
// Page Types (value at offset FIL_PAGE_TYPE)
// ============================================================================

#define FIL_PAGE_INDEX                17855   // B-tree index page
#define FIL_PAGE_RTREE                17854   // R-tree index page  
#define FIL_PAGE_UNDO_LOG             2       // Undo log page
#define FIL_PAGE_INODE                3       // File segment inode
#define FIL_PAGE_IBUF_FREE_LIST       4       // Insert buffer free list
#define FIL_PAGE_TYPE_ALLOCATED       0       // Freshly allocated
#define FIL_PAGE_IBUF_BITMAP          5       // Insert buffer bitmap
#define FIL_PAGE_TYPE_SYS             6       // System page
#define FIL_PAGE_TYPE_TRX_SYS         7       // Transaction system
#define FIL_PAGE_TYPE_FSP_HDR         8       // File space header
#define FIL_PAGE_TYPE_XDES            9       // Extent descriptor
#define FIL_PAGE_TYPE_BLOB            10      // Uncompressed BLOB
#define FIL_PAGE_TYPE_ZBLOB           11      // Compressed BLOB
#define FIL_PAGE_TYPE_ZBLOB2          12      // Compressed BLOB 2
#define FIL_PAGE_COMPRESSED           14      // Compressed page
#define FIL_PAGE_ENCRYPTED            15      // Encrypted page
#define FIL_PAGE_COMPRESSED_AND_ENCRYPTED 16  // Compressed and encrypted
#define FIL_PAGE_ENCRYPTED_RTREE      17      // Encrypted R-tree
#define FIL_PAGE_SDI                 18      // Serialized dictionary information
#define FIL_PAGE_SDI_ZBLOB            19      // Compressed SDI BLOB
#define FIL_PAGE_SDI_BLOB             20      // Uncompressed SDI BLOB

// ============================================================================
// FSP Header Constants (for page size detection)
// ============================================================================

#define FSP_HEADER_OFFSET       38     // Offset of FSP header within page
#define FSP_SPACE_FLAGS         16     // Offset of flags within FSP header
#define FSP_FLAGS_WIDTH         32     // Number of bits in flags

// Flags positions for page size
#define FSP_FLAGS_POS_PAGE_SSIZE     6
#define FSP_FLAGS_MASK_PAGE_SSIZE    0xf
#define FSP_FLAGS_WIDTH_PAGE_SSIZE   4

// Extract page size from flags
#define FSP_FLAGS_GET_PAGE_SSIZE(flags) \
    (((flags) >> FSP_FLAGS_POS_PAGE_SSIZE) & FSP_FLAGS_MASK_PAGE_SSIZE)

// ============================================================================
// Utility Macros
// ============================================================================

// Read 2 bytes in big-endian (replaces mach_read_from_2)
#define MACH_READ_2(ptr) \
    ((uint16_t)(((uint8_t*)(ptr))[0] << 8 | ((uint8_t*)(ptr))[1]))

// Read 4 bytes in big-endian (replaces mach_read_from_4)
#define MACH_READ_4(ptr) \
    ((uint32_t)(((uint8_t*)(ptr))[0] << 24 | \
                ((uint8_t*)(ptr))[1] << 16 | \
                ((uint8_t*)(ptr))[2] << 8  | \
                ((uint8_t*)(ptr))[3]))

// Align pointer to boundary (replaces ut_align)
#define UT_ALIGN(ptr, align) \
    ((void*)(((uintptr_t)(ptr) + ((align) - 1)) & ~((uintptr_t)((align) - 1))))

// ============================================================================
// Page Compression - Use real InnoDB headers instead
// ============================================================================
// The page_zip_des_t structure and related functions are defined in the
// actual InnoDB headers (page0zip.h) to ensure ABI compatibility

// Convert physical size to shift size (replaces page_size_to_ssize)
static inline uint32_t physical_size_to_ssize(size_t size) {
    switch(size) {
        case 1024:  return 10;  // 1KB = 2^10
        case 2048:  return 11;  // 2KB = 2^11
        case 4096:  return 12;  // 4KB = 2^12
        case 8192:  return 13;  // 8KB = 2^13
        case 16384: return 0;   // 16KB uncompressed
        default:    return 0;   // Unknown/invalid
    }
}

// Convert shift size to physical size
static inline size_t ssize_to_physical_size(uint32_t ssize) {
    if (ssize == 0) return 16384;  // Uncompressed
    if (ssize < 10 || ssize > 13) return 0;  // Invalid
    return (1UL << ssize);  // 2^ssize
}

// Check if page appears to be compressed based on type
static inline int is_compressed_page_type(uint16_t page_type) {
    return (page_type == FIL_PAGE_COMPRESSED || 
            page_type == FIL_PAGE_COMPRESSED_AND_ENCRYPTED);
}

// Simple checksum validation (very basic)
static inline int validate_page_header(const unsigned char* page) {
    // Check that page number is not ridiculously large
    uint32_t page_no = MACH_READ_4(page + FIL_PAGE_OFFSET);
    if (page_no > 0xFFFFFFF0) return 0;  // Likely invalid
    
    // Check page type is known
    uint16_t page_type = MACH_READ_2(page + FIL_PAGE_TYPE);
    if (page_type > 100 && page_type < 17000) return 0;  // Unknown type range
    
    return 1;  // Seems valid
}

#endif // INNODB_CONSTANTS_H
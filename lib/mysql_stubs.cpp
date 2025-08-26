// mysql_stubs.cpp - Stub implementations for MySQL/InnoDB symbols
// These are required by libinnodb_zipdecompress.a but not actually used
// for decompression operations

#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <iostream>
#include <cstring>

// Global variables expected by InnoDB code
extern "C" {
    // Page size globals - we always use 16KB
    size_t srv_page_size = 16384;
    unsigned srv_page_size_shift = 14;  // 2^14 = 16384
}

// InnoDB logging/error namespace
namespace ib {
    
    // Base logger class
    class logger {
    public:
        logger() {}
        virtual ~logger();  // Declared in header, defined later
    };
    
    // Error logger  
    class error : public logger {
    public:
        error() {}
        virtual ~error();  // Declared in header, defined later
        
        template<typename T>
        error& operator<<(const T& val) {
            std::cerr << val;
            return *this;
        }
    };
    
    // Warning logger  
    class warn : public logger {
    public:
        warn() {}
        virtual ~warn();  // Declared in header, defined later
        
        template<typename T>
        warn& operator<<(const T& val) {
            std::cerr << val;
            return *this;
        }
    };
    
    // Fatal logger
    class fatal : public logger {
    public:
        fatal() {}
        virtual ~fatal();  // Declared in header, defined later
        
        template<typename T>
        fatal& operator<<(const T& val) {
            std::cerr << val;
            return *this;
        }
    };
    
    // Now define the destructors outside the class (this ensures vtables)
    logger::~logger() {}
    error::~error() {}
    warn::~warn() {}
    fatal::~fatal() {
        std::cerr << " [FATAL]" << std::endl;
    }
}

// Debug assertion function - C++ linkage (not extern "C")
void ut_dbg_assertion_failed(
    const char* expr,
    const char* file,
    unsigned long line)
{
    std::cerr << "Assertion failed: " << expr 
              << " at " << file << ":" << line << std::endl;
    // Don't abort in our stub
}

// Additional stubs that might be needed
extern "C" {
    // Memory allocation hooks
    void* ut_malloc_nokey(size_t size) {
        return malloc(size);
    }
    
    void ut_free(void* ptr) {
        free(ptr);
    }
}
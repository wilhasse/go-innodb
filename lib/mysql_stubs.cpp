// mysql_stubs.cpp - Stub implementations for MySQL/InnoDB symbols
// Following Oracle engineers' guidance for proper InnoDB integration
// These stubs provide the missing symbols expected by libinnodb_zipdecompress.a

#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <cstdio>
#include <cstring>
#include <iostream>
#include <sstream>

// Global variables expected by InnoDB code
// Note: Using unsigned long instead of ulong (which requires my_config.h)
extern "C" {
    // Page size globals - updated at runtime for consistency
    unsigned long srv_page_size = 16384;       // default logical page size
    unsigned long srv_page_size_shift = 14;    // log2(16384)
}

// InnoDB logging/error namespace
// Following Oracle engineers' guidance for proper logger implementation
namespace ib {
    
    // Base logger class with ostringstream buffer (Oracle's approach)
    class logger {
    public:
        logger() {}
        virtual ~logger();
        
    protected:
        std::ostringstream m_oss;  // Message buffer
        
    public:
        template<typename T>
        logger& operator<<(const T& val) {
            m_oss << val;
            return *this;
        }
    };
    
    // Info logger (added for completeness)
    class info : public logger {
    public:
        info() {}
        virtual ~info();
    };
    
    // Warning logger  
    class warn : public logger {
    public:
        warn() {}
        virtual ~warn();
    };
    
    // Error logger  
    class error : public logger {
    public:
        error() {}
        virtual ~error();
    };
    
    // Fatal logger
    class fatal : public logger {
    public:
        fatal() {}
        virtual ~fatal();
    };
    
    // Define destructors outside class for proper vtable generation
    // Following Oracle's pattern with proper message formatting
    logger::~logger() {}
    
    info::~info() {
        if (!m_oss.str().empty()) {
            std::cerr << "[INFO]  zipshim: " << m_oss.str() << "\n";
        }
    }
    
    warn::~warn() {
        if (!m_oss.str().empty()) {
            std::cerr << "[WARN]  zipshim: " << m_oss.str() << "\n";
        }
    }
    
    error::~error() {
        if (!m_oss.str().empty()) {
            std::cerr << "[ERROR] zipshim: " << m_oss.str() << "\n";
        }
    }
    
    fatal::~fatal() {
        if (!m_oss.str().empty()) {
            std::cerr << "[FATAL] zipshim: " << m_oss.str() << "\n";
        }
        std::abort();  // Fatal errors should terminate
    }
}

// InnoDB assertion hook - following Oracle engineers' guidance
// Must use C++ linkage (not extern "C") and [[noreturn]] attribute
[[noreturn]] void ut_dbg_assertion_failed(const char* expr,
                                          const char* file,
                                          unsigned long line) {
    std::fprintf(stderr, "ut_dbg_assertion_failed: %s (%s:%lu)\n",
                 expr ? expr : "(null)", 
                 file ? file : "(null)", 
                 line);
    std::abort();  // Oracle's approach: always abort on assertion failure
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
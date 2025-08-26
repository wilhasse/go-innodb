// innodb_stubs.cpp - Stub implementations for InnoDB dependencies
// These are required by libinnodb_zipdecompress.a but not actually used
// in our decompression code path

#include <cstdio>
#include <cstdlib>
#include <sstream>

// InnoDB logging namespace and classes
namespace ib {

// Base logger class
class logger {
public:
    logger() {}
    virtual ~logger() {}
protected:
    std::ostringstream m_oss;
};

// Warning logger
class warn : public logger {
public:
    warn() {}
    virtual ~warn() {
        // In production, this would log a warning
        // For our use, we can ignore or print to stderr if needed
        // fprintf(stderr, "[WARN] %s\n", m_oss.str().c_str());
    }
};

// Error logger
class error : public logger {
public:
    error() {}
    virtual ~error() {
        // In production, this would log an error
        // For our use, we can ignore or print to stderr if needed
        // fprintf(stderr, "[ERROR] %s\n", m_oss.str().c_str());
    }
};

// Fatal error logger
class fatal : public logger {
public:
    fatal() {}
    [[noreturn]] virtual ~fatal() {
        // Fatal errors should abort
        fprintf(stderr, "[FATAL] %s\n", m_oss.str().c_str());
        abort();
    }
};

} // namespace ib

// Assertion failure handler (C++ mangled name)
[[noreturn]] void ut_dbg_assertion_failed(
    const char* expr,
    const char* file,
    unsigned long line)
{
    fprintf(stderr, "Assertion failed: %s at %s:%lu\n", 
            expr ? expr : "unknown", file, line);
    abort();
}

// Additional stubs that might be needed
extern "C" {
    // Global page size variables (if not already defined)
    // unsigned long srv_page_size = 16384;
    // unsigned long srv_page_size_shift = 14;
}
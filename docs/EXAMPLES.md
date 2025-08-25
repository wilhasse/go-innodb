# Usage Examples

This document provides practical examples of using the InnoDB Page Parser library and CLI tool.

## CLI Tool Examples

### Basic Page Reading

```bash
# Read the root page (page 0) of a tablespace
./go-innodb -file /var/lib/mysql/mydb/users.ibd

# Read a specific page number
./go-innodb -file /var/lib/mysql/mydb/users.ibd -page 3

# Read page with verbose output
./go-innodb -file /var/lib/mysql/mydb/users.ibd -page 3 -v
```

### Output Formats

```bash
# JSON output for programmatic processing
./go-innodb -file data.ibd -page 3 -format json

# Quick summary view
./go-innodb -file data.ibd -page 3 -format summary

# JSON with records included
./go-innodb -file data.ibd -page 3 -format json -records
```

### Record Analysis

```bash
# Show first 10 records in a page
./go-innodb -file data.ibd -page 3 -records -max-records 10

# Show all records (up to default limit of 100)
./go-innodb -file data.ibd -page 3 -records

# Verbose record information
./go-innodb -file data.ibd -page 3 -records -v
```

### Analyzing Multiple Pages

```bash
# Script to analyze first 10 pages
for i in {0..9}; do
    echo "=== Page $i ==="
    ./go-innodb -file data.ibd -page $i -format summary
done

# Find all index pages in first 100 pages
for i in {0..99}; do
    ./go-innodb -file data.ibd -page $i -format json | \
        jq -r 'select(.fil_header.page_type == 17855) | .page_number'
done
```

### Piping to Other Tools

```bash
# Extract index IDs from pages
./go-innodb -file data.ibd -page 3 -format json | \
    jq '.index_page.index_id'

# Count user records across pages
for i in {3..10}; do
    ./go-innodb -file data.ibd -page $i -format json | \
        jq -r '.index_page.user_records // 0'
done | awk '{sum+=$1} END {print sum}'

# Find pages with deleted records
for i in {0..20}; do
    ./go-innodb -file data.ibd -page $i -format json -records 2>/dev/null | \
        jq -r 'select(.index_page.records[]?.deleted == true) | .page_number' 2>/dev/null
done | sort -u
```

## Go Library Examples

### Basic Page Reading

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    goinnodb "github.com/cslog/go-innodb"
)

func main() {
    // Open InnoDB data file
    file, err := os.Open("data.ibd")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // Create page reader
    reader := goinnodb.NewPageReader(file)

    // Read page 3
    page, err := reader.ReadPage(3)
    if err != nil {
        log.Fatal(err)
    }

    // Display basic information
    fmt.Printf("Page Number: %d\n", page.PageNo)
    fmt.Printf("Page Type: %d\n", page.FIL.PageType)
    fmt.Printf("Space ID: %d\n", page.FIL.SpaceID)
    fmt.Printf("LSN: %d\n", page.FIL.LastModLSN)
}
```

### Analyzing Index Pages

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    goinnodb "github.com/cslog/go-innodb"
)

func analyzeIndexPage(filename string, pageNum uint32) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    reader := goinnodb.NewPageReader(file)
    page, err := reader.ReadPage(pageNum)
    if err != nil {
        log.Fatal(err)
    }

    // Check if it's an index page
    if page.PageType() != goinnodb.PageTypeIndex {
        log.Fatalf("Page %d is not an index page", pageNum)
    }

    // Parse as index page
    indexPage, err := goinnodb.ParseIndexPage(page)
    if err != nil {
        log.Fatal(err)
    }

    // Display index information
    fmt.Printf("Index ID: %d\n", indexPage.Hdr.IndexID)
    fmt.Printf("Page Level: %d ", indexPage.Hdr.PageLevel)
    if indexPage.IsLeaf() {
        fmt.Println("(Leaf page)")
    } else {
        fmt.Println("(Internal page)")
    }
    
    fmt.Printf("User Records: %d\n", indexPage.Hdr.NumUserRecs)
    fmt.Printf("Deleted Space: %d bytes\n", indexPage.Hdr.GarbageSpace)
    fmt.Printf("Page Fill: %.1f%%\n", 
        float64(indexPage.UsedBytes())*100/goinnodb.PageSize)

    // Check if it's the root page
    if indexPage.IsRoot() {
        fmt.Println("This is the ROOT page of the index")
    }
}

func main() {
    analyzeIndexPage("data.ibd", 3)
}
```

### Walking Through Records

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    goinnodb "github.com/cslog/go-innodb"
)

func walkPageRecords(filename string, pageNum uint32) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    reader := goinnodb.NewPageReader(file)
    page, err := reader.ReadPage(pageNum)
    if err != nil {
        log.Fatal(err)
    }

    if page.PageType() != goinnodb.PageTypeIndex {
        log.Fatal("Not an index page")
    }

    indexPage, err := goinnodb.ParseIndexPage(page)
    if err != nil {
        log.Fatal(err)
    }

    // Walk through all records (including system records)
    records, err := indexPage.WalkRecords(1000, false)
    if err != nil {
        log.Fatal(err)
    }

    for i, rec := range records {
        recordType := "USER"
        switch rec.Header.Type {
        case goinnodb.RecInfimum:
            recordType = "INFIMUM"
        case goinnodb.RecSupremum:
            recordType = "SUPREMUM"
        case goinnodb.RecNodePointer:
            recordType = "NODE_PTR"
        }

        fmt.Printf("Record %3d: Type=%-8s HeapNo=%3d Deleted=%v NextOffset=%d\n",
            i, recordType, rec.Header.HeapNumber, 
            rec.Header.FlagsDeleted, rec.Header.NextRecOffset)
    }

    // Walk through user records only
    userRecords, err := indexPage.WalkRecords(1000, true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\nTotal user records: %d\n", len(userRecords))
}

func main() {
    walkPageRecords("data.ibd", 3)
}
```

### Following Page Links

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    goinnodb "github.com/cslog/go-innodb"
)

func followPageChain(filename string, startPage uint32, maxPages int) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    reader := goinnodb.NewPageReader(file)
    currentPage := startPage
    pagesRead := 0

    fmt.Println("Following page chain:")
    
    for pagesRead < maxPages {
        page, err := reader.ReadPage(currentPage)
        if err != nil {
            log.Printf("Error reading page %d: %v", currentPage, err)
            break
        }

        fmt.Printf("Page %d -> ", currentPage)
        
        // If it's an index page, show record count
        if page.PageType() == goinnodb.PageTypeIndex {
            if idx, err := goinnodb.ParseIndexPage(page); err == nil {
                fmt.Printf("(Records: %d) -> ", idx.Hdr.NumUserRecs)
            }
        }

        // Follow the next pointer
        if page.FIL.Next == nil {
            fmt.Println("END")
            break
        }

        currentPage = *page.FIL.Next
        pagesRead++
    }

    fmt.Printf("Total pages in chain: %d\n", pagesRead+1)
}

func main() {
    // Follow leaf page chain starting from page 3
    followPageChain("data.ibd", 3, 100)
}
```

### Finding Deleted Records

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    goinnodb "github.com/cslog/go-innodb"
)

func findDeletedRecords(filename string, startPage, endPage uint32) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    reader := goinnodb.NewPageReader(file)
    totalDeleted := 0
    totalGarbage := 0

    for pageNum := startPage; pageNum <= endPage; pageNum++ {
        page, err := reader.ReadPage(pageNum)
        if err != nil {
            continue // Skip unreadable pages
        }

        if page.PageType() != goinnodb.PageTypeIndex {
            continue // Skip non-index pages
        }

        indexPage, err := goinnodb.ParseIndexPage(page)
        if err != nil {
            continue
        }

        // Check garbage space
        if indexPage.Hdr.GarbageSpace > 0 {
            fmt.Printf("Page %d: %d bytes of garbage space\n", 
                pageNum, indexPage.Hdr.GarbageSpace)
            totalGarbage += int(indexPage.Hdr.GarbageSpace)
        }

        // Walk records to find deleted ones
        records, err := indexPage.WalkRecords(1000, true)
        if err != nil {
            continue
        }

        deletedInPage := 0
        for _, rec := range records {
            if rec.Header.FlagsDeleted {
                deletedInPage++
                totalDeleted++
            }
        }

        if deletedInPage > 0 {
            fmt.Printf("Page %d: %d deleted records\n", 
                pageNum, deletedInPage)
        }
    }

    fmt.Printf("\nSummary:\n")
    fmt.Printf("Total deleted records: %d\n", totalDeleted)
    fmt.Printf("Total garbage space: %d bytes\n", totalGarbage)
}

func main() {
    findDeletedRecords("data.ibd", 0, 100)
}
```

### Page Statistics Collector

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    goinnodb "github.com/cslog/go-innodb"
)

type PageStats struct {
    TotalPages      int
    IndexPages      int
    LeafPages       int
    InternalPages   int
    RootPages       int
    TotalRecords    int
    TotalUsedBytes  int
    TotalGarbage    int
    EmptyPages      int
}

func collectStats(filename string, maxPages uint32) PageStats {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    reader := goinnodb.NewPageReader(file)
    stats := PageStats{}

    for pageNum := uint32(0); pageNum < maxPages; pageNum++ {
        page, err := reader.ReadPage(pageNum)
        if err != nil {
            continue
        }

        stats.TotalPages++

        if page.PageType() == goinnodb.PageTypeIndex {
            stats.IndexPages++
            
            indexPage, err := goinnodb.ParseIndexPage(page)
            if err != nil {
                continue
            }

            if indexPage.IsLeaf() {
                stats.LeafPages++
            } else {
                stats.InternalPages++
            }

            if indexPage.IsRoot() {
                stats.RootPages++
            }

            stats.TotalRecords += int(indexPage.Hdr.NumUserRecs)
            stats.TotalUsedBytes += indexPage.UsedBytes()
            stats.TotalGarbage += int(indexPage.Hdr.GarbageSpace)

            if indexPage.Hdr.NumUserRecs == 0 {
                stats.EmptyPages++
            }
        }
    }

    return stats
}

func main() {
    stats := collectStats("data.ibd", 1000)
    
    fmt.Println("=== Page Statistics ===")
    fmt.Printf("Total Pages Scanned: %d\n", stats.TotalPages)
    fmt.Printf("Index Pages: %d\n", stats.IndexPages)
    fmt.Printf("  - Leaf Pages: %d\n", stats.LeafPages)
    fmt.Printf("  - Internal Pages: %d\n", stats.InternalPages)
    fmt.Printf("  - Root Pages: %d\n", stats.RootPages)
    fmt.Printf("  - Empty Pages: %d\n", stats.EmptyPages)
    fmt.Printf("Total User Records: %d\n", stats.TotalRecords)
    
    if stats.IndexPages > 0 {
        avgFill := float64(stats.TotalUsedBytes) / 
                   float64(stats.IndexPages * goinnodb.PageSize) * 100
        fmt.Printf("Average Page Fill: %.1f%%\n", avgFill)
        
        avgRecords := float64(stats.TotalRecords) / float64(stats.LeafPages)
        fmt.Printf("Average Records per Leaf: %.1f\n", avgRecords)
        
        if stats.TotalGarbage > 0 {
            fmt.Printf("Total Garbage Space: %d bytes (%.1f%%)\n",
                stats.TotalGarbage,
                float64(stats.TotalGarbage)/float64(stats.TotalUsedBytes)*100)
        }
    }
}
```

### Error Handling Example

```go
package main

import (
    "errors"
    "fmt"
    "log"
    "os"
    "strings"
    
    goinnodb "github.com/cslog/go-innodb"
)

func safePageRead(filename string, pageNum uint32) {
    file, err := os.Open(filename)
    if err != nil {
        if os.IsNotExist(err) {
            log.Fatal("File does not exist")
        }
        log.Fatalf("Cannot open file: %v", err)
    }
    defer file.Close()

    reader := goinnodb.NewPageReader(file)
    page, err := reader.ReadPage(pageNum)
    
    if err != nil {
        // Handle specific error types
        errStr := err.Error()
        
        switch {
        case strings.Contains(errStr, "LSN mismatch"):
            fmt.Printf("Page %d is corrupted (LSN mismatch)\n", pageNum)
            fmt.Println("This may indicate:")
            fmt.Println("  - Partial write during crash")
            fmt.Println("  - Disk corruption")
            fmt.Println("  - File system issues")
            
        case strings.Contains(errStr, "short page"):
            fmt.Printf("Page %d is truncated\n", pageNum)
            fmt.Println("File may be corrupted or incomplete")
            
        case strings.Contains(errStr, "EOF"):
            fmt.Printf("Page %d is beyond end of file\n", pageNum)
            info, _ := file.Stat()
            maxPage := info.Size() / goinnodb.PageSize
            fmt.Printf("File contains %d pages (0-%d)\n", maxPage, maxPage-1)
            
        default:
            fmt.Printf("Error reading page %d: %v\n", pageNum, err)
        }
        return
    }

    // Try to parse as index page
    if page.PageType() == goinnodb.PageTypeIndex {
        indexPage, err := goinnodb.ParseIndexPage(page)
        if err != nil {
            if strings.Contains(err.Error(), "only compact"):
                fmt.Println("Page uses redundant format (not supported)")
            } else {
                fmt.Printf("Cannot parse index page: %v\n", err)
            }
            return
        }
        
        fmt.Printf("Successfully read page %d\n", pageNum)
        fmt.Printf("Index ID: %d, Records: %d\n", 
            indexPage.Hdr.IndexID, indexPage.Hdr.NumUserRecs)
    } else {
        fmt.Printf("Page %d is type %d (not an index page)\n", 
            pageNum, page.PageType())
    }
}

func main() {
    safePageRead("data.ibd", 3)
}
```

## Advanced Use Cases

### Corruption Detection

```bash
#!/bin/bash
# Check for LSN mismatches in first 1000 pages

for i in {0..999}; do
    output=$(./go-innodb -file data.ibd -page $i 2>&1)
    if echo "$output" | grep -q "LSN mismatch"; then
        echo "Corruption detected in page $i"
    fi
done
```

### Space Usage Analysis

```bash
#!/bin/bash
# Analyze space usage across pages

echo "Page,Used,Percentage"
for i in {0..100}; do
    ./go-innodb -file data.ibd -page $i -format json 2>/dev/null | \
        jq -r 'if .index_page then 
            "\(.page_number),\(.index_page.used_bytes),\(.index_page.used_bytes * 100 / 16384)" 
        else empty end' 2>/dev/null
done
```

### Index Structure Visualization

```go
// Create a DOT graph of B-tree structure
func generateDotGraph(filename string, rootPage uint32) {
    // ... (implementation that reads pages and outputs Graphviz DOT format)
    // This would traverse the B-tree and create a visual representation
}
```

## Performance Tips

1. **Batch Operations**: When reading multiple pages, reuse the same file handle
2. **Selective Parsing**: Only parse as IndexPage when needed
3. **Error Handling**: Implement proper error handling to skip corrupted pages
4. **Memory Usage**: The library reads full 16KB pages; plan memory accordingly
5. **Concurrent Access**: Use separate PageReader instances for concurrent operations

## Debugging Tips

```bash
# Enable verbose output for debugging
./go-innodb -file data.ibd -page 3 -v -records

# Use jq for JSON filtering
./go-innodb -file data.ibd -page 3 -format json | jq '.'

# Compare pages
diff <(./go-innodb -file data1.ibd -page 3 -format json) \
     <(./go-innodb -file data2.ibd -page 3 -format json)
```
// Package goinnodb provides a Go library for parsing and analyzing InnoDB database pages.
//
// The library is organized into logical groups of functionality:
//
// Core Types and Constants:
//   - types.go: Basic type definitions and constants (PageSize, PageType, RecordType, etc.)
//   - endian.go: Big-endian byte reading utilities
//
// Page Structure Components:
//   - fil.go: FIL header and trailer parsing (page metadata)
//   - inner_page.go: Base page structure (16KB page with FIL header/trailer)
//   - index_page.go: INDEX page parsing with records and directory
//   - index_header.go: Index-specific header within a page
//   - fseg_header.go: File segment header parsing
//
// Record Handling:
//   - record_header.go: Compact record format header (5 bytes)
//   - generic_record.go: Generic record structure with header and position
//   - iter.go: Record iteration and traversal utilities
//
// I/O Operations:
//   - reader.go: Page reader for reading from InnoDB data files
//
// Basic usage:
//
//	file, _ := os.Open("table.ibd")
//	defer file.Close()
//	
//	reader := goinnodb.NewPageReader(file)
//	page, _ := reader.ReadPage(3)
//	
//	if page.PageType() == goinnodb.PageTypeIndex {
//	    indexPage, _ := goinnodb.ParseIndexPage(page)
//	    records, _ := indexPage.WalkRecords(100, true)
//	}
//
package goinnodb
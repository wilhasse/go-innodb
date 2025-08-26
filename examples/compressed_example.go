// compressed_example.go - Example of reading compressed InnoDB pages
package main

import (
	"flag"
	"fmt"
	"os"

	innodb "github.com/wilhasse/go-innodb"
)

func main() {
	var (
		fileName = flag.String("file", "", "InnoDB data file (.ibd)")
		pageNo   = flag.Int("page", 4, "Page number to read")
		keySize  = flag.Int("key-block-size", 0, "KEY_BLOCK_SIZE in KB (1, 2, 4, 8) or 0 for auto")
	)
	flag.Parse()

	if *fileName == "" {
		fmt.Println("Usage: compressed_example -file <table.ibd> [-page N] [-key-block-size K]")
		os.Exit(1)
	}

	// Open the file
	file, err := os.Open(*fileName)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Create compressed page reader
	reader := innodb.NewCompressedPageReader(file)

	// Set physical page size if specified
	if *keySize > 0 {
		physicalSize := *keySize * 1024
		if err := reader.SetPhysicalPageSize(physicalSize); err != nil {
			fmt.Printf("Error setting page size: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Using KEY_BLOCK_SIZE=%dK (physical size: %d bytes)\n", *keySize, physicalSize)
	} else {
		fmt.Println("Auto-detecting compression...")
	}

	// Read the page
	fmt.Printf("\nReading page %d...\n", *pageNo)
	page, err := reader.ReadPage(uint32(*pageNo))
	if err != nil {
		fmt.Printf("Error reading page: %v\n", err)
		os.Exit(1)
	}

	// Display page info
	fmt.Printf("\n=== Page %d ===\n", page.PageNo)
	fmt.Printf("Page Type: %d\n", page.PageType())
	fmt.Printf("Space ID: %d\n", page.FIL.SpaceID)
	fmt.Printf("LSN: %d\n", page.FIL.LastModLSN)
	
	// Show page type name
	pageTypeName := "UNKNOWN"
	switch page.PageType() {
	case innodb.PageTypeIndex:
		pageTypeName = "INDEX"
	case innodb.PageTypeAllocated:
		pageTypeName = "ALLOCATED"
	case innodb.PageTypeUndoLog:
		pageTypeName = "UNDO_LOG"
	case innodb.PageTypeSDI:
		pageTypeName = "SDI"
	}
	fmt.Printf("Page Type Name: %s\n", pageTypeName)

	// If it's an index page, show more details
	if page.PageType() == innodb.PageTypeIndex {
		indexPage, err := innodb.ParseIndexPage(page)
		if err != nil {
			fmt.Printf("Error parsing index page: %v\n", err)
		} else {
			fmt.Printf("\nIndex Page Details:\n")
			formatName := "UNKNOWN"
			if indexPage.Hdr.Format == innodb.FormatCompact {
				formatName = "COMPACT"
			} else if indexPage.Hdr.Format == innodb.FormatRedundant {
				formatName = "REDUNDANT"
			}
			fmt.Printf("  Format: %s\n", formatName)
			fmt.Printf("  Records: %d\n", indexPage.Hdr.NumUserRecs)
			fmt.Printf("  Page Level: %d\n", indexPage.Hdr.PageLevel)
			fmt.Printf("  Index ID: %d\n", indexPage.Hdr.IndexID)

			if indexPage.IsLeaf() {
				fmt.Println("  Type: Leaf page (contains data)")
			} else {
				fmt.Println("  Type: Internal node")
			}
		}
	}

	fmt.Println("\n✓ Page read successfully!")

	// Try to detect if original was compressed
	// This is just for demonstration
	fmt.Println("\nCompression Detection:")
	if *keySize == 0 {
		// Read raw data to check
		rawBuf := make([]byte, 16384)
		off := int64(*pageNo) * 16384
		n, _ := file.ReadAt(rawBuf, off)

		if n < 16384 {
			fmt.Printf("  Physical size on disk: %d bytes (likely compressed)\n", n)
		} else if innodb.IsPageCompressed(rawBuf[:n]) {
			fmt.Println("  Page appears to be compressed")
		} else {
			fmt.Println("  Page appears to be uncompressed (16K)")
		}
	}
}

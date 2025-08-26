// decompress_example.go - Example of using the minimal InnoDB decompression library
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	goinnodb "github.com/wilhasse/go-innodb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <compressed.ibd> [page_number]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s ../testdata/test_compressed.ibd\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ../testdata/test_compressed.ibd 3\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]
	pageNum := 0
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &pageNum)
	}

	// Print library version
	fmt.Printf("InnoDB Decompression Library Version: %s\n\n", goinnodb.GetDecompressVersion())

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}

	fmt.Printf("File: %s\n", filename)
	fmt.Printf("Size: %d bytes\n", fileInfo.Size())

	// For compressed files, we need to detect the actual physical page size
	// Check first few bytes to determine page size
	firstPage := make([]byte, 38) // FIL header
	file.ReadAt(firstPage, 0)
	
	// Try to detect from file structure
	pageSize := 16384 // Default
	if fileInfo.Size() == 114688 {
		pageSize = 16384 // test.ibd has 7 pages of 16KB
	} else if fileInfo.Size() == 57344 {
		pageSize = 8192 // test_compressed.ibd has 7 pages of 8KB compressed
	}
	
	numPages := fileInfo.Size() / int64(pageSize)
	fmt.Printf("Detected page size: %d bytes\n", pageSize)
	fmt.Printf("Number of pages: %d\n\n", numPages)

	// Read and process a specific page
	offset := int64(pageNum) * int64(pageSize)
	if offset >= fileInfo.Size() {
		log.Fatalf("Page %d is beyond file size", pageNum)
	}

	// Read the page
	pageData := make([]byte, pageSize)
	n, err := file.ReadAt(pageData, offset)
	if err != nil && err != io.EOF {
		log.Fatalf("Failed to read page %d: %v", pageNum, err)
	}
	pageData = pageData[:n]

	fmt.Printf("Processing page %d (offset %d, read %d bytes)\n", pageNum, offset, n)
	fmt.Println(strings.Repeat("=", 51))

	// Get page info
	info, err := goinnodb.GetPageInfo(pageData)
	if err != nil {
		fmt.Printf("Warning: Failed to get page info: %v\n", err)
	} else {
		fmt.Printf("Page Info:\n")
		fmt.Printf("  Page Number: %d\n", info.PageNumber)
		fmt.Printf("  Page Type: %d (0x%04X)\n", info.PageType, info.PageType)
		fmt.Printf("  Space ID: %d\n", info.SpaceID)
		fmt.Printf("  Compressed: %v\n", info.IsCompressed)
		fmt.Printf("  Physical Size: %d bytes\n", info.PhysicalSize)
		fmt.Printf("  Logical Size: %d bytes\n", info.LogicalSize)
	}

	// Check if compressed
	isPageCompressed, err := goinnodb.IsPageCompressedV2(pageData)
	if err != nil {
		fmt.Printf("Warning: Failed to check compression: %v\n", err)
	} else {
		fmt.Printf("\nCompression check: %v\n", isPageCompressed)
	}

	// Process the page (handles both compressed and uncompressed)
	fmt.Println("\nProcessing page...")
	processedData, err := goinnodb.ProcessPage(pageData)
	if err != nil {
		log.Fatalf("Failed to process page: %v", err)
	}

	fmt.Printf("Processed page size: %d bytes\n", len(processedData))

	// If it was compressed, show the decompression result
	if isPageCompressed && len(pageData) < len(processedData) {
		fmt.Printf("Decompression successful: %d -> %d bytes (%.1fx expansion)\n",
			len(pageData), len(processedData), 
			float64(len(processedData))/float64(len(pageData)))
	}

	// Try to parse the processed page using existing parser
	innerPage, err := goinnodb.NewInnerPage(uint32(pageNum), processedData)
	if err != nil {
		fmt.Printf("\nWarning: Failed to parse as InnerPage: %v\n", err)
	} else {
		fmt.Printf("\nParsed InnerPage successfully:\n")
		fmt.Printf("  Checksum: 0x%08X\n", innerPage.FIL.Checksum)
		fmt.Printf("  LSN: %d\n", innerPage.FIL.LastModLSN)
		fmt.Printf("  Page Type: %s\n", innerPage.FIL.PageType)
		
		// If it's an index page, show more details
		if fmt.Sprintf("%v", innerPage.FIL.PageType) == "INDEX" {
			indexPage, err := goinnodb.ParseIndexPage(innerPage)
			if err != nil {
				fmt.Printf("  Failed to parse as index page: %v\n", err)
			} else {
				fmt.Printf("  Index Page Details:\n")
				fmt.Printf("    Number of records: %d\n", indexPage.Hdr.NumUserRecs)
				fmt.Printf("    Number of directory slots: %d\n", indexPage.Hdr.NumDirSlots)
				fmt.Printf("    Heap top position: %d\n", indexPage.Hdr.HeapTop)
				fmt.Printf("    Page level: %d\n", indexPage.Hdr.PageLevel)
			}
		}
	}

	// Show first few bytes of processed data (in hex)
	fmt.Printf("\nFirst 128 bytes of processed page (hex):\n")
	showLen := 128
	if len(processedData) < showLen {
		showLen = len(processedData)
	}
	for i := 0; i < showLen; i++ {
		if i%16 == 0 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("%04X: ", i)
		}
		fmt.Printf("%02X ", processedData[i])
	}
	fmt.Println()
}
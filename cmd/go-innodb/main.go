package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	goinnodb "github.com/wilhasse/go-innodb"
	"github.com/wilhasse/go-innodb/record"
	"github.com/wilhasse/go-innodb/schema"
)

func main() {
	var (
		file      = flag.String("file", "", "Path to InnoDB data file (required)")
		pageNum   = flag.Uint("page", 0, "Page number to read (default: 0)")
		format    = flag.String("format", "text", "Output format: text, json, or summary")
		showRecs  = flag.Bool("records", false, "Show all records in the page")
		maxRecs   = flag.Int("max-records", 100, "Maximum records to display")
		verbose   = flag.Bool("v", false, "Verbose output")
		sqlFile   = flag.String("sql", "", "Path to SQL file with CREATE TABLE statement")
		parseData = flag.Bool("parse", false, "Parse column data using table schema")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "InnoDB Page Parser Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -file data.ibd -page 3\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -file data.ibd -page 3 -format json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -file data.ibd -page 3 -records\n", os.Args[0])
	}

	flag.Parse()

	if *file == "" {
		fmt.Fprintf(os.Stderr, "Error: -file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Open the file
	f, err := os.Open(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Load table schema if provided
	var tableDef *schema.TableDef
	if *sqlFile != "" {
		tableDef, err = schema.ParseTableDefFromSQLFile(*sqlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing SQL file: %v\n", err)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("Loaded schema: %s\n", tableDef)
		}
	}

	// Create page reader
	reader := goinnodb.NewPageReader(f)

	// Read the page
	page, err := reader.ReadPage(uint32(*pageNum))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading page %d: %v\n", *pageNum, err)
		os.Exit(1)
	}

	// Output based on format
	switch *format {
	case "json":
		outputJSON(page, *showRecs, *maxRecs, tableDef, *parseData)
	case "summary":
		outputSummary(page)
	default:
		outputText(page, *showRecs, *maxRecs, *verbose, tableDef, *parseData)
	}
}

func outputText(page *goinnodb.InnerPage, showRecs bool, maxRecs int, verbose bool, tableDef *schema.TableDef, parseData bool) {
	fmt.Printf("=== Page %d ===\n", page.PageNo)
	fmt.Printf("\nFIL Header:\n")
	fmt.Printf("  Checksum:    0x%08x\n", page.FIL.Checksum)
	fmt.Printf("  Page Number: %d\n", page.FIL.PageNumber)
	fmt.Printf("  Page Type:   %s (%d)\n", pageTypeName(page.FIL.PageType), page.FIL.PageType)
	fmt.Printf("  Space ID:    %d\n", page.FIL.SpaceID)
	fmt.Printf("  LSN:         %d\n", page.FIL.LastModLSN)

	if page.FIL.Prev != nil {
		fmt.Printf("  Prev Page:   %d\n", *page.FIL.Prev)
	} else {
		fmt.Printf("  Prev Page:   NULL\n")
	}

	if page.FIL.Next != nil {
		fmt.Printf("  Next Page:   %d\n", *page.FIL.Next)
	} else {
		fmt.Printf("  Next Page:   NULL\n")
	}

	fmt.Printf("\nFIL Trailer:\n")
	fmt.Printf("  Checksum:    0x%08x\n", page.Trailer.Checksum)
	fmt.Printf("  Low32 LSN:   0x%08x\n", page.Trailer.Low32LSN)

	// If it's an index page, show more details
	if page.FIL.PageType == goinnodb.PageTypeIndex {
		indexPage, err := goinnodb.ParseIndexPage(page)
		if err != nil {
			fmt.Printf("\nError parsing as index page: %v\n", err)
			return
		}

		fmt.Printf("\nIndex Header:\n")
		fmt.Printf("  Format:      %s\n", formatName(indexPage.Hdr.Format))
		fmt.Printf("  Records:     %d user records\n", indexPage.Hdr.NumUserRecs)
		fmt.Printf("  Heap Recs:   %d\n", indexPage.Hdr.NumHeapRecs)
		fmt.Printf("  Dir Slots:   %d\n", indexPage.Hdr.NumDirSlots)
		fmt.Printf("  Heap Top:    %d\n", indexPage.Hdr.HeapTop)
		fmt.Printf("  Garbage:     %d bytes\n", indexPage.Hdr.GarbageSpace)
		fmt.Printf("  Page Level:  %d %s\n", indexPage.Hdr.PageLevel, leafOrInternal(indexPage))
		fmt.Printf("  Index ID:    %d\n", indexPage.Hdr.IndexID)

		if verbose {
			fmt.Printf("  Max Trx ID:  %d\n", indexPage.Hdr.MaxTrxID)
			fmt.Printf("  Direction:   %s\n", directionName(indexPage.Hdr.Direction))
			fmt.Printf("  N Direction: %d\n", indexPage.Hdr.NumInsertsInDirection)
		}

		fmt.Printf("\nPage Usage:  %d / %d bytes (%.1f%%)\n",
			indexPage.UsedBytes(), goinnodb.PageSize,
			float64(indexPage.UsedBytes())*100/float64(goinnodb.PageSize))

		if showRecs {
			fmt.Printf("\nRecords:\n")

			// Parse records with schema if available
			var records []goinnodb.GenericRecord
			if parseData && tableDef != nil {
				// Use compact parser to parse records with column values
				parser := record.NewCompactParser(tableDef)
				parsedRecords := make([]goinnodb.GenericRecord, 0)

				// Walk and parse each record
				rawRecords, err := goinnodb.WalkRecords(indexPage, maxRecs, true)
				if err != nil {
					fmt.Printf("  Error walking records: %v\n", err)
				} else {
					for _, rawRec := range rawRecords {
						// Re-parse with column data
						parsedRec, err := parser.ParseRecord(indexPage.Inner.Data, rawRec.PrimaryKeyPos, indexPage.IsLeaf())
						if err != nil {
							// Fall back to raw record if parsing fails
							parsedRecords = append(parsedRecords, rawRec)
						} else {
							// Copy metadata from raw record
							parsedRec.PageNumber = rawRec.PageNumber
							parsedRecords = append(parsedRecords, *parsedRec)
						}
					}
					records = parsedRecords
				}
			} else {
				// Use standard walk without parsing
				records, err = goinnodb.WalkRecords(indexPage, maxRecs, true)
				if err != nil {
					fmt.Printf("  Error walking records: %v\n", err)
				}
			}

			if records != nil {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

				// Display parsed column values if available
				if parseData && tableDef != nil && len(records) > 0 && len(records[0].Values) > 0 {
					// Build header with column names
					fmt.Fprintf(w, "  #\t")
					for _, col := range tableDef.Columns {
						fmt.Fprintf(w, "%s\t", col.Name)
					}
					fmt.Fprintln(w)

					// Display values
					for i, rec := range records {
						fmt.Fprintf(w, "  %d\t", i)
						for _, col := range tableDef.Columns {
							val, exists := rec.GetValue(col.Name)
							if !exists || val == nil {
								fmt.Fprintf(w, "NULL\t")
							} else {
								fmt.Fprintf(w, "%v\t", val)
							}
						}
						fmt.Fprintln(w)
					}
				} else if verbose {
					// Original verbose display with hex data
					fmt.Fprintf(w, "  #\tHeap#\tType\tDeleted\tOwned\tNext\tData (hex)\tReadable Strings\n")
					for i, rec := range records {
						dataHex := ""
						readable := ""
						if len(rec.Data) > 0 {
							if len(rec.Data) > 50 {
								dataHex = fmt.Sprintf("%x... (%d bytes)", rec.Data[:50], len(rec.Data))
							} else {
								dataHex = fmt.Sprintf("%x", rec.Data)
							}
							readable = extractReadableStrings(rec.Data)
						}
						fmt.Fprintf(w, "  %d\t%d\t%s\t%v\t%d\t%d\t%s\t%s\n",
							i, rec.Header.HeapNumber,
							recordTypeName(rec.Header.Type),
							rec.Header.FlagsDeleted,
							rec.Header.NumOwned,
							rec.Header.NextRecOffset,
							dataHex,
							readable)
					}
				} else {
					fmt.Fprintf(w, "  #\tHeap#\tType\tDeleted\tOwned\tNext\n")
					for i, rec := range records {
						fmt.Fprintf(w, "  %d\t%d\t%s\t%v\t%d\t%d\n",
							i, rec.Header.HeapNumber,
							recordTypeName(rec.Header.Type),
							rec.Header.FlagsDeleted,
							rec.Header.NumOwned,
							rec.Header.NextRecOffset)
					}
				}
				w.Flush()

				if len(records) == maxRecs {
					fmt.Printf("  ... (showing first %d records)\n", maxRecs)
				}
			}
		}
	}
}

func outputSummary(page *goinnodb.InnerPage) {
	fmt.Printf("Page %d: Type=%s, Space=%d, LSN=%d",
		page.PageNo, pageTypeName(page.FIL.PageType),
		page.FIL.SpaceID, page.FIL.LastModLSN)

	if page.FIL.PageType == goinnodb.PageTypeIndex {
		if indexPage, err := goinnodb.ParseIndexPage(page); err == nil {
			fmt.Printf(", Records=%d, Level=%d, IndexID=%d",
				indexPage.Hdr.NumUserRecs,
				indexPage.Hdr.PageLevel,
				indexPage.Hdr.IndexID)
		}
	}
	fmt.Println()
}

func outputJSON(page *goinnodb.InnerPage, showRecs bool, maxRecs int, tableDef *schema.TableDef, parseData bool) {
	output := map[string]interface{}{
		"page_number": page.PageNo,
		"fil_header": map[string]interface{}{
			"checksum":       page.FIL.Checksum,
			"page_number":    page.FIL.PageNumber,
			"page_type":      page.FIL.PageType,
			"page_type_name": pageTypeName(page.FIL.PageType),
			"space_id":       page.FIL.SpaceID,
			"lsn":            page.FIL.LastModLSN,
			"flush_lsn":      page.FIL.FlushLSN,
			"prev":           page.FIL.Prev,
			"next":           page.FIL.Next,
		},
		"fil_trailer": map[string]interface{}{
			"checksum":  page.Trailer.Checksum,
			"low32_lsn": page.Trailer.Low32LSN,
		},
	}

	if page.FIL.PageType == goinnodb.PageTypeIndex {
		if indexPage, err := goinnodb.ParseIndexPage(page); err == nil {
			indexData := map[string]interface{}{
				"format":        indexPage.Hdr.Format,
				"format_name":   formatName(indexPage.Hdr.Format),
				"user_records":  indexPage.Hdr.NumUserRecs,
				"heap_records":  indexPage.Hdr.NumHeapRecs,
				"dir_slots":     indexPage.Hdr.NumDirSlots,
				"heap_top":      indexPage.Hdr.HeapTop,
				"garbage_space": indexPage.Hdr.GarbageSpace,
				"page_level":    indexPage.Hdr.PageLevel,
				"is_leaf":       indexPage.IsLeaf(),
				"is_root":       indexPage.IsRoot(),
				"index_id":      indexPage.Hdr.IndexID,
				"max_trx_id":    indexPage.Hdr.MaxTrxID,
				"used_bytes":    indexPage.UsedBytes(),
			}

			if showRecs {
				if records, err := goinnodb.WalkRecords(indexPage, maxRecs, true); err == nil {
					recData := make([]map[string]interface{}, len(records))
					for i, rec := range records {
						recData[i] = map[string]interface{}{
							"heap_number": rec.Header.HeapNumber,
							"type":        rec.Header.Type,
							"type_name":   recordTypeName(rec.Header.Type),
							"deleted":     rec.Header.FlagsDeleted,
							"min_rec":     rec.Header.FlagsMinRec,
							"num_owned":   rec.Header.NumOwned,
							"next_offset": rec.Header.NextRecOffset,
							"position":    rec.PrimaryKeyPos,
						}
					}
					indexData["records"] = recData
				}
			}

			output["index_page"] = indexData
		} else {
			output["index_error"] = err.Error()
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func pageTypeName(t goinnodb.PageType) string {
	switch t {
	case goinnodb.PageTypeAllocated:
		return "ALLOCATED"
	case goinnodb.PageTypeIndex:
		return "INDEX"
	case goinnodb.PageTypeUndoLog:
		return "UNDO_LOG"
	case goinnodb.PageTypeSDI:
		return "SDI"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

func formatName(f goinnodb.PageFormat) string {
	switch f {
	case goinnodb.FormatCompact:
		return "COMPACT"
	case goinnodb.FormatRedundant:
		return "REDUNDANT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", f)
	}
}

func recordTypeName(t goinnodb.RecordType) string {
	switch t {
	case goinnodb.RecConventional:
		return "DATA"
	case goinnodb.RecNodePointer:
		return "NODE_PTR"
	case goinnodb.RecInfimum:
		return "INFIMUM"
	case goinnodb.RecSupremum:
		return "SUPREMUM"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

func directionName(d goinnodb.PageDirection) string {
	switch d {
	case goinnodb.DirLeft:
		return "LEFT"
	case goinnodb.DirRight:
		return "RIGHT"
	case goinnodb.DirSamePage:
		return "SAME_PAGE"
	case goinnodb.DirDescending:
		return "DESCENDING"
	case goinnodb.DirNoDirection:
		return "NO_DIRECTION"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", d)
	}
}

func leafOrInternal(p *goinnodb.IndexPage) string {
	if p.IsLeaf() {
		if p.IsRoot() {
			return "(root leaf)"
		}
		return "(leaf)"
	}
	if p.IsRoot() {
		return "(root internal)"
	}
	return "(internal)"
}

// extractReadableStrings extracts ASCII strings from binary data
func extractReadableStrings(data []byte) string {
	var result []string
	var current []byte

	for _, b := range data {
		// Check if byte is printable ASCII (32-126)
		if b >= 32 && b <= 126 {
			current = append(current, b)
		} else {
			// If we have accumulated at least 3 characters, consider it a string
			if len(current) >= 3 {
				result = append(result, string(current))
			}
			current = nil
		}
	}

	// Don't forget the last string if any
	if len(current) >= 3 {
		result = append(result, string(current))
	}

	return strings.Join(result, " | ")
}

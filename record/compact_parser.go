// compact_parser.go - Parser for InnoDB compact record format
package record

// NOTE: Compact format layout: [varlen headers][NULL bitmap][5B header][data]
import (
	"fmt"
	"github.com/wilhasse/go-innodb/column"
	"github.com/wilhasse/go-innodb/format"
	"github.com/wilhasse/go-innodb/schema"
)

// CompactParser parses records in InnoDB compact format
type CompactParser struct {
	tableDef *schema.TableDef
}

// NewCompactParser creates a new compact record parser
func NewCompactParser(tableDef *schema.TableDef) *CompactParser {
	return &CompactParser{
		tableDef: tableDef,
	}
}

// ParseRecord parses a record from raw page data
func (p *CompactParser) ParseRecord(pageData []byte, recordPos int, isLeafPage bool) (*GenericRecord, error) {
	// The actual record content starts at recordPos
	// But we need to read backwards to get variable length headers and NULL bitmap

	// First, read the record header (5 bytes before recordPos)
	headerPos := recordPos - format.RecordHeaderSize
	if headerPos < 0 {
		return nil, fmt.Errorf("invalid record position")
	}

	header, err := ParseRecordHeader(pageData, headerPos)
	if err != nil {
		return nil, fmt.Errorf("parse record header: %w", err)
	}

	// Create the record
	record := &GenericRecord{
		Header:        header,
		PrimaryKeyPos: recordPos,
		Values:        make(map[string]interface{}),
	}

	// Handle special records (INFIMUM/SUPREMUM)
	if header.Type == format.RecInfimum || header.Type == format.RecSupremum {
		record.Data = pageData[recordPos : recordPos+format.SystemRecordBytes]
		return record, nil
	}

	// For user records, we need to parse the variable-length headers and NULL bitmap

	// Step 1: Parse NULL bitmap (only for leaf pages with nullable columns)
	nullBitmap := make([]bool, p.tableDef.NullableColumnCount())
	nullBitmapSize := 0

	if isLeafPage && p.tableDef.HasNullableColumn() {
		nullBitmapSize = p.tableDef.NullBitmapSize()
		nullBitmapPos := headerPos - nullBitmapSize

		if nullBitmapPos < 0 {
			return nil, fmt.Errorf("invalid NULL bitmap position")
		}

		// Read NULL bitmap
		nullBytes := pageData[nullBitmapPos:headerPos]
		nullIdx := 0
		for range p.tableDef.NullableColumns() {
			byteIdx := nullIdx / 8
			bitIdx := nullIdx % 8
			if byteIdx < len(nullBytes) {
				nullBitmap[nullIdx] = (nullBytes[byteIdx] & (1 << bitIdx)) != 0
			}
			nullIdx++
		}
	}

	// Step 2: Parse variable-length field headers
	// Headers are stored right-to-left before the NULL bitmap.
	// Because we iterate from the last varlen column to the first, we must
	// PREPEND each decoded length to keep varLengths in column order.
	varLengths := make([]int, 0, len(p.tableDef.VariableLengthColumns()))
	varLenHeaderSize := 0

	if p.tableDef.HasVariableLengthColumn() {
		// Start at the byte just before the NULL bitmap
		varHeaderPos := headerPos - nullBitmapSize
		// Decide which varlen columns are present in the record.
		// - Leaf records (clustered or secondary) carry headers for ALL
		//   variable-length user columns present in the record.
		// - Internal (node-pointer) records carry only key columns.
		// (We rely on the provided TableDef to reflect the clustered index.)
		var varColumns []*schema.Column
		if isLeafPage {
			varColumns = p.tableDef.VariableLengthColumns()
		} else {
			varColumns = p.tableDef.GetPrimaryKeyVarLenColumns()
		}

		// Variable-length headers are stored in reverse column order.
		// We read backwards through memory, but the rightmost header
		// corresponds to the FIRST variable column, not the last.
		for i := 0; i < len(varColumns); i++ {
			col := varColumns[i]

			// Check if this column is NULL
			isNull := false
			if col.Nullable {
				// Find column's index in nullable columns
				for idx, nullCol := range p.tableDef.NullableColumns() {
					if nullCol.Name == col.Name && nullBitmap[idx] {
						isNull = true
						break
					}
				}
			}

			if isNull {
				varLengths = append([]int{0}, varLengths...) // Prepend 0 for NULL column
				continue
			}

			// Read variable length header (1 or 2 bytes)
			varHeaderPos--
			if varHeaderPos < 0 {
				return nil, fmt.Errorf("invalid variable header position")
			}

			length := int(pageData[varHeaderPos])
			varLenHeaderSize++

			// Check if it's a 2-byte length
			if p.needsTwoByteLength(col, length) {
				varHeaderPos--
				if varHeaderPos < 0 {
					return nil, fmt.Errorf("invalid variable header position")
				}

				// High byte is in the first byte read, with overflow flag in bit 6
				overflowFlag := (length & 0x40) != 0
				length = ((length & 0x3F) << 8) | int(pageData[varHeaderPos])
				varLenHeaderSize++

				if overflowFlag {
					// TODO: Handle overflow pages
					return nil, fmt.Errorf("overflow pages not yet supported")
				}
			}

			// Append to maintain column order
			varLengths = append(varLengths, length)
		}
	}

	// Step 3: Parse actual column data starting from recordPos
	// Note: Transaction fields (6-byte trx_id + 7-byte roll_ptr) are stored
	// AFTER the primary key columns in clustered index leaf pages.
	dataPos := recordPos
	varLenIdx := 0

	// First parse primary key columns
	for _, col := range p.tableDef.PrimaryKeyColumns() {
		// Check if column is NULL
		isNull := false
		if col.Nullable {
			for idx, nullCol := range p.tableDef.NullableColumns() {
				if nullCol.Name == col.Name && nullBitmap[idx] {
					isNull = true
					break
				}
			}
		}

		if isNull {
			record.Values[col.Name] = nil
			if col.IsVariableLength() {
				varLenIdx++
			}
			continue
		}

		// Get variable length if applicable
		varLen := 0
		if col.IsVariableLength() {
			if varLenIdx < len(varLengths) {
				varLen = varLengths[varLenIdx]
				varLenIdx++
			}
		}

		// Parse column value
		value, bytesRead, err := column.ParseColumn(pageData, dataPos, col, varLen)
		if err != nil {
			return nil, fmt.Errorf("parse column %s: %w", col.Name, err)
		}

		record.Values[col.Name] = value
		dataPos += bytesRead
	}

	// Skip transaction ID and roll pointer (13 bytes total) for leaf pages
	if isLeafPage {
		// Skip 6-byte transaction ID and 7-byte roll pointer
		dataPos += 13
	}

	// Now parse non-primary key columns
	for _, col := range p.tableDef.Columns {
		// Skip if already parsed as primary key
		if col.IsPrimaryKey {
			continue
		}

		// Check if column is NULL
		isNull := false
		if col.Nullable {
			for idx, nullCol := range p.tableDef.NullableColumns() {
				if nullCol.Name == col.Name && nullBitmap[idx] {
					isNull = true
					break
				}
			}
		}

		if isNull {
			record.Values[col.Name] = nil
			if col.IsVariableLength() {
				varLenIdx++
			}
			continue
		}

		// Get variable length if applicable
		varLen := 0
		if col.IsVariableLength() {
			if varLenIdx < len(varLengths) {
				varLen = varLengths[varLenIdx]
				varLenIdx++
			}
		}

		// Parse column value
		value, bytesRead, err := column.ParseColumn(pageData, dataPos, col, varLen)
		if err != nil {
			return nil, fmt.Errorf("parse column %s: %w", col.Name, err)
		}

		record.Values[col.Name] = value
		dataPos += bytesRead
	}

	// Store raw data for debugging
	endPos := recordPos + header.NextRecOffset
	if header.NextRecOffset <= 0 || endPos > len(pageData) {
		// Last record or invalid offset, read a reasonable amount
		endPos = dataPos
		if endPos-recordPos > 100 {
			endPos = recordPos + 100
		}
	}
	if endPos > recordPos && endPos <= len(pageData) {
		record.Data = pageData[recordPos:endPos]
	}

	return record, nil
}

// needsTwoByteLength checks if a variable-length column needs 2-byte length header
func (p *CompactParser) needsTwoByteLength(col *schema.Column, firstByte int) bool {
	// If length > 127, might need 2 bytes
	// Also check if column can be long (BLOB/TEXT types or VARCHAR > 255)
	if firstByte > 127 {
		switch col.Type {
		case schema.TypeText, schema.TypeMediumText, schema.TypeLongText,
			schema.TypeBlob, schema.TypeMediumBlob, schema.TypeLongBlob:
			return true
		case schema.TypeVarchar, schema.TypeVarBinary:
			// VARCHAR/VARBINARY needs 2 bytes if max length > 255
			maxLen := col.Length
			if col.Charset == "utf8mb4" {
				maxLen *= 4
			} else if col.Charset == "utf8" {
				maxLen *= 3
			}
			return maxLen > 255
		}
	}
	return false
}

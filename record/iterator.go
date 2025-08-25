// iter.go - Record iteration and traversal utilities
package record

import (
	"fmt"
	"github.com/wilhasse/go-innodb/format"
)

// WalkRecordsFromData walks records from raw page data following the compact record header's relative next offset.
// If skipSystem is true, INFIMUM and SUPREMUM are not returned.
// max limits the number of records to traverse (safety).
// pageNo is the page number for reference.
// pageData is the full 16KB page data.
// infimum is the starting infimum record.
func WalkRecordsFromData(pageNo uint32, pageData []byte, infimum GenericRecord, max int, skipSystem bool) ([]GenericRecord, error) {
	var out []GenericRecord
	cur := infimum
	if !skipSystem {
		out = append(out, cur)
	}
	for steps := 0; steps < max; steps++ {
		nextContent := cur.NextRecordPos()
		if cur.Header.NextRecOffset == 0 {
			break // usually SUPREMUM has next==0
		}
		if nextContent < format.FilHeaderSize+format.PageHeaderSize || nextContent >= format.PageSize-format.FilTrailerSize {
			return out, fmt.Errorf("next content position out of bounds: %d", nextContent)
		}
		nextHeaderPos := nextContent - format.RecordHeaderSize
		if nextHeaderPos < 0 {
			return out, fmt.Errorf("negative next header pos")
		}
		hdr, err := ParseRecordHeader(pageData, nextHeaderPos)
		if err != nil {
			return out, err
		}
		rec := GenericRecord{PageNumber: pageNo, Header: hdr, PrimaryKeyPos: nextContent}

		// Read the actual record data
		// For now, read up to the next record or a reasonable amount of bytes
		dataSize := 0
		if hdr.NextRecOffset > 0 && hdr.NextRecOffset > format.RecordHeaderSize {
			// Size is roughly the distance to the next record minus the header
			dataSize = hdr.NextRecOffset - format.RecordHeaderSize
		} else if hdr.Type == format.RecSupremum {
			// Supremum has fixed 8-byte data
			dataSize = 8
		} else {
			// For the last user record or unknown cases, read a reasonable amount
			// This is a heuristic - proper implementation needs column definitions
			dataSize = 100 // Read up to 100 bytes of data
			maxPos := len(pageData) - nextContent
			if dataSize > maxPos {
				dataSize = maxPos
			}
		}

		if dataSize > 0 && nextContent+dataSize <= len(pageData) {
			rec.Data = pageData[nextContent : nextContent+dataSize]
		}

		if rec.Header.Type == format.RecSupremum {
			if !skipSystem {
				out = append(out, rec)
			}
			break
		}
		out = append(out, rec)
		cur = rec
	}
	return out, nil
}

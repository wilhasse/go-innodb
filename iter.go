// iter.go - Record iteration and traversal utilities
package goinnodb

import "fmt"

// WalkRecords walks records on a page following the compact record header's relative next offset.
// If skipSystem is true, INFIMUM and SUPREMUM are not returned.
// max limits the number of records to traverse (safety).
func (p *IndexPage) WalkRecords(max int, skipSystem bool) ([]GenericRecord, error) {
	if p.Hdr.Format != FormatCompact {
		return nil, fmt.Errorf("only compact format supported in WalkRecords")
	}
	var out []GenericRecord
	cur := p.Infimum
	if !skipSystem {
		out = append(out, cur)
	}
	for steps := 0; steps < max; steps++ {
		nextContent := cur.NextRecordPos()
		if cur.Header.NextRecOffset == 0 {
			break // usually SUPREMUM has next==0
		}
		if nextContent < FilHeaderSize+PageHeaderSize || nextContent >= PageSize-FilTrailerSize {
			return out, fmt.Errorf("next content position out of bounds: %d", nextContent)
		}
		nextHeaderPos := nextContent - RecordHeaderSize
		if nextHeaderPos < 0 {
			return out, fmt.Errorf("negative next header pos")
		}
		hdr, err := ParseRecordHeader(p.Inner.Data, nextHeaderPos)
		if err != nil {
			return out, err
		}
		rec := GenericRecord{PageNumber: p.Inner.PageNo, Header: hdr, PrimaryKeyPos: nextContent}

		// Read the actual record data
		// For now, read up to the next record or a reasonable amount of bytes
		dataSize := 0
		if hdr.NextRecOffset > 0 && hdr.NextRecOffset > RecordHeaderSize {
			// Size is roughly the distance to the next record minus the header
			dataSize = hdr.NextRecOffset - RecordHeaderSize
		} else if hdr.Type == RecSupremum {
			// Supremum has fixed 8-byte data
			dataSize = 8
		} else {
			// For the last user record or unknown cases, read a reasonable amount
			// This is a heuristic - proper implementation needs column definitions
			dataSize = 100 // Read up to 100 bytes of data
			maxPos := len(p.Inner.Data) - nextContent
			if dataSize > maxPos {
				dataSize = maxPos
			}
		}

		if dataSize > 0 && nextContent+dataSize <= len(p.Inner.Data) {
			rec.Data = p.Inner.Data[nextContent : nextContent+dataSize]
		}

		if rec.Header.Type == RecSupremum {
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

// generic_record.go - Generic record structure with header and position
package record

import (
	"fmt"
	"strings"
)

// GenericRecord holds header and the position of the content (immediately after header).
type GenericRecord struct {
	PageNumber      uint32
	Header          RecordHeader
	PrimaryKeyPos   int                    // absolute offset where this record's content starts
	ChildPageNumber uint32                 // for internal pages (not decoded here)
	Data            []byte                 // raw record data (excluding header)
	Values          map[string]interface{} // parsed column values (column name -> value)
}

func (r GenericRecord) NextRecordPos() int {
	return r.PrimaryKeyPos + r.Header.NextRecOffset
}

// GetValue returns the parsed value for a column
func (r GenericRecord) GetValue(columnName string) (interface{}, bool) {
	if r.Values == nil {
		return nil, false
	}
	val, exists := r.Values[columnName]
	return val, exists
}

// SetValue sets the parsed value for a column
func (r GenericRecord) SetValue(columnName string, value interface{}) {
	if r.Values == nil {
		r.Values = make(map[string]interface{})
	}
	r.Values[columnName] = value
}

// String returns a string representation of the record
func (r GenericRecord) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Record(page=%d, pos=%d, type=%s", 
		r.PageNumber, r.PrimaryKeyPos, r.Header.Type))
	
	if len(r.Values) > 0 {
		sb.WriteString(", values={")
		first := true
		for k, v := range r.Values {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
		sb.WriteString("}")
	}
	
	sb.WriteString(")")
	return sb.String()
}

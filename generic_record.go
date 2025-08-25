package goinnodb

// GenericRecord holds header and the position of the content (immediately after header).
type GenericRecord struct {
	PageNumber      uint32
	Header          RecordHeader
	PrimaryKeyPos   int    // absolute offset where this record's content starts
	ChildPageNumber uint32 // for internal pages (not decoded here)
}

func (r GenericRecord) NextRecordPos() int {
	return r.PrimaryKeyPos + r.Header.NextRecOffset
}

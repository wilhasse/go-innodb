package goinnodb

// Sizes and constants
const (
	PageSize          = 16 * 1024 // 16384
	FilHeaderSize     = 38
	FilTrailerSize    = 8
	RecordHeaderSize  = 5 // compact header (3B bits + 2B next)
	SystemRecordBytes = 8 // "infimum\x00" or "supremum" literal
	PageDirSlotSize   = 2

	// Index (page) header = 36 bytes
	// FSEG header (immediately after) = 20 bytes
	PageHeaderSize = 56
	PageDataOff    = FilHeaderSize + PageHeaderSize
)

// Page types (subset)
type PageType uint16

const (
	PageTypeAllocated PageType = 0
	PageTypeIndex     PageType = 17855
	PageTypeUndoLog   PageType = 2
	PageTypeSDI       PageType = 17853
)

type PageFormat uint8

const (
	FormatRedundant PageFormat = 0
	FormatCompact   PageFormat = 1
)

type PageDirection uint16

const (
	DirLeft        PageDirection = 1
	DirRight       PageDirection = 2
	DirSameRec     PageDirection = 3
	DirSamePage    PageDirection = 4
	DirNoDirection PageDirection = 5
)

type RecordType uint8

const (
	RecConventional RecordType = 0
	RecNodePointer  RecordType = 1
	RecInfimum      RecordType = 2
	RecSupremum     RecordType = 3
)

var (
	LitInfimum  = []byte("infimum\x00")
	LitSupremum = []byte("supremum")
)

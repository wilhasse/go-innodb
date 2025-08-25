// exports.go - Re-exports for main package API
package goinnodb

import (
	"github.com/wilhasse/go-innodb/format"
	"github.com/wilhasse/go-innodb/page"
	"github.com/wilhasse/go-innodb/record"
)

// Re-export types from format package
type (
	PageType      = format.PageType
	PageFormat    = format.PageFormat
	PageDirection = format.PageDirection
	RecordType    = format.RecordType
)

// Re-export constants from format package
const (
	PageSize          = format.PageSize
	PageTypeIndex     = format.PageTypeIndex
	PageTypeUndoLog   = format.PageTypeUndoLog
	PageTypeAllocated = format.PageTypeAllocated
	PageTypeSDI       = format.PageTypeSDI
	FormatCompact     = format.FormatCompact
	FormatRedundant   = format.FormatRedundant
	RecConventional   = format.RecConventional
	RecNodePointer    = format.RecNodePointer
	RecInfimum        = format.RecInfimum
	RecSupremum       = format.RecSupremum
	DirLeft           = format.DirLeft
	DirRight          = format.DirRight
	DirSamePage       = format.DirSamePage
	DirDescending     = format.DirDescending
	DirNoDirection    = format.DirNoDirection
)

// Re-export types from page package
type (
	InnerPage  = page.InnerPage
	IndexPage  = page.IndexPage
	FilHeader  = page.FilHeader
	FilTrailer = page.FilTrailer
	FsegHeader = page.FsegHeader
)

// Re-export functions from page package
var (
	NewInnerPage    = page.NewInnerPage
	ParseIndexPage  = page.ParseIndexPage
	ParseFilHeader  = page.ParseFilHeader
	ParseFilTrailer = page.ParseFilTrailer
	ParseFsegHeader = page.ParseFsegHeader
)

// Re-export types from record package
type (
	RecordHeader  = record.RecordHeader
	IndexHeader   = record.IndexHeader
	GenericRecord = record.GenericRecord
)

// Re-export functions from record package
var (
	ParseRecordHeader = record.ParseRecordHeader
	ParseIndexHeader  = record.ParseIndexHeader
)

// WalkRecords is a convenience function to walk records on an IndexPage
func WalkRecords(p *IndexPage, max int, skipSystem bool) ([]record.GenericRecord, error) {
	return p.WalkRecords(max, skipSystem)
}

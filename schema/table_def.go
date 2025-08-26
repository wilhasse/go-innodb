// table_def.go - Table definition for InnoDB schema
package schema

import (
	"fmt"
	"strings"
)

// TableDef represents a table definition with columns and metadata
type TableDef struct {
	Name        string             // Table name
	Columns     []*Column          // All columns in order
	ColumnMap   map[string]*Column // Column name to column mapping
	PrimaryKeys []string           // Primary key column names in order
	Charset     string             // Default character set
	Collation   string             // Default collation
	Engine      string             // Storage engine (should be InnoDB)

	// Cached metadata for efficient parsing
	nullableColumns   []*Column
	varLenColumns     []*Column
	primaryKeyColumns []*Column
	nullableCount     int
	hasNullableColumn bool
	hasVarLenColumn   bool
}

// NewTableDef creates a new table definition
func NewTableDef(name string) *TableDef {
	return &TableDef{
		Name:      name,
		Columns:   make([]*Column, 0),
		ColumnMap: make(map[string]*Column),
	}
}

// AddColumn adds a column to the table definition
func (td *TableDef) AddColumn(col *Column) error {
	if _, exists := td.ColumnMap[col.Name]; exists {
		return fmt.Errorf("column %s already exists", col.Name)
	}

	col.Ordinal = len(td.Columns)
	td.Columns = append(td.Columns, col)
	td.ColumnMap[col.Name] = col

	// Update cached metadata
	if col.Nullable {
		td.nullableColumns = append(td.nullableColumns, col)
		td.nullableCount++
		td.hasNullableColumn = true
	}

	if col.IsVariableLength() {
		td.varLenColumns = append(td.varLenColumns, col)
		td.hasVarLenColumn = true
	}

	return nil
}

// SetPrimaryKeys sets the primary key columns
func (td *TableDef) SetPrimaryKeys(keys []string) error {
	td.PrimaryKeys = keys
	td.primaryKeyColumns = make([]*Column, 0, len(keys))

	for _, key := range keys {
		col, exists := td.ColumnMap[key]
		if !exists {
			return fmt.Errorf("primary key column %s not found", key)
		}
		col.IsPrimaryKey = true
		td.primaryKeyColumns = append(td.primaryKeyColumns, col)
	}

	return nil
}

// GetColumn returns a column by name
func (td *TableDef) GetColumn(name string) (*Column, bool) {
	col, exists := td.ColumnMap[name]
	return col, exists
}

// GetColumnByOrdinal returns a column by ordinal position
func (td *TableDef) GetColumnByOrdinal(ordinal int) (*Column, error) {
	if ordinal < 0 || ordinal >= len(td.Columns) {
		return nil, fmt.Errorf("ordinal %d out of range", ordinal)
	}
	return td.Columns[ordinal], nil
}

// NullableColumns returns all nullable columns
func (td *TableDef) NullableColumns() []*Column {
	return td.nullableColumns
}

// VariableLengthColumns returns all variable length columns
func (td *TableDef) VariableLengthColumns() []*Column {
	return td.varLenColumns
}

// PrimaryKeyColumns returns primary key columns in order
func (td *TableDef) PrimaryKeyColumns() []*Column {
	return td.primaryKeyColumns
}

// HasNullableColumn returns true if table has nullable columns
func (td *TableDef) HasNullableColumn() bool {
	return td.hasNullableColumn
}

// HasVariableLengthColumn returns true if table has variable length columns
func (td *TableDef) HasVariableLengthColumn() bool {
	return td.hasVarLenColumn
}

// NullableColumnCount returns the number of nullable columns
func (td *TableDef) NullableColumnCount() int {
	return td.nullableCount
}

// NullBitmapSize returns the size of NULL bitmap in bytes
func (td *TableDef) NullBitmapSize() int {
	return (td.nullableCount + 7) / 8
}

// ColumnCount returns the total number of columns
func (td *TableDef) ColumnCount() int {
	return len(td.Columns)
}

// HasPrimaryKey returns true if table has primary key
func (td *TableDef) HasPrimaryKey() bool {
	return len(td.PrimaryKeys) > 0
}

// GetPrimaryKeyVarLenColumns returns variable length columns in primary key
func (td *TableDef) GetPrimaryKeyVarLenColumns() []*Column {
	var result []*Column
	for _, col := range td.primaryKeyColumns {
		if col.IsVariableLength() {
			result = append(result, col)
		}
	}
	return result
}

// IsColumnPrimaryKey checks if a column is part of primary key
func (td *TableDef) IsColumnPrimaryKey(col *Column) bool {
	return col.IsPrimaryKey
}

// String returns a string representation of the table definition
func (td *TableDef) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Table: %s\n", td.Name))
	sb.WriteString("Columns:\n")
	for _, col := range td.Columns {
		nullable := ""
		if col.Nullable {
			nullable = " NULL"
		} else {
			nullable = " NOT NULL"
		}
		pk := ""
		if col.IsPrimaryKey {
			pk = " PRIMARY KEY"
		}
		sb.WriteString(fmt.Sprintf("  %d. %s %s(%d)%s%s\n",
			col.Ordinal, col.Name, col.Type, col.Length, nullable, pk))
	}
	if len(td.PrimaryKeys) > 0 {
		sb.WriteString(fmt.Sprintf("Primary Keys: %v\n", td.PrimaryKeys))
	}
	return sb.String()
}

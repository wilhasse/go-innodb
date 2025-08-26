// column.go - Column definition for InnoDB table schema
package schema

import "errors"

// Common errors
var (
	ErrUnsupportedType = errors.New("unsupported column type")
)

// ColumnType represents the MySQL column data type
type ColumnType string

const (
	// Integer types
	TypeTinyInt   ColumnType = "TINYINT"
	TypeSmallInt  ColumnType = "SMALLINT"
	TypeMediumInt ColumnType = "MEDIUMINT"
	TypeInt       ColumnType = "INT"
	TypeBigInt    ColumnType = "BIGINT"

	// String types
	TypeChar       ColumnType = "CHAR"
	TypeVarchar    ColumnType = "VARCHAR"
	TypeText       ColumnType = "TEXT"
	TypeTinyText   ColumnType = "TINYTEXT"
	TypeMediumText ColumnType = "MEDIUMTEXT"
	TypeLongText   ColumnType = "LONGTEXT"

	// Binary types
	TypeBinary     ColumnType = "BINARY"
	TypeVarBinary  ColumnType = "VARBINARY"
	TypeBlob       ColumnType = "BLOB"
	TypeTinyBlob   ColumnType = "TINYBLOB"
	TypeMediumBlob ColumnType = "MEDIUMBLOB"
	TypeLongBlob   ColumnType = "LONGBLOB"

	// Date and time types
	TypeDate      ColumnType = "DATE"
	TypeTime      ColumnType = "TIME"
	TypeDateTime  ColumnType = "DATETIME"
	TypeTimestamp ColumnType = "TIMESTAMP"
	TypeYear      ColumnType = "YEAR"

	// Decimal types
	TypeDecimal ColumnType = "DECIMAL"
	TypeNumeric ColumnType = "NUMERIC"
	TypeFloat   ColumnType = "FLOAT"
	TypeDouble  ColumnType = "DOUBLE"

	// Other types
	TypeBit     ColumnType = "BIT"
	TypeEnum    ColumnType = "ENUM"
	TypeSet     ColumnType = "SET"
	TypeBoolean ColumnType = "BOOLEAN"
	TypeBool    ColumnType = "BOOL"
	TypeJSON    ColumnType = "JSON"

	// Internal type for tables without primary key
	TypeRowID ColumnType = "ROW_ID"
)

// Column represents a column definition in a MySQL table
type Column struct {
	Name          string     // Column name
	Type          ColumnType // Column data type
	Ordinal       int        // Position in table (0-based)
	Length        int        // Length for CHAR, VARCHAR, etc.
	Precision     int        // Precision for DECIMAL, TIME, DATETIME, TIMESTAMP
	Scale         int        // Scale for DECIMAL
	Nullable      bool       // Whether column can be NULL
	AutoIncrement bool       // AUTO_INCREMENT flag
	Unsigned      bool       // UNSIGNED flag for numeric types
	Charset       string     // Character set for string columns
	Collation     string     // Collation for string columns
	DefaultValue  string     // Default value
	IsPrimaryKey  bool       // Part of primary key
	EnumValues    []string   // Values for ENUM type
	SetValues     []string   // Values for SET type
}

// IsVariableLength returns true if the column has variable length storage
func (c *Column) IsVariableLength() bool {
	switch c.Type {
	case TypeVarchar, TypeVarBinary,
		TypeText, TypeTinyText, TypeMediumText, TypeLongText,
		TypeBlob, TypeTinyBlob, TypeMediumBlob, TypeLongBlob,
		TypeJSON:
		return true
	case TypeChar, TypeBinary:
		// In InnoDB, CHAR can be variable if using multi-byte charset
		// and actual data is longer than the defined length
		return c.Charset != "" && c.Charset != "latin1" && c.Charset != "ascii"
	default:
		return false
	}
}

// IsFixedLength returns true if the column has fixed length storage
func (c *Column) IsFixedLength() bool {
	switch c.Type {
	case TypeChar, TypeBinary:
		// Only fixed if single-byte charset or small enough
		return c.Charset == "" || c.Charset == "latin1" || c.Charset == "ascii"
	default:
		return false
	}
}

// StorageSize returns the storage size in bytes for fixed-length columns
func (c *Column) StorageSize() int {
	switch c.Type {
	case TypeTinyInt:
		return 1
	case TypeSmallInt, TypeYear:
		return 2
	case TypeMediumInt, TypeDate:
		return 3
	case TypeInt, TypeFloat:
		return 4
	case TypeBigInt, TypeDouble, TypeDateTime, TypeTimestamp:
		return 8
	case TypeTime:
		return 3 + (c.Precision+1)/2
	case TypeDecimal, TypeNumeric:
		// Complex calculation based on precision and scale
		return calculateDecimalSize(c.Precision, c.Scale)
	case TypeBit:
		return (c.Length + 7) / 8
	case TypeChar, TypeBinary:
		if c.IsFixedLength() {
			return c.Length
		}
	case TypeRowID:
		return 6 // Internal 6-byte row ID
	}
	return 0 // Variable length or unknown
}

// calculateDecimalSize calculates storage size for DECIMAL type
func calculateDecimalSize(precision, scale int) int {
	// MySQL stores decimal as binary with 4 bytes per 9 digits
	// This is a simplified calculation
	integerDigits := precision - scale
	integerBytes := (integerDigits / 9) * 4
	if integerDigits%9 > 0 {
		integerBytes += (integerDigits%9 + 1) / 2
	}

	fractionBytes := (scale / 9) * 4
	if scale%9 > 0 {
		fractionBytes += (scale%9 + 1) / 2
	}

	return integerBytes + fractionBytes
}

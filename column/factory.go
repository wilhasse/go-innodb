// factory.go - Factory for getting appropriate column parser
package column

import (
	"github.com/wilhasse/go-innodb/schema"
)

var (
	intParser      = &IntParser{}
	stringParser   = &StringParser{}
	dateTimeParser = &DateTimeParser{}
	// Add more parsers as needed
)

// GetParser returns the appropriate parser for the column type
func GetParser(col *schema.Column) Parser {
	switch col.Type {
	// Integer types
	case schema.TypeTinyInt, schema.TypeSmallInt, schema.TypeMediumInt,
		schema.TypeInt, schema.TypeBigInt, schema.TypeYear,
		schema.TypeBoolean, schema.TypeBool:
		return intParser

	// String types
	case schema.TypeChar, schema.TypeVarchar,
		schema.TypeText, schema.TypeTinyText, schema.TypeMediumText, schema.TypeLongText,
		schema.TypeBinary, schema.TypeVarBinary,
		schema.TypeBlob, schema.TypeTinyBlob, schema.TypeMediumBlob, schema.TypeLongBlob:
		return stringParser

	// Date/time types
	case schema.TypeDate, schema.TypeTime, schema.TypeDateTime,
		schema.TypeTimestamp:
		return dateTimeParser

	// TODO: Add more parsers for:
	// - DECIMAL/NUMERIC
	// - FLOAT/DOUBLE
	// - ENUM/SET
	// - BIT
	// - JSON

	default:
		return nil
	}
}

// ParseColumn parses a column value using the appropriate parser
func ParseColumn(input []byte, offset int, col *schema.Column, varLen int) (interface{}, int, error) {
	parser := GetParser(col)
	if parser == nil {
		return nil, 0, schema.ErrUnsupportedType
	}
	return parser.Parse(input, offset, col, varLen)
}

// SkipColumn skips a column value without parsing
func SkipColumn(input []byte, offset int, col *schema.Column, varLen int) (int, error) {
	parser := GetParser(col)
	if parser == nil {
		return 0, schema.ErrUnsupportedType
	}
	return parser.Skip(input, offset, col, varLen)
}

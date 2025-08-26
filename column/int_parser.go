// int_parser.go - Parser for integer column types
package column

import (
	"github.com/wilhasse/go-innodb/schema"
)

// IntParser handles all integer type columns
type IntParser struct {
	BaseParser
}

// Parse parses integer value based on column type
func (p *IntParser) Parse(input []byte, offset int, col *schema.Column, varLen int) (interface{}, int, error) {
	switch col.Type {
	case schema.TypeTinyInt:
		if col.Unsigned {
			val, err := p.readUint8(input, offset)
			return val, 1, err
		}
		val, err := p.readInt8(input, offset)
		return val, 1, err

	case schema.TypeSmallInt, schema.TypeYear:
		if col.Unsigned {
			val, err := p.readUint16(input, offset)
			return val, 2, err
		}
		if col.Type == schema.TypeYear {
			// YEAR is stored as unsigned byte, 0 = year 0000, otherwise add 1900
			val, err := p.readUint8(input, offset)
			if err != nil {
				return nil, 0, err
			}
			if val == 0 {
				return uint16(0), 1, nil
			}
			return uint16(uint16(val) + 1900), 1, nil
		}
		val, err := p.readInt16(input, offset)
		return val, 2, err

	case schema.TypeMediumInt:
		if col.Unsigned {
			val, err := p.readUnsignedMediumInt(input, offset)
			return val, 3, err
		}
		val, err := p.readMediumInt(input, offset)
		return val, 3, err

	case schema.TypeInt:
		if col.Unsigned {
			val, err := p.readUint32(input, offset)
			return val, 4, err
		}
		val, err := p.readInt32(input, offset)
		return val, 4, err

	case schema.TypeBigInt:
		if col.Unsigned {
			val, err := p.readUint64(input, offset)
			return val, 8, err
		}
		val, err := p.readInt64(input, offset)
		return val, 8, err

	default:
		// For BOOLEAN (stored as TINYINT(1))
		if col.Type == schema.TypeBoolean || col.Type == schema.TypeBool {
			val, err := p.readUint8(input, offset)
			if err != nil {
				return nil, 0, err
			}
			return val != 0, 1, nil
		}
		
		return nil, 0, schema.ErrUnsupportedType
	}
}

// Skip skips integer value without parsing
func (p *IntParser) Skip(input []byte, offset int, col *schema.Column, varLen int) (int, error) {
	switch col.Type {
	case schema.TypeTinyInt, schema.TypeBoolean, schema.TypeBool:
		return 1, nil
	case schema.TypeSmallInt:
		return 2, nil
	case schema.TypeYear:
		return 1, nil
	case schema.TypeMediumInt:
		return 3, nil
	case schema.TypeInt:
		return 4, nil
	case schema.TypeBigInt:
		return 8, nil
	default:
		return 0, schema.ErrUnsupportedType
	}
}
// string_parser.go - Parser for string/text column types
package column

import (
	"strings"
	"github.com/wilhasse/go-innodb/schema"
)

// StringParser handles VARCHAR, CHAR, TEXT and other string types
type StringParser struct {
	BaseParser
}

// Parse parses string value based on column type
func (p *StringParser) Parse(input []byte, offset int, col *schema.Column, varLen int) (interface{}, int, error) {
	var bytesRead int
	
	switch col.Type {
	case schema.TypeChar:
		// CHAR can be variable length in multi-byte charsets
		if col.IsVariableLength() && varLen > 0 {
			// Variable length CHAR
			data, err := p.readBytes(input, offset, varLen)
			if err != nil {
				return nil, 0, err
			}
			bytesRead = varLen
			// Trim trailing spaces for CHAR
			str := string(data)
			str = strings.TrimRight(str, " ")
			return str, bytesRead, nil
		} else {
			// Fixed length CHAR
			length := col.Length
			if col.Charset == "utf8mb4" {
				length *= 4 // Maximum bytes per character
			} else if col.Charset == "utf8" {
				length *= 3
			}
			data, err := p.readBytes(input, offset, length)
			if err != nil {
				return nil, 0, err
			}
			bytesRead = length
			// Trim trailing spaces
			str := string(data)
			str = strings.TrimRight(str, " ")
			return str, bytesRead, nil
		}
		
	case schema.TypeVarchar, schema.TypeText, schema.TypeTinyText, 
		 schema.TypeMediumText, schema.TypeLongText:
		// Variable length string types use varLen parameter
		if varLen <= 0 {
			return "", 0, nil
		}
		data, err := p.readBytes(input, offset, varLen)
		if err != nil {
			return nil, 0, err
		}
		return string(data), varLen, nil
		
	case schema.TypeBinary:
		// Fixed length binary
		length := col.Length
		data, err := p.readBytes(input, offset, length)
		if err != nil {
			return nil, 0, err
		}
		return data, length, nil
		
	case schema.TypeVarBinary, schema.TypeBlob, schema.TypeTinyBlob, 
		 schema.TypeMediumBlob, schema.TypeLongBlob:
		// Variable length binary types
		if varLen <= 0 {
			return []byte{}, 0, nil
		}
		data, err := p.readBytes(input, offset, varLen)
		if err != nil {
			return nil, 0, err
		}
		return data, varLen, nil
		
	default:
		return nil, 0, schema.ErrUnsupportedType
	}
}

// Skip skips string value without parsing
func (p *StringParser) Skip(input []byte, offset int, col *schema.Column, varLen int) (int, error) {
	switch col.Type {
	case schema.TypeChar:
		if col.IsVariableLength() && varLen > 0 {
			return varLen, nil
		}
		length := col.Length
		if col.Charset == "utf8mb4" {
			length *= 4
		} else if col.Charset == "utf8" {
			length *= 3
		}
		return length, nil
		
	case schema.TypeVarchar, schema.TypeText, schema.TypeTinyText, 
		 schema.TypeMediumText, schema.TypeLongText,
		 schema.TypeVarBinary, schema.TypeBlob, schema.TypeTinyBlob, 
		 schema.TypeMediumBlob, schema.TypeLongBlob:
		return varLen, nil
		
	case schema.TypeBinary:
		return col.Length, nil
		
	default:
		return 0, schema.ErrUnsupportedType
	}
}
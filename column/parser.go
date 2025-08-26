// parser.go - Column parser interface and base implementation
package column

import (
	"github.com/wilhasse/go-innodb/format"
	"github.com/wilhasse/go-innodb/schema"
)

// Parser interface for parsing column values from raw bytes
type Parser interface {
	// Parse reads and parses column value from input
	Parse(input []byte, offset int, col *schema.Column, varLen int) (value interface{}, bytesRead int, err error)
	
	// Skip skips column value in input without parsing
	Skip(input []byte, offset int, col *schema.Column, varLen int) (bytesRead int, err error)
}

// BaseParser provides common functionality for column parsers
type BaseParser struct{}

// readBytes reads specified number of bytes from input
func (p *BaseParser) readBytes(input []byte, offset, length int) ([]byte, error) {
	if offset+length > len(input) {
		return nil, format.ErrShortRead
	}
	return input[offset : offset+length], nil
}

// readUint8 reads an unsigned 8-bit integer
func (p *BaseParser) readUint8(input []byte, offset int) (uint8, error) {
	if offset+1 > len(input) {
		return 0, format.ErrShortRead
	}
	return input[offset], nil
}

// readUint16 reads an unsigned 16-bit integer (big-endian)
func (p *BaseParser) readUint16(input []byte, offset int) (uint16, error) {
	val, err := format.Be16(input, offset)
	return uint16(val), err
}

// readUint32 reads an unsigned 32-bit integer (big-endian)
func (p *BaseParser) readUint32(input []byte, offset int) (uint32, error) {
	return format.Be32(input, offset)
}

// readUint64 reads an unsigned 64-bit integer (big-endian)
func (p *BaseParser) readUint64(input []byte, offset int) (uint64, error) {
	return format.Be64(input, offset)
}

// readInt8 reads a signed 8-bit integer with XOR transformation
func (p *BaseParser) readInt8(input []byte, offset int) (int8, error) {
	val, err := p.readUint8(input, offset)
	if err != nil {
		return 0, err
	}
	// XOR with sign bit to convert from InnoDB format
	return int8(val ^ 0x80), nil
}

// readInt16 reads a signed 16-bit integer with XOR transformation
func (p *BaseParser) readInt16(input []byte, offset int) (int16, error) {
	val, err := p.readUint16(input, offset)
	if err != nil {
		return 0, err
	}
	// XOR with sign bit to convert from InnoDB format
	return int16(val ^ 0x8000), nil
}

// readInt32 reads a signed 32-bit integer with XOR transformation
func (p *BaseParser) readInt32(input []byte, offset int) (int32, error) {
	val, err := p.readUint32(input, offset)
	if err != nil {
		return 0, err
	}
	// XOR with sign bit to convert from InnoDB format
	return int32(val ^ 0x80000000), nil
}

// readInt64 reads a signed 64-bit integer with XOR transformation
func (p *BaseParser) readInt64(input []byte, offset int) (int64, error) {
	val, err := p.readUint64(input, offset)
	if err != nil {
		return 0, err
	}
	// XOR with sign bit to convert from InnoDB format
	return int64(val ^ 0x8000000000000000), nil
}

// readMediumInt reads a 3-byte signed integer with XOR transformation
func (p *BaseParser) readMediumInt(input []byte, offset int) (int32, error) {
	if offset+3 > len(input) {
		return 0, format.ErrShortRead
	}
	
	// Read 3 bytes as unsigned
	val := uint32(input[offset]) | 
		(uint32(input[offset+1]) << 8) | 
		(uint32(input[offset+2]) << 16)
	
	// XOR with sign bit
	val ^= 0x800000
	
	// Sign extend if negative
	if val&0x800000 != 0 {
		val |= 0xFF000000
	}
	
	return int32(val), nil
}

// readUnsignedMediumInt reads a 3-byte unsigned integer
func (p *BaseParser) readUnsignedMediumInt(input []byte, offset int) (uint32, error) {
	if offset+3 > len(input) {
		return 0, format.ErrShortRead
	}
	
	return uint32(input[offset]) | 
		(uint32(input[offset+1]) << 8) | 
		(uint32(input[offset+2]) << 16), nil
}
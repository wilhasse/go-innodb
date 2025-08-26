// datetime_parser.go - Parser for date and time column types
package column

import (
	"fmt"
	"time"
	"github.com/wilhasse/go-innodb/schema"
)

// DateTimeParser handles DATE, TIME, DATETIME, TIMESTAMP types
type DateTimeParser struct {
	BaseParser
}

// Parse parses date/time value based on column type
func (p *DateTimeParser) Parse(input []byte, offset int, col *schema.Column, varLen int) (interface{}, int, error) {
	switch col.Type {
	case schema.TypeDate:
		// DATE is stored as 3-byte integer
		// Bits: 15 for year, 4 for month, 5 for day
		val, err := p.readUnsignedMediumInt(input, offset)
		if err != nil {
			return nil, 0, err
		}
		
		// XOR transformation for signed storage
		val ^= 0x800000
		
		day := val & 0x1F
		val >>= 5
		month := val & 0x0F
		val >>= 4
		year := val
		
		if year == 0 && month == 0 && day == 0 {
			return "0000-00-00", 3, nil
		}
		
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day), 3, nil
		
	case schema.TypeTimestamp:
		// TIMESTAMP is 4 bytes (Unix timestamp)
		val, err := p.readUint32(input, offset)
		if err != nil {
			return nil, 0, err
		}
		
		if val == 0 {
			return "0000-00-00 00:00:00", 4, nil
		}
		
		t := time.Unix(int64(val), 0).UTC()
		return t.Format("2006-01-02 15:04:05"), 4, nil
		
	case schema.TypeDateTime:
		// DATETIME is 8 bytes packed
		// For MySQL 5.6.4+, it's stored as:
		// 1 bit sign (always 1 for positive)
		// 17 bits year*13+month
		// 5 bits day
		// 5 bits hour  
		// 6 bits minute
		// 6 bits second
		// Total: 40 bits = 5 bytes + fractional seconds
		
		// Read 5 bytes
		if offset+5 > len(input) {
			return nil, 0, fmt.Errorf("short read for DATETIME")
		}
		
		// Unpack big-endian 5 bytes
		packedValue := uint64(0)
		for i := 0; i < 5; i++ {
			packedValue = (packedValue << 8) | uint64(input[offset+4-i])
		}
		
		second := int(packedValue & 0x3F)
		packedValue >>= 6
		minute := int(packedValue & 0x3F)
		packedValue >>= 6
		hour := int(packedValue & 0x1F)
		packedValue >>= 5
		day := int(packedValue & 0x1F)
		packedValue >>= 5
		yearMonth := int(packedValue & 0x1FFFF)
		
		month := yearMonth % 13
		year := yearMonth / 13
		
		bytesRead := 5
		fractionStr := ""
		
		// Handle fractional seconds if precision > 0
		if col.Precision > 0 {
			fracBytes := (col.Precision + 1) / 2
			fraction, err := p.readFraction(input, offset+5, col.Precision)
			if err != nil {
				return nil, 0, err
			}
			if fraction > 0 {
				fractionStr = fmt.Sprintf(".%06d", fraction)[:col.Precision+1]
			}
			bytesRead += fracBytes
		}
		
		return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d%s", 
			year, month, day, hour, minute, second, fractionStr), bytesRead, nil
		
	case schema.TypeTime:
		// TIME format with possible fractional seconds
		// Storage depends on precision
		fracSize := (col.Precision + 1) / 2
		bufSize := 3 + fracSize
		
		if offset+bufSize > len(input) {
			return nil, 0, fmt.Errorf("short read for TIME")
		}
		
		// Unpack big-endian
		packedValue := uint64(0)
		for i := 0; i < bufSize; i++ {
			packedValue = (packedValue << 8) | uint64(input[offset+bufSize-1-i])
		}
		
		// Extract sign bit position based on fractional size
		fracBits := fracSize * 8
		fracMask := (uint64(1) << fracBits) - 1
		signPos := fracBits + 23
		signVal := uint64(1) << signPos
		
		isNegative := (packedValue & signVal) == 0
		if isNegative {
			packedValue = signVal - packedValue
		}
		
		usec := int(packedValue & fracMask)
		packedValue >>= fracBits
		
		second := int(packedValue & 0x3F)
		packedValue >>= 6
		minute := int(packedValue & 0x3F)
		packedValue >>= 6
		hour := int(packedValue & 0x3FF)
		
		// Adjust microseconds based on precision
		prec := col.Precision
		for prec < 6 {
			usec *= 100
			prec += 2
		}
		
		fractionStr := ""
		if col.Precision > 0 && usec > 0 {
			fractionStr = fmt.Sprintf(".%06d", usec)[:col.Precision+1]
		}
		
		sign := ""
		if isNegative {
			sign = "-"
		}
		
		return fmt.Sprintf("%s%02d:%02d:%02d%s", sign, hour, minute, second, fractionStr), bufSize, nil
		
	case schema.TypeYear:
		// YEAR is 1 byte
		val, err := p.readUint8(input, offset)
		if err != nil {
			return nil, 0, err
		}
		
		if val == 0 {
			return uint16(0), 1, nil
		}
		
		return uint16(uint16(val) + 1900), 1, nil
		
	default:
		return nil, 0, schema.ErrUnsupportedType
	}
}

// Skip skips date/time value without parsing
func (p *DateTimeParser) Skip(input []byte, offset int, col *schema.Column, varLen int) (int, error) {
	switch col.Type {
	case schema.TypeDate:
		return 3, nil
	case schema.TypeTimestamp:
		return 4 + (col.Precision+1)/2, nil
	case schema.TypeDateTime:
		return 5 + (col.Precision+1)/2, nil
	case schema.TypeTime:
		return 3 + (col.Precision+1)/2, nil
	case schema.TypeYear:
		return 1, nil
	default:
		return 0, schema.ErrUnsupportedType
	}
}

// readFraction reads fractional seconds
func (p *DateTimeParser) readFraction(input []byte, offset int, precision int) (int, error) {
	if precision <= 0 {
		return 0, nil
	}
	
	bufsz := (precision + 1) / 2
	if offset+bufsz > len(input) {
		return 0, fmt.Errorf("short read for fraction")
	}
	
	// Unpack big-endian
	usec := uint64(0)
	for i := 0; i < bufsz; i++ {
		usec = (usec << 8) | uint64(input[offset+bufsz-1-i])
	}
	
	// Normalize to microseconds
	prec := precision
	for prec < 6 {
		usec *= 100
		prec += 2
	}
	
	return int(usec), nil
}
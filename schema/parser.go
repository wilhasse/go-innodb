// parser.go - Parse CREATE TABLE SQL statements to extract schema
package schema

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/xwb1989/sqlparser"
)

// ParseTableDefFromSQL parses a CREATE TABLE statement and returns TableDef
func ParseTableDefFromSQL(sql string) (*TableDef, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parse SQL failed: %w", err)
	}

	ddl, ok := stmt.(*sqlparser.DDL)
	if !ok || ddl.Action != sqlparser.CreateStr {
		return nil, fmt.Errorf("statement is not CREATE TABLE")
	}

	tableName := ddl.Table.Name.String()
	tableDef := NewTableDef(tableName)
	
	if ddl.TableSpec == nil {
		return nil, fmt.Errorf("no table spec in CREATE TABLE")
	}

	// Parse columns
	var primaryKeys []string
	for _, col := range ddl.TableSpec.Columns {
		column, err := parseColumn(col)
		if err != nil {
			return nil, fmt.Errorf("parse column %s failed: %w", col.Name, err)
		}

		if err := tableDef.AddColumn(column); err != nil {
			return nil, err
		}

		// Check for primary key in column definition
		// Note: sqlparser may not expose this directly, we'll rely on indexes instead</		
	}

	// Parse table options
	// Note: Options parsing may vary depending on sqlparser version
	// For now, set defaults
	tableDef.Engine = "InnoDB"
	tableDef.Charset = "utf8mb4"

	// Parse indexes for primary keys
	for _, idx := range ddl.TableSpec.Indexes {
		if idx.Info.Primary {
			primaryKeys = nil // Override column-level primary keys
			for _, col := range idx.Columns {
				primaryKeys = append(primaryKeys, col.Column.String())
			}
		}
	}

	if len(primaryKeys) > 0 {
		if err := tableDef.SetPrimaryKeys(primaryKeys); err != nil {
			return nil, err
		}
	}

	return tableDef, nil
}

// ParseTableDefFromSQLFile reads and parses CREATE TABLE from a SQL file
func ParseTableDefFromSQLFile(filename string) (*TableDef, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read SQL file failed: %w", err)
	}

	return ParseTableDefFromSQL(string(content))
}

// parseColumn converts sqlparser.ColumnDefinition to our Column type
func parseColumn(col *sqlparser.ColumnDefinition) (*Column, error) {
	column := &Column{
		Name: col.Name.String(),
	}

	// Parse column type
	typeName := strings.ToUpper(col.Type.Type)
	column.Type = ColumnType(typeName)

	// Parse length/precision/scale
	if col.Type.Length != nil {
		length, err := strconv.Atoi(string(col.Type.Length.Val))
		if err == nil {
			column.Length = length
			column.Precision = length // For DECIMAL, etc.
		}
	}

	if col.Type.Scale != nil {
		scale, err := strconv.Atoi(string(col.Type.Scale.Val))
		if err == nil {
			column.Scale = scale
		}
	}

	// Parse type modifiers
	column.Unsigned = bool(col.Type.Unsigned)
	column.Nullable = !bool(col.Type.NotNull)
	column.AutoIncrement = bool(col.Type.Autoincrement)

	// Parse charset and collation
	if col.Type.Charset != "" {
		column.Charset = col.Type.Charset
	}
	if col.Type.Collate != "" {
		column.Collation = col.Type.Collate
	}

	// Parse default value
	if col.Type.Default != nil {
		column.DefaultValue = sqlparser.String(col.Type.Default)
	}

	// Parse ENUM values
	if column.Type == TypeEnum && col.Type.EnumValues != nil {
		for _, val := range col.Type.EnumValues {
			// Remove quotes from enum values
			enumVal := strings.Trim(val, "'\"")
			column.EnumValues = append(column.EnumValues, enumVal)
		}
	}

	// Handle special type mappings
	column.Type = normalizeColumnType(column.Type, column.Length)

	// Set default charset if not specified
	if column.Charset == "" && isStringType(column.Type) {
		column.Charset = "utf8mb4" // Default for MySQL 8.0+
	}

	return column, nil
}

// normalizeColumnType normalizes column types to our standard types
func normalizeColumnType(colType ColumnType, length int) ColumnType {
	switch strings.ToUpper(string(colType)) {
	case "INTEGER":
		return TypeInt
	case "DOUBLE PRECISION", "REAL":
		return TypeDouble
	case "DEC":
		return TypeDecimal
	case "BOOL":
		return TypeBoolean
	case "TINYINT":
		if length == 1 {
			return TypeBoolean // TINYINT(1) is often used as boolean
		}
		return TypeTinyInt
	default:
		return colType
	}
}

// isStringType checks if a column type is a string type
func isStringType(colType ColumnType) bool {
	switch colType {
	case TypeChar, TypeVarchar,
		TypeText, TypeTinyText, TypeMediumText, TypeLongText:
		return true
	default:
		return false
	}
}
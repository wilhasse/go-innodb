# Test Data

This directory contains test InnoDB data files (.ibd) and related files for testing the go-innodb parser.

## Directory Structure

```
testdata/
├── users/          # Simple users table example
│   ├── users.ibd           # InnoDB data file
│   ├── users.sql           # Table creation SQL
│   ├── users_data.txt      # Sample data (CSV format)
│   └── ...
└── README.md      # This file
```

## Test Files

### users/
A simple users table with basic columns (id, name, email, created_at) containing 3 test records:
- Alice (alice@example.com)
- Bob (bob@example.com)
- Charlie (charlie@example.com)

## Usage Examples

```bash
# Parse the users table data
./go-innodb -file testdata/users/users.ibd -page 4 -records -v

# Show page summary
./go-innodb -file testdata/users/users.ibd -page 4 -format summary

# Export as JSON
./go-innodb -file testdata/users/users.ibd -page 4 -format json -records
```

## Adding New Test Files

When adding new test files:
1. Create a new subdirectory under `testdata/`
2. Include the .ibd file
3. Include the SQL schema file (.sql)
4. Document the table structure and sample data
5. Add usage examples

## Note

These test files are tracked in git (see .gitignore exception for `testdata/**/*.ibd`) to ensure consistent testing across environments.
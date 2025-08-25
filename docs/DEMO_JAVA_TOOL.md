# Java InnoDB Reader Demo

## About the Java Tool
The `demo-java-innodb-reader.sh` script demonstrates the Java implementation of InnoDB file reader from the Alibaba project: https://github.com/alibaba/innodb-java-reader

This is different from the Go implementation in this repository - it's a separate Java-based tool that provides similar functionality.

## Prerequisites
The script requires the Java tool to be built first:

1. **Java 17+** and **Maven** must be installed
2. Build the Java project:
```bash
cd innodb-java-reader
mvn clean install -DskipTests -Dpmd.skip=true -Dcheckstyle.skip=true
```

This will create the necessary JAR file at:
`innodb-java-reader/innodb-java-reader-cli/target/innodb-java-reader-cli.jar`

## Running the Demo
```bash
./demo-java-innodb-reader.sh
```

## What It Demonstrates
- Reading InnoDB page structure
- Extracting all table data
- Querying by primary key
- Range queries
- Exporting to CSV/TSV formats
- Direct page-level access

## Test Data
Uses the sample users table in `testdata/users/`:
- `users.ibd` - InnoDB data file
- `users.sql` - Table structure (modified to use compatible collation)

## Note on Dependencies
This script is a demonstration of the **Java** tool, not the Go implementation. It requires:
- The compiled JAR file from the innodb-java-reader project
- Java runtime to execute the JAR
- The test InnoDB files in testdata/users/

## Comparison with Go Implementation
While this repository contains a Go implementation for reading InnoDB files, the Java tool provides:
- More mature feature set
- Command-line interface with various export options
- Support for different query types (primary key, range, page-level)
- Multiple output formats (CSV, TSV, etc.)

Both tools can read InnoDB files directly without requiring a running MySQL server.
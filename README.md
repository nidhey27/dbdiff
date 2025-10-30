# dbdiff - Database Schema Comparison Tool

A fast, clean CLI tool written in Go that compares two databases (PostgreSQL/MySQL) for schema-level differences.

## Features

Compares the following schema elements:

- **Tables** - presence/absence
- **Columns** - data type, nullability, default values
- **Primary Keys** - columns
- **Foreign Keys** - columns, referenced table/columns, ON DELETE/UPDATE rules
- **Unique Constraints** - columns
- **Indexes** - name, columns, uniqueness
- **Check Constraints** - expressions (where supported)

## Supported Databases

- PostgreSQL
- MySQL

## Installation

```bash
# Clone or download the repository
cd dbdiff

# Install dependencies
go mod download

# Build the binary
go build -o dbdiff main.go

# Optional: Install globally
go install
```

## Usage

### Basic Usage

```bash
dbdiff \
  --source <source-connection-string> \
  --source-driver <postgres|mysql> \
  --target <target-connection-string> \
  --target-driver <postgres|mysql> \
  [--json]
```

### PostgreSQL Example

```bash
dbdiff \
  --source "postgres://user:pass@localhost:5432/db1?sslmode=disable" \
  --source-driver postgres \
  --target "postgres://user:pass@localhost:5432/db2?sslmode=disable" \
  --target-driver postgres
```

### MySQL Example

```bash
dbdiff \
  --source "user:pass@tcp(localhost:3306)/db1?parseTime=true" \
  --source-driver mysql \
  --target "user:pass@tcp(localhost:3306)/db2?parseTime=true" \
  --target-driver mysql
```

### Cross-Database Comparison

You can compare schemas across different database types:

```bash
dbdiff \
  --source "postgres://user:pass@localhost:5432/pgdb?sslmode=disable" \
  --source-driver postgres \
  --target "user:pass@tcp(localhost:3306)/mysqldb?parseTime=true" \
  --target-driver mysql
```

### JSON Output

For machine-readable output, use the `--json` flag:

```bash
dbdiff \
  --source "..." \
  --source-driver postgres \
  --target "..." \
  --target-driver postgres \
  --json
```

## Exit Codes

- `0` - No differences found
- `1` - Error occurred
- `2` - Differences found

This makes it easy to use in CI/CD pipelines:

```bash
if dbdiff --source "$SOURCE" --source-driver postgres --target "$TARGET" --target-driver postgres; then
  echo "Schemas match!"
else
  echo "Schema drift detected!"
  exit 1
fi
```

## Output Format

### Pretty Text Output (Default)

```
Schema Differences Found:
================================================================================

📋 Tables only in SOURCE:
  - old_table

📋 Tables only in TARGET:
  + new_table

📊 Table: users
--------------------------------------------------------------------------------
  Columns only in TARGET:
    + email_verified

  Column differences:
    ~ age: type: integer → bigint
    ~ status: nullable: false → true

  Primary Key: columns: [id] → [id email]

  Foreign Keys only in TARGET:
    + fk_users_roles

  Indexes differences:
    ~ idx_email: unique: false → true
```

### JSON Output

```json
{
  "tables_only_in_source": ["old_table"],
  "tables_only_in_target": ["new_table"],
  "table_diffs": [
    {
      "table_name": "users",
      "columns_only_in_target": ["email_verified"],
      "column_diffs": [
        {
          "column_name": "age",
          "diff": "type: integer → bigint"
        }
      ]
    }
  ]
}
```

## Architecture

The tool is structured with clean separation of concerns:

- **Models** - Schema representation (Table, Column, PrimaryKey, ForeignKey, etc.)
- **Dialects** - Database-specific schema extraction (PostgresDialect, MySQLDialect)
- **Diff Engine** - Schema comparison logic
- **Output** - Pretty text and JSON formatters
- **CLI** - Command-line interface and main orchestration

## Extending

To add support for a new database:

1. Implement the `Dialect` interface:
   ```go
   type Dialect interface {
       ExtractSchema(db *sql.DB) (*Schema, error)
   }
   ```

2. Add the dialect to `getDialect()` function

3. Import the appropriate database driver

## Testing

To test the tool, you'll need running database instances. Here's a quick setup using Docker:

### PostgreSQL Test Setup

```bash
# Start two PostgreSQL instances
docker run -d --name pg1 -e POSTGRES_PASSWORD=pass -p 5432:5432 postgres:15
docker run -d --name pg2 -e POSTGRES_PASSWORD=pass -p 5433:5432 postgres:15

# Create test schemas
psql "postgres://postgres:pass@localhost:5432/postgres" -c "CREATE TABLE users (id INT PRIMARY KEY, name TEXT);"
psql "postgres://postgres:pass@localhost:5433/postgres" -c "CREATE TABLE users (id BIGINT PRIMARY KEY, name TEXT, email TEXT);"

# Run comparison
./dbdiff \
  --source "postgres://postgres:pass@localhost:5432/postgres?sslmode=disable" \
  --source-driver postgres \
  --target "postgres://postgres:pass@localhost:5433/postgres?sslmode=disable" \
  --target-driver postgres
```

### MySQL Test Setup

```bash
# Start two MySQL instances
docker run -d --name mysql1 -e MYSQL_ROOT_PASSWORD=pass -p 3306:3306 mysql:8
docker run -d --name mysql2 -e MYSQL_ROOT_PASSWORD=pass -p 3307:3306 mysql:8

# Create test schemas
mysql -h 127.0.0.1 -P 3306 -u root -ppass -e "CREATE DATABASE testdb; USE testdb; CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));"
mysql -h 127.0.0.1 -P 3307 -u root -ppass -e "CREATE DATABASE testdb; USE testdb; CREATE TABLE users (id BIGINT PRIMARY KEY, name VARCHAR(255), email VARCHAR(255));"

# Run comparison
./dbdiff \
  --source "root:pass@tcp(127.0.0.1:3306)/testdb?parseTime=true" \
  --source-driver mysql \
  --target "root:pass@tcp(127.0.0.1:3307)/testdb?parseTime=true" \
  --target-driver mysql
```

## License

MIT

## Contributing

Contributions welcome! This is a production-ready MVP that can be extended with:

- More database support (SQLite, SQL Server, Oracle, etc.)
- Schema migration generation
- Filtering options (ignore certain tables/columns)
- Configuration file support
- Parallel schema extraction for large databases
- Data comparison (not just schema)


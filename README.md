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

### Option 1: Download Pre-built Binaries (Recommended)

Download the latest release for your platform from the [Releases page](https://github.com/YOUR_USERNAME/dbdiff/releases).

**Linux (AMD64)**
```bash
wget https://github.com/YOUR_USERNAME/dbdiff/releases/latest/download/dbdiff-linux-amd64
chmod +x dbdiff-linux-amd64
sudo mv dbdiff-linux-amd64 /usr/local/bin/dbdiff
```

**Linux (ARM64)**
```bash
wget https://github.com/YOUR_USERNAME/dbdiff/releases/latest/download/dbdiff-linux-arm64
chmod +x dbdiff-linux-arm64
sudo mv dbdiff-linux-arm64 /usr/local/bin/dbdiff
```

**macOS (Intel)**
```bash
wget https://github.com/YOUR_USERNAME/dbdiff/releases/latest/download/dbdiff-darwin-amd64
chmod +x dbdiff-darwin-amd64
sudo mv dbdiff-darwin-amd64 /usr/local/bin/dbdiff
```

**macOS (Apple Silicon)**
```bash
wget https://github.com/YOUR_USERNAME/dbdiff/releases/latest/download/dbdiff-darwin-arm64
chmod +x dbdiff-darwin-arm64
sudo mv dbdiff-darwin-arm64 /usr/local/bin/dbdiff
```

**Windows (AMD64)**
```powershell
# Download dbdiff-windows-amd64.exe from releases
# Add to your PATH or run directly
```

**Windows (ARM64)**
```powershell
# Download dbdiff-windows-arm64.exe from releases
# Add to your PATH or run directly
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/dbdiff.git
cd dbdiff

# Install dependencies
go mod download

# Build for current platform
make build

# Or build for all platforms
make build-all

# Optional: Install globally
make install
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

## Creating a Release

### For Maintainers

To create a new release with pre-built binaries:

1. **Tag the release:**
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

2. **GitHub Actions will automatically:**
   - Build binaries for all platforms (Linux, macOS, Windows - AMD64 & ARM64)
   - Generate SHA256 checksums
   - Create a GitHub release with all binaries attached
   - Add installation instructions to the release notes

3. **Manual local build (optional):**
   ```bash
   # Build all platform binaries locally
   make release VERSION=1.0.0

   # Binaries will be in build/ directory
   ls -lh build/
   ```

### Available Platforms

- **Linux**: AMD64, ARM64
- **macOS**: AMD64 (Intel), ARM64 (Apple Silicon)
- **Windows**: AMD64, ARM64

## Development

### Makefile Commands

```bash
make help          # Show all available commands
make build         # Build for current platform
make build-all     # Build for all platforms
make checksums     # Generate SHA256 checksums
make install       # Install to /usr/local/bin
make clean         # Remove build artifacts
make test          # Run tests
make deps          # Download dependencies
make release       # Create release build (requires VERSION)
```

### Project Structure

```
dbdiff/
├── main.go                      # Complete implementation (~1,300 lines)
├── go.mod                       # Go module definition
├── Makefile                     # Build automation
├── .github/workflows/release.yml # Automated release workflow
└── README.md                    # Documentation
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
- Homebrew formula for easier macOS installation
- Chocolatey package for easier Windows installation


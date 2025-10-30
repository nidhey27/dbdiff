package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// ============================================================================
// MODELS - Schema representation
// ============================================================================

type Schema struct {
	Tables map[string]*Table `json:"tables"`
}

type Table struct {
	Name              string                   `json:"name"`
	Columns           map[string]*Column       `json:"columns"`
	PrimaryKey        *PrimaryKey              `json:"primary_key,omitempty"`
	ForeignKeys       map[string]*ForeignKey   `json:"foreign_keys"`
	UniqueConstraints map[string]*Unique       `json:"unique_constraints"`
	Indexes           map[string]*Index        `json:"indexes"`
	CheckConstraints  map[string]*CheckConstr  `json:"check_constraints"`
}

type Column struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	IsNullable   bool    `json:"is_nullable"`
	DefaultValue *string `json:"default_value,omitempty"`
}

type PrimaryKey struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

type ForeignKey struct {
	Name            string   `json:"name"`
	Columns         []string `json:"columns"`
	RefTable        string   `json:"ref_table"`
	RefColumns      []string `json:"ref_columns"`
	OnDelete        string   `json:"on_delete"`
	OnUpdate        string   `json:"on_update"`
}

type Unique struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

type Index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	IsUnique bool     `json:"is_unique"`
}

type CheckConstr struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// ============================================================================
// FILTER CONFIG - Filtering options
// ============================================================================

type FilterConfig struct {
	IgnoreTables       []string // Exact table names to ignore
	IgnoreTablePattern *regexp.Regexp // Regex pattern for table names to ignore
	IgnoreColumns      map[string][]string // Map of table -> columns to ignore
	IgnoreIndexes      bool // Ignore all index differences
	IgnoreForeignKeys  bool // Ignore all foreign key differences
	IgnoreChecks       bool // Ignore all check constraint differences
}

func NewFilterConfig() *FilterConfig {
	return &FilterConfig{
		IgnoreTables:  []string{},
		IgnoreColumns: make(map[string][]string),
	}
}

func (fc *FilterConfig) ShouldIgnoreTable(tableName string) bool {
	// Check exact matches
	for _, t := range fc.IgnoreTables {
		if t == tableName {
			return true
		}
	}
	// Check pattern
	if fc.IgnoreTablePattern != nil && fc.IgnoreTablePattern.MatchString(tableName) {
		return true
	}
	return false
}

func (fc *FilterConfig) ShouldIgnoreColumn(tableName, columnName string) bool {
	if cols, ok := fc.IgnoreColumns[tableName]; ok {
		for _, c := range cols {
			if c == columnName {
				return true
			}
		}
	}
	return false
}

// ============================================================================
// DIFF - Difference representation
// ============================================================================

type SchemaDiff struct {
	TablesOnlyInSource []string     `json:"tables_only_in_source,omitempty"`
	TablesOnlyInTarget []string     `json:"tables_only_in_target,omitempty"`
	TableDiffs         []*TableDiff `json:"table_diffs,omitempty"`
}

type TableDiff struct {
	TableName              string        `json:"table_name"`
	ColumnsOnlyInSource    []string      `json:"columns_only_in_source,omitempty"`
	ColumnsOnlyInTarget    []string      `json:"columns_only_in_target,omitempty"`
	ColumnDiffs            []*ColumnDiff `json:"column_diffs,omitempty"`
	PrimaryKeyDiff         *string       `json:"primary_key_diff,omitempty"`
	ForeignKeysOnlyInSource []string     `json:"foreign_keys_only_in_source,omitempty"`
	ForeignKeysOnlyInTarget []string     `json:"foreign_keys_only_in_target,omitempty"`
	ForeignKeyDiffs        []*FKDiff     `json:"foreign_key_diffs,omitempty"`
	UniquesOnlyInSource    []string      `json:"uniques_only_in_source,omitempty"`
	UniquesOnlyInTarget    []string      `json:"uniques_only_in_target,omitempty"`
	UniqueDiffs            []*UniqueDiff `json:"unique_diffs,omitempty"`
	IndexesOnlyInSource    []string      `json:"indexes_only_in_source,omitempty"`
	IndexesOnlyInTarget    []string      `json:"indexes_only_in_target,omitempty"`
	IndexDiffs             []*IndexDiff  `json:"index_diffs,omitempty"`
	ChecksOnlyInSource     []string      `json:"checks_only_in_source,omitempty"`
	ChecksOnlyInTarget     []string      `json:"checks_only_in_target,omitempty"`
	CheckDiffs             []*CheckDiff  `json:"check_diffs,omitempty"`
}

type ColumnDiff struct {
	ColumnName string `json:"column_name"`
	Diff       string `json:"diff"`
}

type FKDiff struct {
	Name string `json:"name"`
	Diff string `json:"diff"`
}

type UniqueDiff struct {
	Name string `json:"name"`
	Diff string `json:"diff"`
}

type IndexDiff struct {
	Name string `json:"name"`
	Diff string `json:"diff"`
}

type CheckDiff struct {
	Name string `json:"name"`
	Diff string `json:"diff"`
}

// ============================================================================
// DIALECT INTERFACE - Database-specific schema extraction
// ============================================================================

type Dialect interface {
	ExtractSchema(db *sql.DB) (*Schema, error)
	ExtractSchemaParallel(db *sql.DB) (*Schema, error)
}

// ============================================================================
// POSTGRES DIALECT
// ============================================================================

type PostgresDialect struct{}

func (p *PostgresDialect) ExtractSchema(db *sql.DB) (*Schema, error) {
	schema := &Schema{Tables: make(map[string]*Table)}

	// Get all tables
	tables, err := p.getTables(db)
	if err != nil {
		return nil, err
	}

	for _, tableName := range tables {
		table := &Table{
			Name:              tableName,
			Columns:           make(map[string]*Column),
			ForeignKeys:       make(map[string]*ForeignKey),
			UniqueConstraints: make(map[string]*Unique),
			Indexes:           make(map[string]*Index),
			CheckConstraints:  make(map[string]*CheckConstr),
		}

		// Extract columns
		if err := p.extractColumns(db, tableName, table); err != nil {
			return nil, err
		}

		// Extract primary key
		if err := p.extractPrimaryKey(db, tableName, table); err != nil {
			return nil, err
		}

		// Extract foreign keys
		if err := p.extractForeignKeys(db, tableName, table); err != nil {
			return nil, err
		}

		// Extract unique constraints
		if err := p.extractUniqueConstraints(db, tableName, table); err != nil {
			return nil, err
		}

		// Extract indexes
		if err := p.extractIndexes(db, tableName, table); err != nil {
			return nil, err
		}

		// Extract check constraints
		if err := p.extractCheckConstraints(db, tableName, table); err != nil {
			return nil, err
		}

		schema.Tables[tableName] = table
	}

	return schema, nil
}

func (p *PostgresDialect) ExtractSchemaParallel(db *sql.DB) (*Schema, error) {
	schema := &Schema{Tables: make(map[string]*Table)}

	// Get all tables
	tables, err := p.getTables(db)
	if err != nil {
		return nil, err
	}

	// Use a wait group and mutex for parallel extraction
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(tables))

	for _, tableName := range tables {
		wg.Add(1)
		go func(tName string) {
			defer wg.Done()

			table := &Table{
				Name:              tName,
				Columns:           make(map[string]*Column),
				ForeignKeys:       make(map[string]*ForeignKey),
				UniqueConstraints: make(map[string]*Unique),
				Indexes:           make(map[string]*Index),
				CheckConstraints:  make(map[string]*CheckConstr),
			}

			// Extract all metadata for this table
			if err := p.extractColumns(db, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting columns for %s: %w", tName, err)
				return
			}

			if err := p.extractPrimaryKey(db, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting primary key for %s: %w", tName, err)
				return
			}

			if err := p.extractForeignKeys(db, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting foreign keys for %s: %w", tName, err)
				return
			}

			if err := p.extractUniqueConstraints(db, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting unique constraints for %s: %w", tName, err)
				return
			}

			if err := p.extractIndexes(db, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting indexes for %s: %w", tName, err)
				return
			}

			if err := p.extractCheckConstraints(db, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting check constraints for %s: %w", tName, err)
				return
			}

			// Safely add to schema
			mu.Lock()
			schema.Tables[tName] = table
			mu.Unlock()
		}(tableName)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return schema, nil
}

func (p *PostgresDialect) getTables(db *sql.DB) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (p *PostgresDialect) extractColumns(db *sql.DB, tableName string, table *Table) error {
	query := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, dataType, isNullable string
		var defaultVal sql.NullString
		if err := rows.Scan(&name, &dataType, &isNullable, &defaultVal); err != nil {
			return err
		}

		col := &Column{
			Name:       name,
			DataType:   dataType,
			IsNullable: isNullable == "YES",
		}
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}
		table.Columns[name] = col
	}
	return rows.Err()
}

func (p *PostgresDialect) extractPrimaryKey(db *sql.DB, tableName string, table *Table) error {
	query := `
		SELECT
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = $1
		  AND tc.constraint_type = 'PRIMARY KEY'
		GROUP BY tc.constraint_name
	`
	var name string
	var columns string
	err := db.QueryRow(query, tableName).Scan(&name, &columns)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	// Parse PostgreSQL array format: {col1,col2}
	cols := strings.Trim(columns, "{}")
	if cols != "" {
		table.PrimaryKey = &PrimaryKey{
			Name:    name,
			Columns: strings.Split(cols, ","),
		}
	}
	return nil
}

func (p *PostgresDialect) extractForeignKeys(db *sql.DB, tableName string, table *Table) error {
	query := `
		SELECT
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns,
			ccu.table_name AS foreign_table_name,
			array_agg(ccu.column_name ORDER BY kcu.ordinal_position) as foreign_columns,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = $1
		  AND tc.constraint_type = 'FOREIGN KEY'
		GROUP BY tc.constraint_name, ccu.table_name, rc.update_rule, rc.delete_rule
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, columns, refTable, refColumns, updateRule, deleteRule string
		if err := rows.Scan(&name, &columns, &refTable, &refColumns, &updateRule, &deleteRule); err != nil {
			return err
		}

		cols := strings.Trim(columns, "{}")
		refCols := strings.Trim(refColumns, "{}")

		fk := &ForeignKey{
			Name:       name,
			Columns:    strings.Split(cols, ","),
			RefTable:   refTable,
			RefColumns: strings.Split(refCols, ","),
			OnUpdate:   updateRule,
			OnDelete:   deleteRule,
		}
		table.ForeignKeys[name] = fk
	}
	return rows.Err()
}

func (p *PostgresDialect) extractUniqueConstraints(db *sql.DB, tableName string, table *Table) error {
	query := `
		SELECT
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = $1
		  AND tc.constraint_type = 'UNIQUE'
		GROUP BY tc.constraint_name
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, columns string
		if err := rows.Scan(&name, &columns); err != nil {
			return err
		}

		cols := strings.Trim(columns, "{}")
		uniq := &Unique{
			Name:    name,
			Columns: strings.Split(cols, ","),
		}
		table.UniqueConstraints[name] = uniq
	}
	return rows.Err()
}

func (p *PostgresDialect) extractIndexes(db *sql.DB, tableName string, table *Table) error {
	query := `
		SELECT
			i.relname as index_name,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) as columns,
			ix.indisunique
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		LEFT JOIN pg_constraint c ON c.conindid = i.oid
		WHERE t.relname = $1
		  AND t.relkind = 'r'
		  AND c.contype IS NULL  -- Exclude constraint-backed indexes
		GROUP BY i.relname, ix.indisunique
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, columns string
		var isUnique bool
		if err := rows.Scan(&name, &columns, &isUnique); err != nil {
			return err
		}

		cols := strings.Trim(columns, "{}")
		idx := &Index{
			Name:     name,
			Columns:  strings.Split(cols, ","),
			IsUnique: isUnique,
		}
		table.Indexes[name] = idx
	}
	return rows.Err()
}

func (p *PostgresDialect) extractCheckConstraints(db *sql.DB, tableName string, table *Table) error {
	query := `
		SELECT
			con.conname as constraint_name,
			pg_get_constraintdef(con.oid) as check_clause
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		WHERE rel.relname = $1
		  AND con.contype = 'c'
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, expr string
		if err := rows.Scan(&name, &expr); err != nil {
			return err
		}

		check := &CheckConstr{
			Name:       name,
			Expression: expr,
		}
		table.CheckConstraints[name] = check
	}
	return rows.Err()
}

// ============================================================================
// MYSQL DIALECT
// ============================================================================

type MySQLDialect struct{}

func (m *MySQLDialect) ExtractSchema(db *sql.DB) (*Schema, error) {
	schema := &Schema{Tables: make(map[string]*Table)}

	// Get database name
	var dbName string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&dbName); err != nil {
		return nil, err
	}

	// Get all tables
	tables, err := m.getTables(db, dbName)
	if err != nil {
		return nil, err
	}

	for _, tableName := range tables {
		table := &Table{
			Name:              tableName,
			Columns:           make(map[string]*Column),
			ForeignKeys:       make(map[string]*ForeignKey),
			UniqueConstraints: make(map[string]*Unique),
			Indexes:           make(map[string]*Index),
			CheckConstraints:  make(map[string]*CheckConstr),
		}

		// Extract columns
		if err := m.extractColumns(db, dbName, tableName, table); err != nil {
			return nil, err
		}

		// Extract primary key
		if err := m.extractPrimaryKey(db, dbName, tableName, table); err != nil {
			return nil, err
		}

		// Extract foreign keys
		if err := m.extractForeignKeys(db, dbName, tableName, table); err != nil {
			return nil, err
		}

		// Extract unique constraints
		if err := m.extractUniqueConstraints(db, dbName, tableName, table); err != nil {
			return nil, err
		}

		// Extract indexes
		if err := m.extractIndexes(db, dbName, tableName, table); err != nil {
			return nil, err
		}

		// Extract check constraints (MySQL 8.0.16+)
		if err := m.extractCheckConstraints(db, dbName, tableName, table); err != nil {
			// Ignore errors for older MySQL versions
			_ = err
		}

		schema.Tables[tableName] = table
	}

	return schema, nil
}

func (m *MySQLDialect) ExtractSchemaParallel(db *sql.DB) (*Schema, error) {
	schema := &Schema{Tables: make(map[string]*Table)}

	// Get database name
	var dbName string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&dbName); err != nil {
		return nil, err
	}

	// Get all tables
	tables, err := m.getTables(db, dbName)
	if err != nil {
		return nil, err
	}

	// Use a wait group and mutex for parallel extraction
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(tables))

	for _, tableName := range tables {
		wg.Add(1)
		go func(tName string) {
			defer wg.Done()

			table := &Table{
				Name:              tName,
				Columns:           make(map[string]*Column),
				ForeignKeys:       make(map[string]*ForeignKey),
				UniqueConstraints: make(map[string]*Unique),
				Indexes:           make(map[string]*Index),
				CheckConstraints:  make(map[string]*CheckConstr),
			}

			// Extract all metadata for this table
			if err := m.extractColumns(db, dbName, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting columns for %s: %w", tName, err)
				return
			}

			if err := m.extractPrimaryKey(db, dbName, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting primary key for %s: %w", tName, err)
				return
			}

			if err := m.extractForeignKeys(db, dbName, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting foreign keys for %s: %w", tName, err)
				return
			}

			if err := m.extractUniqueConstraints(db, dbName, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting unique constraints for %s: %w", tName, err)
				return
			}

			if err := m.extractIndexes(db, dbName, tName, table); err != nil {
				errChan <- fmt.Errorf("error extracting indexes for %s: %w", tName, err)
				return
			}

			// Extract check constraints (MySQL 8.0.16+)
			if err := m.extractCheckConstraints(db, dbName, tName, table); err != nil {
				// Ignore errors for older MySQL versions
				_ = err
			}

			// Safely add to schema
			mu.Lock()
			schema.Tables[tName] = table
			mu.Unlock()
		}(tableName)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return schema, nil
}

func (m *MySQLDialect) getTables(db *sql.DB, dbName string) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (m *MySQLDialect) extractColumns(db *sql.DB, dbName, tableName string, table *Table) error {
	query := `
		SELECT
			column_name,
			column_type,
			is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`
	rows, err := db.Query(query, dbName, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, dataType, isNullable string
		var defaultVal sql.NullString
		if err := rows.Scan(&name, &dataType, &isNullable, &defaultVal); err != nil {
			return err
		}

		col := &Column{
			Name:       name,
			DataType:   dataType,
			IsNullable: isNullable == "YES",
		}
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}
		table.Columns[name] = col
	}
	return rows.Err()
}

func (m *MySQLDialect) extractPrimaryKey(db *sql.DB, dbName, tableName string, table *Table) error {
	query := `
		SELECT
			constraint_name,
			GROUP_CONCAT(column_name ORDER BY ordinal_position) as columns
		FROM information_schema.key_column_usage
		WHERE table_schema = ?
		  AND table_name = ?
		  AND constraint_name = 'PRIMARY'
		GROUP BY constraint_name
	`
	var name string
	var columns sql.NullString
	err := db.QueryRow(query, dbName, tableName).Scan(&name, &columns)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	if columns.Valid && columns.String != "" {
		table.PrimaryKey = &PrimaryKey{
			Name:    name,
			Columns: strings.Split(columns.String, ","),
		}
	}
	return nil
}

func (m *MySQLDialect) extractForeignKeys(db *sql.DB, dbName, tableName string, table *Table) error {
	query := `
		SELECT
			kcu.constraint_name,
			GROUP_CONCAT(kcu.column_name ORDER BY kcu.ordinal_position) as columns,
			kcu.referenced_table_name,
			GROUP_CONCAT(kcu.referenced_column_name ORDER BY kcu.ordinal_position) as ref_columns,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc
			ON kcu.constraint_name = rc.constraint_name
			AND kcu.table_schema = rc.constraint_schema
		WHERE kcu.table_schema = ?
		  AND kcu.table_name = ?
		  AND kcu.referenced_table_name IS NOT NULL
		GROUP BY kcu.constraint_name, kcu.referenced_table_name, rc.update_rule, rc.delete_rule
	`
	rows, err := db.Query(query, dbName, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, columns, refTable, refColumns, updateRule, deleteRule string
		if err := rows.Scan(&name, &columns, &refTable, &refColumns, &updateRule, &deleteRule); err != nil {
			return err
		}

		fk := &ForeignKey{
			Name:       name,
			Columns:    strings.Split(columns, ","),
			RefTable:   refTable,
			RefColumns: strings.Split(refColumns, ","),
			OnUpdate:   updateRule,
			OnDelete:   deleteRule,
		}
		table.ForeignKeys[name] = fk
	}
	return rows.Err()
}

func (m *MySQLDialect) extractUniqueConstraints(db *sql.DB, dbName, tableName string, table *Table) error {
	query := `
		SELECT
			constraint_name,
			GROUP_CONCAT(column_name ORDER BY ordinal_position) as columns
		FROM information_schema.key_column_usage
		WHERE table_schema = ?
		  AND table_name = ?
		  AND constraint_name != 'PRIMARY'
		  AND constraint_name IN (
			SELECT constraint_name
			FROM information_schema.table_constraints
			WHERE table_schema = ?
			  AND table_name = ?
			  AND constraint_type = 'UNIQUE'
		  )
		GROUP BY constraint_name
	`
	rows, err := db.Query(query, dbName, tableName, dbName, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, columns string
		if err := rows.Scan(&name, &columns); err != nil {
			return err
		}

		uniq := &Unique{
			Name:    name,
			Columns: strings.Split(columns, ","),
		}
		table.UniqueConstraints[name] = uniq
	}
	return rows.Err()
}

func (m *MySQLDialect) extractIndexes(db *sql.DB, dbName, tableName string, table *Table) error {
	query := `
		SELECT
			index_name,
			GROUP_CONCAT(column_name ORDER BY seq_in_index) as columns,
			MAX(non_unique) as non_unique
		FROM information_schema.statistics
		WHERE table_schema = ?
		  AND table_name = ?
		  AND index_name != 'PRIMARY'
		  AND index_name NOT IN (
			SELECT constraint_name
			FROM information_schema.table_constraints
			WHERE table_schema = ?
			  AND table_name = ?
			  AND constraint_type IN ('UNIQUE', 'FOREIGN KEY')
		  )
		GROUP BY index_name
	`
	rows, err := db.Query(query, dbName, tableName, dbName, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, columns string
		var nonUnique int
		if err := rows.Scan(&name, &columns, &nonUnique); err != nil {
			return err
		}

		idx := &Index{
			Name:     name,
			Columns:  strings.Split(columns, ","),
			IsUnique: nonUnique == 0,
		}
		table.Indexes[name] = idx
	}
	return rows.Err()
}

func (m *MySQLDialect) extractCheckConstraints(db *sql.DB, dbName, tableName string, table *Table) error {
	query := `
		SELECT
			constraint_name,
			check_clause
		FROM information_schema.check_constraints
		WHERE constraint_schema = ?
		  AND constraint_name IN (
			SELECT constraint_name
			FROM information_schema.table_constraints
			WHERE table_schema = ?
			  AND table_name = ?
			  AND constraint_type = 'CHECK'
		  )
	`
	rows, err := db.Query(query, dbName, dbName, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, expr string
		if err := rows.Scan(&name, &expr); err != nil {
			return err
		}

		check := &CheckConstr{
			Name:       name,
			Expression: expr,
		}
		table.CheckConstraints[name] = check
	}
	return rows.Err()
}

// ============================================================================
// DIFF ENGINE
// ============================================================================

func ComputeDiff(source, target *Schema, filter *FilterConfig) *SchemaDiff {
	diff := &SchemaDiff{}

	// Find tables only in source or target
	sourceTableNames := getSortedKeys(source.Tables)
	targetTableNames := getSortedKeys(target.Tables)

	sourceSet := makeSet(sourceTableNames)
	targetSet := makeSet(targetTableNames)

	for _, name := range sourceTableNames {
		if !targetSet[name] && !filter.ShouldIgnoreTable(name) {
			diff.TablesOnlyInSource = append(diff.TablesOnlyInSource, name)
		}
	}

	for _, name := range targetTableNames {
		if !sourceSet[name] && !filter.ShouldIgnoreTable(name) {
			diff.TablesOnlyInTarget = append(diff.TablesOnlyInTarget, name)
		}
	}

	// Compare common tables
	for _, tableName := range sourceTableNames {
		if targetSet[tableName] && !filter.ShouldIgnoreTable(tableName) {
			tableDiff := compareTable(source.Tables[tableName], target.Tables[tableName], filter)
			if !isTableDiffEmpty(tableDiff) {
				diff.TableDiffs = append(diff.TableDiffs, tableDiff)
			}
		}
	}

	return diff
}

func compareTable(source, target *Table, filter *FilterConfig) *TableDiff {
	diff := &TableDiff{TableName: source.Name}

	// Compare columns
	sourceColNames := getSortedKeys(source.Columns)
	targetColNames := getSortedKeys(target.Columns)

	sourceColSet := makeSet(sourceColNames)
	targetColSet := makeSet(targetColNames)

	for _, name := range sourceColNames {
		if !targetColSet[name] && !filter.ShouldIgnoreColumn(source.Name, name) {
			diff.ColumnsOnlyInSource = append(diff.ColumnsOnlyInSource, name)
		}
	}

	for _, name := range targetColNames {
		if !sourceColSet[name] && !filter.ShouldIgnoreColumn(target.Name, name) {
			diff.ColumnsOnlyInTarget = append(diff.ColumnsOnlyInTarget, name)
		}
	}

	for _, colName := range sourceColNames {
		if targetColSet[colName] && !filter.ShouldIgnoreColumn(source.Name, colName) {
			colDiff := compareColumn(source.Columns[colName], target.Columns[colName])
			if colDiff != "" {
				diff.ColumnDiffs = append(diff.ColumnDiffs, &ColumnDiff{
					ColumnName: colName,
					Diff:       colDiff,
				})
			}
		}
	}

	// Compare primary keys
	pkDiff := comparePrimaryKey(source.PrimaryKey, target.PrimaryKey)
	if pkDiff != "" {
		diff.PrimaryKeyDiff = &pkDiff
	}

	// Compare foreign keys
	if !filter.IgnoreForeignKeys {
		compareMaps(
			source.ForeignKeys, target.ForeignKeys,
			&diff.ForeignKeysOnlyInSource, &diff.ForeignKeysOnlyInTarget,
			func(s, t *ForeignKey) string { return compareForeignKey(s, t) },
			&diff.ForeignKeyDiffs,
		)
	}

	// Compare unique constraints
	compareMaps(
		source.UniqueConstraints, target.UniqueConstraints,
		&diff.UniquesOnlyInSource, &diff.UniquesOnlyInTarget,
		func(s, t *Unique) string { return compareUnique(s, t) },
		&diff.UniqueDiffs,
	)

	// Compare indexes
	if !filter.IgnoreIndexes {
		compareMaps(
			source.Indexes, target.Indexes,
			&diff.IndexesOnlyInSource, &diff.IndexesOnlyInTarget,
			func(s, t *Index) string { return compareIndex(s, t) },
			&diff.IndexDiffs,
		)
	}

	// Compare check constraints
	if !filter.IgnoreChecks {
		compareMaps(
			source.CheckConstraints, target.CheckConstraints,
			&diff.ChecksOnlyInSource, &diff.ChecksOnlyInTarget,
			func(s, t *CheckConstr) string { return compareCheck(s, t) },
			&diff.CheckDiffs,
		)
	}

	return diff
}

func compareColumn(source, target *Column) string {
	var diffs []string

	if source.DataType != target.DataType {
		diffs = append(diffs, fmt.Sprintf("type: %s → %s", source.DataType, target.DataType))
	}

	if source.IsNullable != target.IsNullable {
		diffs = append(diffs, fmt.Sprintf("nullable: %v → %v", source.IsNullable, target.IsNullable))
	}

	srcDefault := ""
	if source.DefaultValue != nil {
		srcDefault = *source.DefaultValue
	}
	tgtDefault := ""
	if target.DefaultValue != nil {
		tgtDefault = *target.DefaultValue
	}
	if srcDefault != tgtDefault {
		diffs = append(diffs, fmt.Sprintf("default: %q → %q", srcDefault, tgtDefault))
	}

	return strings.Join(diffs, "; ")
}

func comparePrimaryKey(source, target *PrimaryKey) string {
	if source == nil && target == nil {
		return ""
	}
	if source == nil {
		return fmt.Sprintf("added: %v", target.Columns)
	}
	if target == nil {
		return fmt.Sprintf("removed: %v", source.Columns)
	}
	if !equalStringSlices(source.Columns, target.Columns) {
		return fmt.Sprintf("columns: %v → %v", source.Columns, target.Columns)
	}
	return ""
}

func compareForeignKey(source, target *ForeignKey) string {
	var diffs []string

	if !equalStringSlices(source.Columns, target.Columns) {
		diffs = append(diffs, fmt.Sprintf("columns: %v → %v", source.Columns, target.Columns))
	}

	if source.RefTable != target.RefTable {
		diffs = append(diffs, fmt.Sprintf("ref_table: %s → %s", source.RefTable, target.RefTable))
	}

	if !equalStringSlices(source.RefColumns, target.RefColumns) {
		diffs = append(diffs, fmt.Sprintf("ref_columns: %v → %v", source.RefColumns, target.RefColumns))
	}

	if source.OnDelete != target.OnDelete {
		diffs = append(diffs, fmt.Sprintf("on_delete: %s → %s", source.OnDelete, target.OnDelete))
	}

	if source.OnUpdate != target.OnUpdate {
		diffs = append(diffs, fmt.Sprintf("on_update: %s → %s", source.OnUpdate, target.OnUpdate))
	}

	return strings.Join(diffs, "; ")
}

func compareUnique(source, target *Unique) string {
	if !equalStringSlices(source.Columns, target.Columns) {
		return fmt.Sprintf("columns: %v → %v", source.Columns, target.Columns)
	}
	return ""
}

func compareIndex(source, target *Index) string {
	var diffs []string

	if !equalStringSlices(source.Columns, target.Columns) {
		diffs = append(diffs, fmt.Sprintf("columns: %v → %v", source.Columns, target.Columns))
	}

	if source.IsUnique != target.IsUnique {
		diffs = append(diffs, fmt.Sprintf("unique: %v → %v", source.IsUnique, target.IsUnique))
	}

	return strings.Join(diffs, "; ")
}

func compareCheck(source, target *CheckConstr) string {
	if source.Expression != target.Expression {
		return fmt.Sprintf("expression: %s → %s", source.Expression, target.Expression)
	}
	return ""
}

// Generic comparison helper for maps
func compareMaps[T any, D any](
	sourceMap, targetMap map[string]T,
	onlyInSource, onlyInTarget *[]string,
	compareFn func(T, T) string,
	diffs *[]D,
) {
	sourceKeys := getSortedKeys(sourceMap)
	targetKeys := getSortedKeys(targetMap)

	sourceSet := makeSet(sourceKeys)
	targetSet := makeSet(targetKeys)

	for _, key := range sourceKeys {
		if !targetSet[key] {
			*onlyInSource = append(*onlyInSource, key)
		}
	}

	for _, key := range targetKeys {
		if !sourceSet[key] {
			*onlyInTarget = append(*onlyInTarget, key)
		}
	}

	for _, key := range sourceKeys {
		if targetSet[key] {
			diffStr := compareFn(sourceMap[key], targetMap[key])
			if diffStr != "" {
				// Use reflection to create the appropriate diff type
				var diff D
				switch any(diff).(type) {
				case *FKDiff:
					*diffs = append(*diffs, any(&FKDiff{Name: key, Diff: diffStr}).(D))
				case *UniqueDiff:
					*diffs = append(*diffs, any(&UniqueDiff{Name: key, Diff: diffStr}).(D))
				case *IndexDiff:
					*diffs = append(*diffs, any(&IndexDiff{Name: key, Diff: diffStr}).(D))
				case *CheckDiff:
					*diffs = append(*diffs, any(&CheckDiff{Name: key, Diff: diffStr}).(D))
				}
			}
		}
	}
}

// ============================================================================
// MIGRATION GENERATION
// ============================================================================

func GenerateMigrationSQL(diff *SchemaDiff, driver string) string {
	var migrations []string

	// Generate CREATE TABLE statements for tables only in target
	for _, tableName := range diff.TablesOnlyInTarget {
		migrations = append(migrations, fmt.Sprintf("-- Table '%s' exists in target but not in source", tableName))
		migrations = append(migrations, fmt.Sprintf("-- Manual review required for table: %s\n", tableName))
	}

	// Generate DROP TABLE statements for tables only in source
	for _, tableName := range diff.TablesOnlyInSource {
		migrations = append(migrations, fmt.Sprintf("-- DROP TABLE %s;  -- Table exists in source but not in target\n", tableName))
	}

	// Generate ALTER TABLE statements for table differences
	for _, tableDiff := range diff.TableDiffs {
		tableMigrations := generateTableMigrations(tableDiff, driver)
		if len(tableMigrations) > 0 {
			migrations = append(migrations, fmt.Sprintf("-- Migrations for table: %s", tableDiff.TableName))
			migrations = append(migrations, tableMigrations...)
			migrations = append(migrations, "")
		}
	}

	if len(migrations) == 0 {
		return "-- No migrations needed\n"
	}

	header := fmt.Sprintf("-- Migration SQL generated for %s\n", driver)
	header += "-- Review and test these statements before applying to production!\n"
	header += "-- Some statements may need manual adjustment.\n\n"

	return header + strings.Join(migrations, "\n")
}

func generateTableMigrations(diff *TableDiff, driver string) []string {
	var migrations []string

	// Add columns
	for _, colName := range diff.ColumnsOnlyInTarget {
		migrations = append(migrations, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;  -- Column exists in target", diff.TableName, colName))
	}

	// Drop columns
	for _, colName := range diff.ColumnsOnlyInSource {
		migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s DROP COLUMN %s;  -- Column exists in source but not in target", diff.TableName, colName))
	}

	// Modify columns
	for _, colDiff := range diff.ColumnDiffs {
		if driver == "postgres" {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s ALTER COLUMN %s ...;  -- %s", diff.TableName, colDiff.ColumnName, colDiff.Diff))
		} else {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s MODIFY COLUMN %s ...;  -- %s", diff.TableName, colDiff.ColumnName, colDiff.Diff))
		}
	}

	// Add indexes
	for _, idxName := range diff.IndexesOnlyInTarget {
		migrations = append(migrations, fmt.Sprintf("-- CREATE INDEX %s ON %s (...);  -- Index exists in target", idxName, diff.TableName))
	}

	// Drop indexes
	for _, idxName := range diff.IndexesOnlyInSource {
		if driver == "postgres" {
			migrations = append(migrations, fmt.Sprintf("-- DROP INDEX %s;  -- Index exists in source but not in target", idxName))
		} else {
			migrations = append(migrations, fmt.Sprintf("-- DROP INDEX %s ON %s;  -- Index exists in source but not in target", idxName, diff.TableName))
		}
	}

	// Add foreign keys
	for _, fkName := range diff.ForeignKeysOnlyInTarget {
		migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (...) REFERENCES ...;  -- FK exists in target", diff.TableName, fkName))
	}

	// Drop foreign keys
	for _, fkName := range diff.ForeignKeysOnlyInSource {
		if driver == "postgres" {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s DROP CONSTRAINT %s;  -- FK exists in source but not in target", diff.TableName, fkName))
		} else {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s DROP FOREIGN KEY %s;  -- FK exists in source but not in target", diff.TableName, fkName))
		}
	}

	// Add unique constraints
	for _, uqName := range diff.UniquesOnlyInTarget {
		migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (...);  -- Unique constraint exists in target", diff.TableName, uqName))
	}

	// Drop unique constraints
	for _, uqName := range diff.UniquesOnlyInSource {
		if driver == "postgres" {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s DROP CONSTRAINT %s;  -- Unique constraint exists in source but not in target", diff.TableName, uqName))
		} else {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s DROP INDEX %s;  -- Unique constraint exists in source but not in target", diff.TableName, uqName))
		}
	}

	// Add check constraints
	for _, chkName := range diff.ChecksOnlyInTarget {
		migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s ADD CONSTRAINT %s CHECK (...);  -- Check constraint exists in target", diff.TableName, chkName))
	}

	// Drop check constraints
	for _, chkName := range diff.ChecksOnlyInSource {
		if driver == "postgres" {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s DROP CONSTRAINT %s;  -- Check constraint exists in source but not in target", diff.TableName, chkName))
		} else {
			migrations = append(migrations, fmt.Sprintf("-- ALTER TABLE %s DROP CHECK %s;  -- Check constraint exists in source but not in target", diff.TableName, chkName))
		}
	}

	return migrations
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func getSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func makeSet(items []string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range items {
		set[item] = true
	}
	return set
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isTableDiffEmpty(diff *TableDiff) bool {
	return len(diff.ColumnsOnlyInSource) == 0 &&
		len(diff.ColumnsOnlyInTarget) == 0 &&
		len(diff.ColumnDiffs) == 0 &&
		diff.PrimaryKeyDiff == nil &&
		len(diff.ForeignKeysOnlyInSource) == 0 &&
		len(diff.ForeignKeysOnlyInTarget) == 0 &&
		len(diff.ForeignKeyDiffs) == 0 &&
		len(diff.UniquesOnlyInSource) == 0 &&
		len(diff.UniquesOnlyInTarget) == 0 &&
		len(diff.UniqueDiffs) == 0 &&
		len(diff.IndexesOnlyInSource) == 0 &&
		len(diff.IndexesOnlyInTarget) == 0 &&
		len(diff.IndexDiffs) == 0 &&
		len(diff.ChecksOnlyInSource) == 0 &&
		len(diff.ChecksOnlyInTarget) == 0 &&
		len(diff.CheckDiffs) == 0
}

func isDiffEmpty(diff *SchemaDiff) bool {
	return len(diff.TablesOnlyInSource) == 0 &&
		len(diff.TablesOnlyInTarget) == 0 &&
		len(diff.TableDiffs) == 0
}

// ============================================================================
// OUTPUT FORMATTING
// ============================================================================

func PrintDiff(diff *SchemaDiff, asJSON bool) {
	if asJSON {
		printJSON(diff)
		return
	}

	printPretty(diff)
}

func printJSON(diff *SchemaDiff) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(diff); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func printPretty(diff *SchemaDiff) {
	if isDiffEmpty(diff) {
		fmt.Println("✓ No schema differences found")
		return
	}

	fmt.Println("Schema Differences Found:")
	fmt.Println(strings.Repeat("=", 80))

	// Tables only in source
	if len(diff.TablesOnlyInSource) > 0 {
		fmt.Println("\n📋 Tables only in SOURCE:")
		for _, table := range diff.TablesOnlyInSource {
			fmt.Printf("  - %s\n", table)
		}
	}

	// Tables only in target
	if len(diff.TablesOnlyInTarget) > 0 {
		fmt.Println("\n📋 Tables only in TARGET:")
		for _, table := range diff.TablesOnlyInTarget {
			fmt.Printf("  + %s\n", table)
		}
	}

	// Table differences
	for _, tableDiff := range diff.TableDiffs {
		fmt.Printf("\n📊 Table: %s\n", tableDiff.TableName)
		fmt.Println(strings.Repeat("-", 80))

		// Columns
		if len(tableDiff.ColumnsOnlyInSource) > 0 {
			fmt.Println("  Columns only in SOURCE:")
			for _, col := range tableDiff.ColumnsOnlyInSource {
				fmt.Printf("    - %s\n", col)
			}
		}

		if len(tableDiff.ColumnsOnlyInTarget) > 0 {
			fmt.Println("  Columns only in TARGET:")
			for _, col := range tableDiff.ColumnsOnlyInTarget {
				fmt.Printf("    + %s\n", col)
			}
		}

		if len(tableDiff.ColumnDiffs) > 0 {
			fmt.Println("  Column differences:")
			for _, colDiff := range tableDiff.ColumnDiffs {
				fmt.Printf("    ~ %s: %s\n", colDiff.ColumnName, colDiff.Diff)
			}
		}

		// Primary Key
		if tableDiff.PrimaryKeyDiff != nil {
			fmt.Printf("  Primary Key: %s\n", *tableDiff.PrimaryKeyDiff)
		}

		// Foreign Keys
		printConstraintDiffs("Foreign Keys", tableDiff.ForeignKeysOnlyInSource, tableDiff.ForeignKeysOnlyInTarget, tableDiff.ForeignKeyDiffs)

		// Unique Constraints
		printConstraintDiffs("Unique Constraints", tableDiff.UniquesOnlyInSource, tableDiff.UniquesOnlyInTarget, tableDiff.UniqueDiffs)

		// Indexes
		printConstraintDiffs("Indexes", tableDiff.IndexesOnlyInSource, tableDiff.IndexesOnlyInTarget, tableDiff.IndexDiffs)

		// Check Constraints
		printConstraintDiffs("Check Constraints", tableDiff.ChecksOnlyInSource, tableDiff.ChecksOnlyInTarget, tableDiff.CheckDiffs)
	}

	fmt.Println()
}

func printConstraintDiffs[T interface{ GetName() string; GetDiff() string }](
	label string,
	onlyInSource, onlyInTarget []string,
	diffs []T,
) {
	hasAny := len(onlyInSource) > 0 || len(onlyInTarget) > 0 || len(diffs) > 0
	if !hasAny {
		return
	}

	if len(onlyInSource) > 0 {
		fmt.Printf("  %s only in SOURCE:\n", label)
		for _, name := range onlyInSource {
			fmt.Printf("    - %s\n", name)
		}
	}

	if len(onlyInTarget) > 0 {
		fmt.Printf("  %s only in TARGET:\n", label)
		for _, name := range onlyInTarget {
			fmt.Printf("    + %s\n", name)
		}
	}

	if len(diffs) > 0 {
		fmt.Printf("  %s differences:\n", label)
		for _, d := range diffs {
			fmt.Printf("    ~ %s: %s\n", d.GetName(), d.GetDiff())
		}
	}
}

// Implement interface methods for diff types
func (d *FKDiff) GetName() string    { return d.Name }
func (d *FKDiff) GetDiff() string    { return d.Diff }
func (d *UniqueDiff) GetName() string { return d.Name }
func (d *UniqueDiff) GetDiff() string { return d.Diff }
func (d *IndexDiff) GetName() string  { return d.Name }
func (d *IndexDiff) GetDiff() string  { return d.Diff }
func (d *CheckDiff) GetName() string  { return d.Name }
func (d *CheckDiff) GetDiff() string  { return d.Diff }

// ============================================================================
// CLI & MAIN
// ============================================================================

func main() {
	// Connection flags
	sourceConn := flag.String("source", "", "Source database connection string")
	sourceDriver := flag.String("source-driver", "", "Source database driver (postgres or mysql)")
	targetConn := flag.String("target", "", "Target database connection string")
	targetDriver := flag.String("target-driver", "", "Target database driver (postgres or mysql)")

	// Output flags
	asJSON := flag.Bool("json", false, "Output as JSON")
	generateMigration := flag.Bool("migration", false, "Generate SQL migration script")

	// Performance flags
	parallel := flag.Bool("parallel", false, "Use parallel schema extraction (faster for large databases)")

	// Filter flags
	ignoreTables := flag.String("ignore-tables", "", "Comma-separated list of table names to ignore")
	ignoreTablePattern := flag.String("ignore-table-pattern", "", "Regex pattern for table names to ignore")
	ignoreIndexes := flag.Bool("ignore-indexes", false, "Ignore all index differences")
	ignoreForeignKeys := flag.Bool("ignore-foreign-keys", false, "Ignore all foreign key differences")
	ignoreChecks := flag.Bool("ignore-checks", false, "Ignore all check constraint differences")

	flag.Parse()

	// Validate flags
	if *sourceConn == "" || *sourceDriver == "" || *targetConn == "" || *targetDriver == "" {
		fmt.Fprintln(os.Stderr, "Usage: dbdiff --source <conn> --source-driver <driver> --target <conn> --target-driver <driver> [options]")
		fmt.Fprintln(os.Stderr, "\nRequired flags:")
		fmt.Fprintln(os.Stderr, "  --source <conn>          Source database connection string")
		fmt.Fprintln(os.Stderr, "  --source-driver <driver> Source database driver (postgres or mysql)")
		fmt.Fprintln(os.Stderr, "  --target <conn>          Target database connection string")
		fmt.Fprintln(os.Stderr, "  --target-driver <driver> Target database driver (postgres or mysql)")
		fmt.Fprintln(os.Stderr, "\nOutput options:")
		fmt.Fprintln(os.Stderr, "  --json                   Output as JSON")
		fmt.Fprintln(os.Stderr, "  --migration              Generate SQL migration script")
		fmt.Fprintln(os.Stderr, "\nPerformance options:")
		fmt.Fprintln(os.Stderr, "  --parallel               Use parallel schema extraction (faster for large databases)")
		fmt.Fprintln(os.Stderr, "\nFilter options:")
		fmt.Fprintln(os.Stderr, "  --ignore-tables <list>   Comma-separated list of table names to ignore")
		fmt.Fprintln(os.Stderr, "  --ignore-table-pattern <regex>  Regex pattern for table names to ignore")
		fmt.Fprintln(os.Stderr, "  --ignore-indexes         Ignore all index differences")
		fmt.Fprintln(os.Stderr, "  --ignore-foreign-keys    Ignore all foreign key differences")
		fmt.Fprintln(os.Stderr, "  --ignore-checks          Ignore all check constraint differences")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  Basic comparison:")
		fmt.Fprintln(os.Stderr, `    dbdiff --source "postgres://user:pass@localhost:5432/db1?sslmode=disable" --source-driver postgres \`)
		fmt.Fprintln(os.Stderr, `           --target "postgres://user:pass@localhost:5432/db2?sslmode=disable" --target-driver postgres`)
		fmt.Fprintln(os.Stderr, "\n  With filtering:")
		fmt.Fprintln(os.Stderr, `    dbdiff --source "..." --source-driver postgres --target "..." --target-driver postgres \`)
		fmt.Fprintln(os.Stderr, `           --ignore-tables "temp_table,old_table" --ignore-indexes`)
		fmt.Fprintln(os.Stderr, "\n  Generate migration:")
		fmt.Fprintln(os.Stderr, `    dbdiff --source "..." --source-driver postgres --target "..." --target-driver postgres --migration`)
		fmt.Fprintln(os.Stderr, "\n  Parallel extraction:")
		fmt.Fprintln(os.Stderr, `    dbdiff --source "..." --source-driver postgres --target "..." --target-driver postgres --parallel`)
		os.Exit(1)
	}

	// Build filter config
	filter := NewFilterConfig()
	if *ignoreTables != "" {
		filter.IgnoreTables = strings.Split(*ignoreTables, ",")
		// Trim whitespace
		for i := range filter.IgnoreTables {
			filter.IgnoreTables[i] = strings.TrimSpace(filter.IgnoreTables[i])
		}
	}
	if *ignoreTablePattern != "" {
		pattern, err := regexp.Compile(*ignoreTablePattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid table pattern regex: %v\n", err)
			os.Exit(1)
		}
		filter.IgnoreTablePattern = pattern
	}
	filter.IgnoreIndexes = *ignoreIndexes
	filter.IgnoreForeignKeys = *ignoreForeignKeys
	filter.IgnoreChecks = *ignoreChecks

	// Connect to source database
	sourceDB, err := sql.Open(*sourceDriver, *sourceConn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to source database: %v\n", err)
		os.Exit(1)
	}
	defer sourceDB.Close()

	if err := sourceDB.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Error pinging source database: %v\n", err)
		os.Exit(1)
	}

	// Connect to target database
	targetDB, err := sql.Open(*targetDriver, *targetConn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to target database: %v\n", err)
		os.Exit(1)
	}
	defer targetDB.Close()

	if err := targetDB.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Error pinging target database: %v\n", err)
		os.Exit(1)
	}

	// Get dialects
	sourceDialect := getDialect(*sourceDriver)
	targetDialect := getDialect(*targetDriver)

	if sourceDialect == nil {
		fmt.Fprintf(os.Stderr, "Unsupported source driver: %s\n", *sourceDriver)
		os.Exit(1)
	}

	if targetDialect == nil {
		fmt.Fprintf(os.Stderr, "Unsupported target driver: %s\n", *targetDriver)
		os.Exit(1)
	}

	// Extract schemas (with optional parallel extraction)
	var sourceSchema, targetSchema *Schema

	if *parallel {
		sourceSchema, err = sourceDialect.ExtractSchemaParallel(sourceDB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting source schema: %v\n", err)
			os.Exit(1)
		}

		targetSchema, err = targetDialect.ExtractSchemaParallel(targetDB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting target schema: %v\n", err)
			os.Exit(1)
		}
	} else {
		sourceSchema, err = sourceDialect.ExtractSchema(sourceDB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting source schema: %v\n", err)
			os.Exit(1)
		}

		targetSchema, err = targetDialect.ExtractSchema(targetDB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting target schema: %v\n", err)
			os.Exit(1)
		}
	}

	// Compute diff with filter
	diff := ComputeDiff(sourceSchema, targetSchema, filter)

	// Output based on flags
	if *generateMigration {
		// Generate and print migration SQL
		migrationSQL := GenerateMigrationSQL(diff, *sourceDriver)
		fmt.Print(migrationSQL)
	} else {
		// Print diff output
		PrintDiff(diff, *asJSON)
	}

	// Exit with appropriate code
	if isDiffEmpty(diff) {
		os.Exit(0)
	} else {
		os.Exit(2)
	}
}

func getDialect(driver string) Dialect {
	switch driver {
	case "postgres":
		return &PostgresDialect{}
	case "mysql":
		return &MySQLDialect{}
	default:
		return nil
	}
}
package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ragokan/limen"
)

type schemaIntrospector struct {
	db     *sql.DB
	driver Driver
}

func newSchemaIntrospector(db *sql.DB, driver Driver) *schemaIntrospector {
	return &schemaIntrospector{
		db:     db,
		driver: driver,
	}
}

func (s *schemaIntrospector) getTables(tableNames []string) (map[string]bool, error) {
	result := make(map[string]bool, len(tableNames))

	query, args := s.driver.TableExistsBatchQuery(tableNames)
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		result[tableName] = true
	}

	return result, rows.Err()
}

func (s *schemaIntrospector) introspectTable(tableName limen.SchemaTableName) (*limen.SchemaDefinition, error) {
	columns, err := s.introspectColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect columns: %w", err)
	}

	indexes, err := s.introspectIndexes(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect indexes: %w", err)
	}

	foreignKeys, err := s.introspectForeignKeys(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect foreign keys: %w", err)
	}

	return &limen.SchemaDefinition{
		TableName:   tableName,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: foreignKeys,
		SchemaName:  limen.SchemaName(tableName),
	}, nil
}

func (s *schemaIntrospector) introspectColumns(tableName limen.SchemaTableName) ([]limen.ColumnDefinition, error) {
	query, args := s.driver.IntrospectColumnsQuery(string(tableName))
	return introspectRows(s.db, query, args, s.driver.ParseColumnRow)
}

func (s *schemaIntrospector) introspectIndexes(tableName limen.SchemaTableName) ([]limen.IndexDefinition, error) {
	query, args := s.driver.IntrospectIndexesQuery(string(tableName))
	return introspectRows(s.db, query, args, s.driver.ParseIndexRow)
}

func (s *schemaIntrospector) introspectForeignKeys(tableName limen.SchemaTableName) ([]limen.ForeignKeyDefinition, error) {
	query, args := s.driver.IntrospectForeignKeysQuery(string(tableName))
	return introspectRows(s.db, query, args, s.driver.ParseForeignKeyRow)
}

func introspectRows[T any](db *sql.DB, query string, args []any, parse func(func(dest ...any) error) (T, error)) ([]T, error) {
	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []T
	for rows.Next() {
		value, err := parse(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}

	return out, rows.Err()
}

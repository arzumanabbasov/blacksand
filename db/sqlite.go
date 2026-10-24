package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

//go:embed schema_sqlite.sql
var sqliteSchema string

var sqliteUnderlyingDriver driver.Driver

func init() {
	registerSQLiteFunctions()
	registered, err := sql.Open("sqlite", "")
	if err != nil {
		panic(err)
	}
	sqliteUnderlyingDriver = registered.Driver()
	registered.Close()
	sql.Register("blacksand-sqlite", sqliteDialectDriver{})
}

func (d *DB) IsSQLite() bool { _, ok := d.Driver().(sqliteDialectDriver); return ok }

func openSQLite(path string) (*DB, error) {
	if path == "" {
		return nil, fmt.Errorf("SQLite path must not be empty")
	}
	if path != ":memory:" {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
	}
	name := filepath.ToSlash(path) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(15000)&_pragma=journal_mode(WAL)&_txlock=immediate"
	database, err := sql.Open("blacksand-sqlite", name)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(8)
	if path == ":memory:" {
		database.SetMaxOpenConns(1)
	}
	d := &DB{database}
	if _, err := database.Exec(sqliteSchema); err != nil {
		database.Close()
		return nil, fmt.Errorf("apply SQLite schema: %w", err)
	}
	if _, err := database.Exec("PRAGMA user_version=1"); err != nil {
		database.Close()
		return nil, err
	}
	if err := d.seedBuiltins(); err != nil {
		database.Close()
		return nil, fmt.Errorf("seed SQLite builtins: %w", err)
	}
	return d, nil
}

// The dialect is applied at the driver boundary so transactional and prepared
// statements use the same translation as DB queries. BEGIN IMMEDIATE reserves
// the SQLite writer before any read/check/write operation; this replaces PG's
// row/advisory locking on this single-file backend without a check/write race.
type sqliteDialectDriver struct{}

func (sqliteDialectDriver) Open(name string) (driver.Conn, error) {
	conn, err := sqliteUnderlyingDriver.Open(name)
	if err != nil {
		return nil, err
	}
	return &sqliteDialectConn{Conn: conn}, nil
}

type sqliteDialectConn struct{ driver.Conn }

func (c *sqliteDialectConn) CheckNamedValue(value *driver.NamedValue) error {
	if value.Value == nil {
		return nil
	}
	if text, ok := value.Value.(string); ok && strings.ContainsRune(text, 0) {
		return fmt.Errorf("database text cannot contain NUL bytes")
	}
	if timestamp, ok := value.Value.(time.Time); ok {
		value.Value = sqliteStamp(timestamp)
		return nil
	}
	if _, ok := value.Value.(json.RawMessage); ok {
		value.Value = string(value.Value.(json.RawMessage))
		return nil
	}
	rv := reflect.ValueOf(value.Value)
	if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() != reflect.Uint8 {
		return nil
	}
	converted, err := driver.DefaultParameterConverter.ConvertValue(value.Value)
	value.Value = converted
	return err
}
func sqliteArgs(query string, args []driver.NamedValue) ([]driver.NamedValue, error) {
	args = append([]driver.NamedValue(nil), args...)
	for i := range args {
		value := args[i].Value
		if value == nil {
			continue
		}
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() != reflect.Uint8 {
			data, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			args[i].Value = string(data)
			continue
		}
		cast := sqliteArrayCast(query, args[i].Ordinal)
		if cast == "" {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		var entries []string
		if err := pgtype.NewMap().Scan(pgtype.TextArrayOID, pgtype.TextFormatCode, []byte(text), &entries); err != nil {
			return nil, fmt.Errorf("SQLite array argument: %w", err)
		}
		var encoded []byte
		var err error
		switch cast {
		case "jsonb":
			items := make([]json.RawMessage, len(entries))
			for j, entry := range entries {
				items[j] = json.RawMessage(entry)
			}
			encoded, err = json.Marshal(items)
		case "bigint", "int", "integer":
			items := make([]json.Number, len(entries))
			for j, entry := range entries {
				items[j] = json.Number(entry)
			}
			encoded, err = json.Marshal(items)
		default:
			encoded, err = json.Marshal(entries)
		}
		if err != nil {
			return nil, err
		}
		args[i].Value = string(encoded)
	}
	return args, nil
}
func (c *sqliteDialectConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	normalized, err := sqliteArgs(query, args)
	if err != nil {
		return nil, err
	}
	sql, err := sqliteSQL(query)
	if err != nil {
		return nil, err
	}
	result, err := c.Conn.(driver.ExecerContext).ExecContext(ctx, sql, normalized)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate column name") && strings.Contains(strings.ToUpper(query), "ADD COLUMN IF NOT EXISTS") {
		return driver.RowsAffected(0), nil
	}
	return result, sqliteConstraintError(err)
}
func (c *sqliteDialectConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(query)), "select nextval(pg_get_serial_sequence('assets','id'))") {
		if _, err := c.Conn.(driver.ExecerContext).ExecContext(ctx, `INSERT INTO sqlite_sequence(name,seq) SELECT 'assets',0 WHERE NOT EXISTS(SELECT 1 FROM sqlite_sequence WHERE name='assets')`, nil); err != nil {
			return nil, err
		}
		return c.Conn.(driver.QueryerContext).QueryContext(ctx, `UPDATE sqlite_sequence SET seq=max(seq,(SELECT COALESCE(max(id),0) FROM assets))+1 WHERE name='assets' RETURNING seq`, nil)
	}
	normalized, err := sqliteArgs(query, args)
	if err != nil {
		return nil, err
	}
	sql, err := sqliteSQL(query)
	if err != nil {
		return nil, err
	}
	rows, err := c.Conn.(driver.QueryerContext).QueryContext(ctx, sql, normalized)
	if err != nil {
		return nil, fmt.Errorf("SQLite query: %w [SQL: %.700s]", sqliteConstraintError(err), sql)
	}
	return &sqliteDialectRows{Rows: rows}, nil
}
func (c *sqliteDialectConn) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, options)
}
func (c *sqliteDialectConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}
func (c *sqliteDialectConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	sql, err := sqliteSQL(query)
	if err != nil {
		return nil, err
	}
	statement, err := c.Conn.(driver.ConnPrepareContext).PrepareContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	return &sqliteDialectStmt{Stmt: statement, query: query}, nil
}

type sqliteDialectStmt struct {
	driver.Stmt
	query string
}

func sqliteConstraintError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for text, code := range map[string]string{"UNIQUE constraint failed": "23505", "FOREIGN KEY constraint failed": "23503", "CHECK constraint failed": "23514", "NOT NULL constraint failed": "23502"} {
		if strings.Contains(message, text) {
			return &pgconn.PgError{Code: code, Message: message}
		}
	}
	return err
}

func (s *sqliteDialectStmt) NumInput() int { return -1 }
func (s *sqliteDialectStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	normalized, err := sqliteArgs(s.query, args)
	if err != nil {
		return nil, err
	}
	return s.Stmt.(driver.StmtExecContext).ExecContext(ctx, normalized)
}
func (s *sqliteDialectStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	normalized, err := sqliteArgs(s.query, args)
	if err != nil {
		return nil, err
	}
	rows, err := s.Stmt.(driver.StmtQueryContext).QueryContext(ctx, normalized)
	if err != nil {
		return nil, err
	}
	return &sqliteDialectRows{Rows: rows}, nil
}

type sqliteDialectRows struct{ driver.Rows }

func (r *sqliteDialectRows) Next(dest []driver.Value) error {
	if err := r.Rows.Next(dest); err != nil {
		return err
	}
	columns := r.Columns()
	for i, value := range dest {
		text, ok := value.(string)
		if !ok {
			continue
		}
		name := strings.ToLower(columns[i])
		if name == "ts" || strings.HasSuffix(name, "_at") || strings.HasSuffix(name, "_until") || name == "last_used" || name == "last_fire" || strings.HasPrefix(name, "max(") || strings.HasPrefix(name, "min(") {
			if parsed, err := sqliteTime(text); err == nil {
				dest[i] = parsed
				continue
			}
		}
		if json.Valid([]byte(text)) {
			dest[i] = []byte(text)
		}
	}
	return nil
}
func (r *sqliteDialectRows) ColumnTypeDatabaseTypeName(index int) string {
	if source, ok := r.Rows.(driver.RowsColumnTypeDatabaseTypeName); ok {
		return source.ColumnTypeDatabaseTypeName(index)
	}
	return ""
}

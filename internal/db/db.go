package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"github.com/bcrowe306/linksy/migrations"
)

type DB struct {
	*sql.DB
	DriverName string
}

func Connect(dbType, dsn string) (*DB, error) {
	var driver string
	if dbType == "postgres" || strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		driver = "postgres"
	} else {
		driver = "sqlite"
		// Enforce foreign keys and WAL on every pooled connection
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	}

	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database (%s): %w", driver, err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database (%s): %w", driver, err)
	}

	if driver == "sqlite" {
		// Prevent SQLITE_BUSY when multiple goroutines write concurrently
		conn.SetMaxOpenConns(1)
	}

	database := &DB{
		DB:         conn,
		DriverName: driver,
	}

	if err := database.Migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return database, nil
}

// Rebind converts '?' placeholders into the driver's native placeholder style
// so queries can be written once for both SQLite and PostgreSQL.
func (db *DB) Rebind(query string) string {
	if db.DriverName != "postgres" {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 16)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// Migrate applies every embedded migration for the active dialect that has not
// yet been recorded in schema_migrations. Each migration runs in a transaction.
func (db *DB) Migrate() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("failed creating schema_migrations: %w", err)
	}

	dir := db.DriverName
	entries, err := fs.ReadDir(migrations.FS, dir)
	if err != nil {
		return fmt.Errorf("failed reading migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".sql")
		var exists int
		if err := db.QueryRow(db.Rebind(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`), version).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}

		body, err := fs.ReadFile(migrations.FS, dir+"/"+name)
		if err != nil {
			return err
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", name, err)
		}
		if _, err := tx.Exec(db.Rebind(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`), version, time.Now().UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

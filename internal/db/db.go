package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
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
	}

	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database (%s): %w", driver, err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database (%s): %w", driver, err)
	}

	database := &DB{
		DB:         conn,
		DriverName: driver,
	}

	if driver == "sqlite" {
		// Prevent SQLITE_BUSY when multiple goroutines write concurrently
		conn.SetMaxOpenConns(1)
		_, _ = conn.Exec("PRAGMA journal_mode=WAL;")
		_, _ = conn.Exec("PRAGMA busy_timeout=5000;")
	}

	if err := database.Migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return database, nil
}

func (db *DB) Migrate() error {
	var queries []string

	if db.DriverName == "postgres" {
		queries = []string{
			`CREATE TABLE IF NOT EXISTS users (
				id VARCHAR(64) PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				first_name VARCHAR(100),
				last_name VARCHAR(100),
				avatar_url TEXT,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS teams (
				id VARCHAR(64) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				slug VARCHAR(255) UNIQUE NOT NULL,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS team_members (
				id VARCHAR(64) PRIMARY KEY,
				team_id VARCHAR(64) NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
				user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				role VARCHAR(32) NOT NULL,
				created_at TIMESTAMPTZ NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS links (
				id VARCHAR(64) PRIMARY KEY,
				team_id VARCHAR(64) NOT NULL,
				user_id VARCHAR(64) NOT NULL,
				slug VARCHAR(128) UNIQUE NOT NULL,
				target_url TEXT NOT NULL,
				title VARCHAR(255),
				description TEXT,
				is_active BOOLEAN DEFAULT TRUE,
				expires_at TIMESTAMPTZ,
				password VARCHAR(255),
				click_count BIGINT DEFAULT 0,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS qr_codes (
				id VARCHAR(64) PRIMARY KEY,
				team_id VARCHAR(64) NOT NULL,
				user_id VARCHAR(64) NOT NULL,
				link_id VARCHAR(64),
				type VARCHAR(32) NOT NULL,
				title VARCHAR(255),
				payload TEXT NOT NULL,
				raw_data TEXT,
				style TEXT NOT NULL,
				scan_count BIGINT DEFAULT 0,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS documents (
				id VARCHAR(64) PRIMARY KEY,
				team_id VARCHAR(64) NOT NULL,
				user_id VARCHAR(64) NOT NULL,
				filename VARCHAR(255) NOT NULL,
				stored_path VARCHAR(255) NOT NULL,
				mime_type VARCHAR(128) NOT NULL,
				file_size BIGINT NOT NULL,
				version INT DEFAULT 1,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS analytics_events (
				id VARCHAR(64) PRIMARY KEY,
				link_id VARCHAR(64),
				qr_code_id VARCHAR(64),
				event_type VARCHAR(32) NOT NULL,
				ip VARCHAR(64),
				user_agent TEXT,
				browser VARCHAR(64),
				os VARCHAR(64),
				device VARCHAR(64),
				referrer TEXT,
				country VARCHAR(64),
				city VARCHAR(64),
				created_at TIMESTAMPTZ NOT NULL
			);`,
			`CREATE INDEX IF NOT EXISTS idx_links_slug ON links(slug);`,
			`CREATE INDEX IF NOT EXISTS idx_analytics_link_id ON analytics_events(link_id);`,
			`CREATE INDEX IF NOT EXISTS idx_analytics_qr_id ON analytics_events(qr_code_id);`,
			`CREATE INDEX IF NOT EXISTS idx_analytics_created_at ON analytics_events(created_at);`,
		}
	} else {
		// SQLite compatible schema
		queries = []string{
			`CREATE TABLE IF NOT EXISTS users (
				id TEXT PRIMARY KEY,
				email TEXT UNIQUE NOT NULL,
				first_name TEXT,
				last_name TEXT,
				avatar_url TEXT,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS teams (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				slug TEXT UNIQUE NOT NULL,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS team_members (
				id TEXT PRIMARY KEY,
				team_id TEXT NOT NULL,
				user_id TEXT NOT NULL,
				role TEXT NOT NULL,
				created_at DATETIME NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS links (
				id TEXT PRIMARY KEY,
				team_id TEXT NOT NULL,
				user_id TEXT NOT NULL,
				slug TEXT UNIQUE NOT NULL,
				target_url TEXT NOT NULL,
				title TEXT,
				description TEXT,
				is_active INTEGER DEFAULT 1,
				expires_at DATETIME,
				password TEXT,
				click_count INTEGER DEFAULT 0,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS qr_codes (
				id TEXT PRIMARY KEY,
				team_id TEXT NOT NULL,
				user_id TEXT NOT NULL,
				link_id TEXT,
				type TEXT NOT NULL,
				title TEXT,
				payload TEXT NOT NULL,
				raw_data TEXT,
				style TEXT NOT NULL,
				scan_count INTEGER DEFAULT 0,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS documents (
				id TEXT PRIMARY KEY,
				team_id TEXT NOT NULL,
				user_id TEXT NOT NULL,
				filename TEXT NOT NULL,
				stored_path TEXT NOT NULL,
				mime_type TEXT NOT NULL,
				file_size INTEGER NOT NULL,
				version INTEGER DEFAULT 1,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			);`,
			`CREATE TABLE IF NOT EXISTS analytics_events (
				id TEXT PRIMARY KEY,
				link_id TEXT,
				qr_code_id TEXT,
				event_type TEXT NOT NULL,
				ip TEXT,
				user_agent TEXT,
				browser TEXT,
				os TEXT,
				device TEXT,
				referrer TEXT,
				country TEXT,
				city TEXT,
				created_at DATETIME NOT NULL
			);`,
			`CREATE INDEX IF NOT EXISTS idx_links_slug ON links(slug);`,
			`CREATE INDEX IF NOT EXISTS idx_analytics_link_id ON analytics_events(link_id);`,
			`CREATE INDEX IF NOT EXISTS idx_analytics_qr_id ON analytics_events(qr_code_id);`,
			`CREATE INDEX IF NOT EXISTS idx_analytics_created_at ON analytics_events(created_at);`,
		}
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed executing migration statement: %w\nQuery: %s", err, query)
		}
	}

	return nil
}

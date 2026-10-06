package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite" 
)

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func Migrate(db *sql.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		query, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(query)); err != nil {
			return fmt.Errorf("миграция %s: %w", f, err)
		}
	}
	return nil
}
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewConnection(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS articles (
			id BIGSERIAL PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES auth.users(id),
			url TEXT NOT NULL,
			title TEXT,
			format TEXT NOT NULL CHECK (format IN ('text', 'audio', 'video')),
			length TEXT NOT NULL CHECK (length IN ('s', 'm', 'l')),
			status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'ready', 'failed')),
			thumbnail_path TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			language TEXT,
			style TEXT,
			original_content TEXT,
			summary TEXT,
			text_body TEXT,
			audio_file_path TEXT,
			video_file_path TEXT,
			duration_seconds INTEGER,
			error_message TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
		CREATE INDEX IF NOT EXISTS idx_articles_user_id ON articles(user_id);
		CREATE INDEX IF NOT EXISTS idx_articles_format ON articles(format);
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}

	return nil
}

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
	// Create a sample users table
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS articles (
			id SERIAL PRIMARY KEY,
			url TEXT NOT NULL,
			format VARCHAR(10) NOT NULL CHECK (format IN ('text', 'audio')),
			length VARCHAR(10) NOT NULL CHECK (length IN ('s', 'm', 'l')),
			language VARCHAR(50),
			style VARCHAR(100),
			status VARCHAR(20) NOT NULL DEFAULT 'init' CHECK (status IN ('init', 'processing', 'available', 'failed')),
			original_content TEXT,
			summary TEXT,
			audio_file_path TEXT,
			error_message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
		CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}

	return nil
}

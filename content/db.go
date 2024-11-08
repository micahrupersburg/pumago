package content

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for database/sql
	"log"
	"path/filepath"
	"pumago/config"
)

type DB struct {
	*sql.DB
}

func DefaultDB() DB {
	dbFile := filepath.Join(config.Dir(), "db.sqlite")
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	out := DB{db}
	err = out.createContentTable()
	if err != nil {
		log.Fatalf("Failed to create content table: %v", err)
	}
	return out
}
func (c *Content) PrimaryKey() string {
	return c.ID
}
func (db *DB) createContentTable() error {
	query := `
    CREATE TABLE IF NOT EXISTS file_entries (
        id TEXT,
        url TEXT,
        title TEXT,
        last_modified_millis INTEGER,
        fragment INTEGER,
        origin INTEGER,
        content TEXT,
        status INTEGER,
        PRIMARY KEY (id, origin)
    );`
	_, err := db.Exec(query)
	return err
}

func (db *DB) insertContent(entry Content) error {
	query := `
    INSERT INTO file_entries (id, url, title, last_modified_millis, fragment, origin, content, status)
    VALUES (?, ?, ?, ?, ?, ? , ?, ?);`
	_, err := db.Exec(query, entry.ID, entry.URL, entry.Title, entry.LastModifiedMillis, entry.Fragment, entry.Origin, entry.Content, entry.Status)
	return err
}

func (db *DB) getContentByID(origin Origin, id string) (Content, error) {
	query := `SELECT id, url, title, last_modified_millis, fragment, origin, content, status FROM file_entries WHERE id = ? and origin = ?;`
	row := db.QueryRow(query, id, origin)

	var entry Content
	err := row.Scan(&entry.ID, &entry.URL, &entry.Title, &entry.LastModifiedMillis, &entry.Fragment, &entry.Origin, &entry.Content, &entry.Status)
	if err != nil {
		return entry, err
	}
	return entry, nil
}

func (db *DB) updateContentStatus(entry Content) error {
	query := `
    UPDATE file_entries
    SET status = ?
    WHERE id = ? AND origin = ?;`
	_, err := db.Exec(query, entry.Status, entry.ID, entry.Origin)
	return err
}

func (db *DB) deleteContent(origin Origin, id string) error {
	query := `DELETE FROM file_entries WHERE id = ? and origin = ? ;`
	_, err := db.Exec(query, id, origin)
	return err
}

// Content Manager Implementation
func (db *DB) Add(content Content) error {
	return db.insertContent(content)
}

func (db *DB) Processed(content Content) error {
	content.Status = PROCESSED
	return db.updateContentStatus(content)
}

func (db *DB) Failed(content Content) error {
	content.Status = FAILED
	return db.updateContentStatus(content)
}

func (db *DB) Get(origin Origin, id string) (Content, error) {
	return db.getContentByID(origin, id)
}

func (db *DB) List(origin Origin, status Status) ([]Content, error) {
	query := `SELECT id, url, title, last_modified_millis, fragment, origin, content, status FROM file_entries WHERE status = ? and origin = ?;`
	rows, err := db.Query(query, status, origin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []Content
	for rows.Next() {
		var content Content
		if err := rows.Scan(&content.ID, &content.URL, &content.Title, &content.LastModifiedMillis, &content.Fragment, &content.Origin, &content.Content, &content.Status); err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return contents, nil
}

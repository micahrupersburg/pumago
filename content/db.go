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

	err = out.creatStateTable()
	if err != nil {
		log.Fatalf("Failed to create state table: %v", err)
	}
	return out
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
func (db *DB) UpdateAll(status Status) error {
	query := `UPDATE file_entries SET status = ?;`
	_, err := db.Exec(query, status)
	return err
}

func (db *DB) All(status Status) ([]Content, error) {
	query := `SELECT id, url, title, last_modified_millis, fragment, origin, content, status FROM file_entries WHERE status = ?;`
	rows, err := db.Query(query, status)
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

func (db *DB) DeleteState(key string) {
	query := `DELETE FROM file_entries WHERE key = ?;`
	_, err := db.Exec(query, key)
	if err != nil {
		log.Printf("Failed to delete state: %v", err)
	}
}
func (db *DB) GetState(space string, key string) (string, error) {
	query := `SELECT value FROM states WHERE space =? key = ?;`
	row := db.QueryRow(query, space, key)
	var value string
	err := row.Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}
func (db *DB) SetState(space string, key string, value string) error {
	query := `INSERT OR REPLACE INTO states (space, key, value) VALUES (?, ?, ?);`
	_, err := db.Exec(query, space, key, value)
	if err != nil {
		log.Printf("Failed to set state: %v", err)
	}
	return err
}

func (db *DB) creatStateTable() error {
	query := `
    CREATE TABLE IF NOT EXISTS states (
        space TEXT,
        key TEXT,
        value TEXT,
  		PRIMARY KEY (space, key)
    );`
	_, err := db.Exec(query)
	return err
}
func (db *DB) SaveSettings(space string, all map[string]string) {
	for key, value := range all {
		db.SetState(space, key, value) //todo find a way to clobber old values
	}
}
func (db *DB) LoadSettings(space string) (map[string]string, error) {
	query := `SELECT key, value FROM states where space =?;`
	rows, err := db.Query(query, space)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

package content

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for database/sql
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Browser struct {
	historyPath string
	//maxHistory  int
	query    string
	origin   Origin
	lastread int64
}

func (c *Browser) Origin() Origin {
	return c.origin
}

// CopyHistoryToTemp copies the history database to a temporary file.
func (c *Browser) CopyHistoryToTemp() (string, error) {
	tmpHistory := filepath.Join(os.TempDir(), fmt.Sprintf("History%d", rand.Int63()))
	if err := copyFile(c.historyPath, tmpHistory); err != nil {
		return "", err
	}
	return tmpHistory, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

type HistoryItem struct {
	title         string
	url           string
	lastVisitTime int64
}

var skipPrefixes = []string{
	"http://localhost",
	"https://www.google.com/search",
}

// FetchHistory loads the browser history from the database.
func (c *Browser) doHistoryQuery() ([]HistoryItem, error) {
	// Copy history file to a temporary location.
	tmpHistory, err := c.CopyHistoryToTemp()

	if err != nil {
		return nil, fmt.Errorf("failed to copy history file: %w", err)
	}

	defer os.Remove(tmpHistory) // Ensure cleanup of the temporary file

	db, err := sql.Open("sqlite3", tmpHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	sqlQuery := strings.ReplaceAll(c.query, "?", fmt.Sprintf("%d", c.lastread))
	//log.Printf("Using query:", sqlQuery)

	rows, err := db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var entries = make([]HistoryItem, 0)
	for rows.Next() {

		item := HistoryItem{}
		if err := rows.Scan(&item.title, &item.url, &item.lastVisitTime); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if item.lastVisitTime <= c.lastread {
			continue
		}
		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(item.url, prefix) {
				skip = true
				break
			}
		}
		if !skip {
			entries = append(entries, item)
		}
	}
	return entries, nil
}
func (c *Browser) FetchContent() ([]Content, error) {
	history, err := c.doHistoryQuery()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}
	entries := make([]Content, 0)
	for _, item := range history {
		log.Printf("Downloading content %s", item.url)
		data, err := c.download(item.url)
		if err != nil {
			log.Printf("Failed to download content %s: %v", item.url, err)
			return nil, fmt.Errorf("failed to download content %s: %w", item.url, err)
		}
		entry := Content{
			ID:                 item.url,
			URL:                item.url,
			Title:              item.title,
			LastModifiedMillis: item.lastVisitTime,
			Origin:             c.origin,
			Content:            data,
		}
		entries = append(entries, entry)
	}

	fmt.Println("Done loading links from history.")
	c.lastread = time.Now().Unix()
	return entries, nil
}

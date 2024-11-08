package content

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	uuid "github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for database/sql
	pdf "github.com/pdfcpu/pdfcpu/pkg/api"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Browser struct {
	historyPath string
	maxHistory  int
	query       string
	origin      Origin
	lastread    int64
}

func ChromeBrowser() *Browser {
	historyPath := filepath.Join(os.Getenv("HOME"), "Library/Application Support/Google/Chrome/Default/History")
	return &Browser{
		historyPath: historyPath,
		maxHistory:  100,
		query:       "SELECT title, url, last_visit_time  where last_visit_time > ? FROM \"urls\" order by last_visit_time desc",
		origin:      CHROME,
	}
}

func SafariBrowser() *Browser {
	historyPath := filepath.Join(os.Getenv("HOME"), "Library/Safari/History.db")
	return &Browser{
		historyPath: historyPath,
		maxHistory:  100,
		query:       "select history_visits.title, history_items.url,  history_visits.visit_time from history_items inner join history_visits on history_items.id = history_visits.history_item  where history_visits.visit_time  > ? order by visit_time desc",
		origin:      SAFARI,
	}
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

// FetchHistory loads the browser history from the database.
func (c *Browser) FetchContent() ([]Content, error) {
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

	sqlQuery := fmt.Sprintf("%s LIMIT %d", c.query, c.maxHistory)
	sqlQuery = strings.ReplaceAll(sqlQuery, "?", fmt.Sprintf("%d", c.lastread))
	fmt.Println("Using query:", sqlQuery)

	rows, err := db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var entries []Content
	for rows.Next() {
		var title, url string
		var lastVisitTime int64

		if err := rows.Scan(&title, &url, &lastVisitTime); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if lastVisitTime <= c.lastread {
			continue
		}
		// Skip Google search URLs
		if len(url) > 24 && url[:24] == "https://www.google.com/search" {
			continue
		}
		data, err := c.download(url)
		if err != nil {
			return nil, err
		}
		entry := Content{
			ID:                 url,
			URL:                url,
			Title:              title,
			LastModifiedMillis: lastVisitTime,
			Origin:             c.origin,
			Content:            data,
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	fmt.Println("Done loading links from history.")
	c.lastread = time.Now().Unix()
	return entries, nil
}

func (c *Browser) download(url string) (string, error) {
	// Set up a context with timeout for cleanup.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a headless browser instance.
	ctx, cancelBrowser := chromedp.NewContext(ctx)
	defer cancelBrowser()

	// Capture the HTML or PDF.

	var buf []byte

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),                         // Navigate to the URL.
		chromedp.WaitVisible(`body`, chromedp.ByQuery), // Wait until body is visible.

		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().
				WithDisplayHeaderFooter(false).
				WithLandscape(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return "", err
	}
	filename := ConvertURLToFilename(url) + ".pdf"
	err = os.WriteFile(filepath.Join(os.TempDir(), filename), buf, 0644)
	if err != nil {
		return "", err
	}

	outputDir := filepath.Join(os.TempDir(), ConvertURLToFilename(url), uuid.New().String())
	pdf.ExtractContentFile(filename, outputDir, nil, nil)
	contents, err := readTextFiles(outputDir)
	if err != nil {
		return "", err
	}
	return strings.Join(contents, " "), nil
}
func readTextFiles(outputDir string) ([]string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	output := make([]string, 0)
	for _, entry := range entries {

		if !entry.IsDir() {
			if filepath.Ext(entry.Name()) == ".txt" {
				data, err := os.ReadFile(filepath.Join(outputDir, entry.Name()))
				if err != nil {
					return nil, fmt.Errorf("failed to read file: %w", err)
				}
				output = append(output, string(data))
			}
		}
	}

	return output, nil
}

package content

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	pdf "github.com/pdfcpu/pdfcpu/pkg/api"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (c *Browser) download(url string) (string, error) {
	// Set up a context with timeout for cleanup.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a headless browser instance.
	ctx, cancelBrowser := chromedp.NewContext(ctx)
	defer cancelBrowser()

	// Capture the HTML or PDF.

	var buf []byte

	chromedp.Flag("headless", true)
	chromedp.Flag("disable-history", true)
	chromedp.Flag("no-startup-window", true)
	chromedp.Flag("disable-gpu", true)
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),                         // Navigate to the URL.
		chromedp.WaitVisible(`body`, chromedp.ByQuery), // Wait until body is visible.

		chromedp.ActionFunc(func(ctx context.Context) error {
			//time.Sleep(1 * time.Second) // Wait for a second.
			var err error
			buf, _, err = page.PrintToPDF().
				WithDisplayHeaderFooter(false).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return "", fmt.Errorf("failed to capture PDF: %w", err)
	}
	return ExtractPdfContent(buf)
}
func ExtractPdfContent(data []byte) (string, error) {
	outputDir := filepath.Join(os.TempDir(), uuid.New().String())
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return "", err
	}
	filename := filepath.Join(outputDir, uuid.New().String()+".pdf")
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write PDF file: %w", err)
	}

	err = pdf.ExtractContentFile(filename, outputDir, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to extract PDF content from %s to %s: %w", filename, outputDir, err)
	}
	contents, err := readTextFiles(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to read text files: %w", err)
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
